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
	"fmt"
	"os"
	pd "p86l/internal/debug"
	"p86l/internal/file"
	"runtime"
	"strings"

	"github.com/hajimehoshi/guigui"
	"github.com/hashicorp/go-version"
	"github.com/pkg/browser"
	"golang.org/x/text/language"
)

func LoadB(am *AppModel, context *guigui.Context, model *Model, loadType string) *pd.Error {
	dm := am.Debug()
	log := dm.Log()
	fs := am.FileSystem()

	switch loadType {
	case "data":
		if err := fs.IsDirR(fs.FileDataPath()); err != nil {
			log.Info().Str("Data", "data not found, creating data...").Str("utils", "loadB").Msg(pd.FileManager)
			d := NewData()
			d.Log(dm)
			model.data.file = d
			return model.data.Save(am)
		}
	case "cache":
		if err := fs.IsDirR(fs.FileCachePath()); err != nil {
			log.Info().Str("Cache", "cache not found").Str("utils", "loadB").Msg(pd.FileManager)
			return nil
		}
	}

	switch loadType {
	case "data":
		d, err := LoadData(am)
		if err != nil {
			return err
		}

		tag, rErr := language.Parse(d.Locale)
		if rErr != nil {
			return pd.New(rErr, pd.DataError, pd.ErrDataLocaleInvalid)
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
		c, err := LoadCache(am)
		if err != nil {
			return err
		}
		if err := c.Validate(dm); err == nil {
			model.cache.valid = true
		}
		model.cache.file = *c
	}
	return nil
}

func OpenBrowser(dm *pd.Debug, url string) {
	dm.Log().Info().Str("Url", url).Msg("OpenBrowser")
	if err := browser.OpenURL(url); err != nil {
		dm.SetPopup(pd.New(err, pd.AppError, pd.ErrBrowserOpen))
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

func CheckNewerVersion(currentVersion, newVersion string) (bool, *pd.Error) {
	current, err := version.NewVersion(currentVersion)
	if err != nil {
		return false, pd.New(fmt.Errorf("invalid current version: %w", err), pd.AppError, pd.ErrGameVersionInvalid)
	}

	newer, err := version.NewVersion(newVersion)
	if err != nil {
		return false, pd.New(fmt.Errorf("invalid new version: %w", err), pd.AppError, pd.ErrGameVersionInvalid)
	}

	return newer.GreaterThan(current), nil
}

// -- Funcs for loading and saving --

func LoadData(am *AppModel) (*file.Data, *pd.Error) {
	dm := am.Debug()
	fs := am.FileSystem()

	b, err := fs.Load(fs.FileDataPath())
	if err != nil {
		return nil, err
	}

	d, err := fs.DecodeData(dm, b)
	if err != nil {
		return nil, err
	}

	return &d, nil
}

func LoadCache(am *AppModel) (*file.Cache, *pd.Error) {
	dm := am.Debug()
	fs := am.FileSystem()

	b, err := fs.Load(fs.FileCachePath())
	if err != nil {
		return nil, err
	}

	c, err := fs.DecodeCache(dm, b)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

func SaveData(am *AppModel, d file.Data) *pd.Error {
	dm := am.Debug()
	fs := am.FileSystem()

	b, err := fs.EncodeData(dm, d)
	if err != nil {
		return err
	}

	err = fs.Save(fs.FileDataPath(), b)
	if err != nil {
		return err
	}

	return nil
}

func SaveCache(am *AppModel, c file.Cache) *pd.Error {
	dm := am.Debug()
	fs := am.FileSystem()

	b, err := fs.EncodeCache(dm, c)
	if err != nil {
		return err
	}

	err = fs.Save(fs.FileCachePath(), b)
	if err != nil {
		return err
	}

	return nil
}
