/*
 * SPDX-License-Identifier: GPL-3.0-only
 * SPDX-FileCopyrightText: 2025 Project 86 Community
 *
 * Project-86-Launcher: A Launcher developed for Project-86 for managing game files.
 * Copyright (C) 2025 Project 86 Community
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package p86l

import (
	"context"
	"fmt"
	"os"
	"p86l/configs"
	pd "p86l/internal/debug"
	"p86l/internal/file"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v71/github"
	"github.com/hajimehoshi/guigui"
	"github.com/hashicorp/go-version"
	"github.com/pkg/browser"
	"golang.org/x/text/language"
)

type logFileInfo struct {
	name      string
	timestamp int64
	modTime   time.Time
}

func NewLogFile(root *os.Root, path string) (*os.File, error) {
	timestamp := time.Now().UTC().Unix()
	filename := fmt.Sprintf("log-%d.txt", timestamp)

	file, err := root.Create(filepath.Join(path, filename))
	if err != nil {
		return nil, fmt.Errorf("Failed to create log file: %w", err)
	}

	return file, nil
}

func RotateLogFiles(root *os.Root, pathDirCompany, pathDirLogs string) error {
	logDir := filepath.Join(pathDirCompany, pathDirLogs)

	entries, err := os.ReadDir(logDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("Failed to read log directory: %w", err)
	}

	var logFiles []logFileInfo
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), "log-") && strings.HasSuffix(entry.Name(), ".txt") {
			tsStr := strings.TrimPrefix(entry.Name(), "log-")
			tsStr = strings.TrimSuffix(tsStr, ".txt")

			ts, err := strconv.ParseInt(tsStr, 10, 64)
			if err != nil {
				continue // Skip files with invalid timestamps.
			}

			info, err := entry.Info()
			if err != nil {
				continue // Skip files we can't get info for.
			}

			logFiles = append(logFiles, logFileInfo{
				name:      entry.Name(),
				timestamp: ts,
				modTime:   info.ModTime(),
			})
		}
	}

	maxKeep := 10

	// If we have less than 10 files, no need to rotate.
	if len(logFiles) <= maxKeep {
		return nil
	}

	sort.Slice(logFiles, func(i, j int) bool {
		return logFiles[i].timestamp < logFiles[j].timestamp
	})

	filesToDelete := len(logFiles) - maxKeep
	for i := range filesToDelete {
		filePath := filepath.Join(pathDirLogs, logFiles[i].name)
		err := root.Remove(filePath)
		if err != nil {
			return fmt.Errorf("Failed to delete old log file %s: %w", filePath, err)
		}
	}

	return nil
}

func LoadB(am *AppModel, context *guigui.Context, model *Model, loadType string) pd.Result {
	dm := am.Debug()
	log := dm.Log()
	fs := am.FileSystem()

	switch loadType {
	case "data":
		if result := fs.ExistsRoot(fs.PathFileData()); !result.Ok {
			log.Info().Str("Data", "data not found, creating data...").Str("utils", "loadB").Msg(pd.FileManager)
			d := NewData()
			d.Log(dm)
			model.data.file = d
			result := model.data.Save(am)
			if !result.Ok {
				return result
			}
			return pd.Ok()
		}
	case "cache":
		if result := fs.ExistsRoot(fs.PathFileCache()); !result.Ok {
			log.Info().Str("Cache", "cache not found").Str("utils", "loadB").Msg(pd.FileManager)
			return pd.Ok()
		}
	}

	switch loadType {
	case "data":
		result, d := LoadData(am)
		if !result.Ok {
			return result
		}

		tag, rErr := language.Parse(d.Locale)
		if rErr != nil {
			return pd.NotOk(pd.New(rErr, pd.DataError, pd.ErrDataLocaleInvalid))
		}
		model.data.SetPosition(d.WindowX, d.WindowY)
		model.data.SetSize(d.WindowWidth, d.WindowHeight)
		model.data.File().WindowMaximize = d.WindowMaximize
		model.data.SetLocale(am, context, tag)
		model.data.SetAppScale(dm, context, d.AppScale)
		model.data.SetColorMode(dm, context, d.ColorMode)
		model.data.SetUsePreRelease(dm, d.UsePreRelease)
		return model.data.SetGameVersion(dm, d.GameVersion)
	case "cache":
		result, c := LoadCache(am)
		if !result.Ok {
			return result
		}
		if result := c.Validate(c.Repo); result.Ok {
			model.cache.valid = true
		}
		model.cache.file = *c
	}
	return pd.Ok()
}

func OpenBrowser(dm *pd.Debug, url string) {
	dm.Log().Info().Str("Url", url).Msg("OpenBrowser")
	if err := browser.OpenURL(url); err != nil {
		dm.SetPopup(pd.New(err, pd.AppError, pd.ErrBrowserOpen), pd.FileManager)
	}
}

func GetUsername() string {
	var username string
	switch runtime.GOOS {
	case "windows":
		username = os.Getenv("USERNAME")
	default:
		username = os.Getenv("USER")
	}

	if username == "" {
		username = os.Getenv("LOGNAME")
	}
	return strings.TrimSpace(username)
}

func IsValidPreGameFile(filename string) bool {
	return strings.Contains(filename, "Project86-v") &&
		strings.Contains(filename, ".zip")
}

func IsValidGameFile(filename string) bool {
	return strings.Contains(filename, "Project86-v") &&
		strings.Contains(filename, ".zip") &&
		!strings.Contains(filename, "dev")
}

func CheckNewerVersion(currentVersion, newVersion string) (pd.Result, bool) {
	current, err := version.NewVersion(currentVersion)
	if err != nil {
		return pd.NotOk(pd.New(fmt.Errorf("invalid current version: %w", err), pd.AppError, pd.ErrGameVersionInvalid)), false
	}

	newer, err := version.NewVersion(newVersion)
	if err != nil {
		return pd.NotOk(pd.New(fmt.Errorf("invalid new version: %w", err), pd.AppError, pd.ErrGameVersionInvalid)), false
	}

	return pd.Ok(), newer.GreaterThan(current)
}

func GetPreRelease(am *AppModel) (pd.Result, *github.RepositoryRelease) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	opt := &github.ListOptions{
		PerPage: 100,
	}

	maxPages := 5 // Limit to 500 releases (5 pages)
	pageCount := 0

	for {
		rs, resp, err := am.GithubClient().Repositories.ListReleases(ctx, configs.RepoOwner, configs.RepoName, opt)
		if err != nil {
			return pd.NotOk(pd.New(fmt.Errorf("failed to get prerelease: %w", err), pd.NetworkError, pd.ErrNetworkPrereleaseRequest)), nil
		}

		for _, r := range rs {
			if r.Prerelease != nil && *r.Prerelease {
				return pd.Ok(), r
			}
		}

		if resp.NextPage == 0 || pageCount >= maxPages {
			break
		}

		opt.Page = resp.NextPage
		pageCount++
	}

	return pd.NotOk(pd.New(fmt.Errorf("no pre-releases found"), pd.NetworkError, pd.ErrGitHubNoPreRelease)), nil
}

// -- Funcs for loading and saving --

func LoadData(am *AppModel) (pd.Result, *file.Data) {
	dm := am.Debug()
	fs := am.FileSystem()

	result, b := fs.Load(dm, fs.PathFileData())
	if !result.Ok {
		return result, nil
	}
	result, d := fs.DecodeData(dm, b)
	if !result.Ok {
		return result, nil
	}

	return pd.Ok(), &d
}

func LoadCache(am *AppModel) (pd.Result, *file.Cache) {
	dm := am.Debug()
	fs := am.FileSystem()

	result, b := fs.Load(dm, fs.PathFileCache())
	if !result.Ok {
		return result, nil
	}
	result, c := fs.DecodeCache(dm, b)
	if !result.Ok {
		return result, nil
	}

	return pd.Ok(), &c
}

func SaveData(am *AppModel, d file.Data) pd.Result {
	dm := am.Debug()
	fs := am.FileSystem()

	result, b := fs.EncodeData(dm, d)
	if !result.Ok {
		return result
	}
	result = fs.Save(dm, fs.PathFileData(), b)
	if !result.Ok {
		return result
	}

	return pd.Ok()
}

func SaveCache(am *AppModel, c file.Cache) pd.Result {
	dm := am.Debug()
	fs := am.FileSystem()
	// TODO: Prehaps set file version via env?
	c.V = 0

	result, b := fs.EncodeCache(dm, c)
	if !result.Ok {
		return result
	}
	result = fs.Save(dm, fs.PathFileCache(), b)
	if !result.Ok {
		return result
	}

	return pd.Ok()
}
