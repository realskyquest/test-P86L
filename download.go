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
	"archive/zip"
	gctx "context"
	"fmt"
	"io"
	"net/http"
	"os"
	"p86l/configs"
	pd "p86l/internal/debug"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v71/github"
)

func GetPreRelease(am *AppModel) (*github.RepositoryRelease, error) {
	ctx := gctx.Background()
	opt := &github.ListOptions{
		PerPage: 100,
	}

	for {
		rs, resp, err := am.GithubClient().Repositories.ListReleases(ctx, configs.RepoOwner, configs.RepoName, opt)
		if err != nil {
			return nil, err
		}

		for _, r := range rs {
			if r.Prerelease != nil && *r.Prerelease {
				return r, nil
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return nil, fmt.Errorf("no pre-releases found")
}

func DownloadGame(model *Model, filename, src string, preRelease bool) pd.Result {
	am := model.App()
	fs := am.FileSystem()
	gameType := "game"
	if preRelease {
		gameType = "pregame"
	}

	model.SetProgress("Downloading...")
	zipName := fmt.Sprintf("%s.zip", gameType)
	result := DownloadFile(model, filename, src, zipName)
	if !result.Ok {
		model.SetProgress("")
		return result
	}

	model.SetProgress("Removing old files...")
	gameDir := filepath.Join(fs.CompanyDirPath, "build", gameType)
	if err := os.RemoveAll(gameDir); err != nil {
		model.SetProgress("")
		return pd.NotOk(pd.New(err, pd.FSError, pd.ErrFSDirRemove))
	}

	model.SetProgress("Extracting...")
	srcPath := filepath.Join(fs.CompanyDirPath, zipName)
	if result := unzipToDir(model, srcPath, gameDir); !result.Ok {
		model.SetProgress("")
		return result
	}

	model.SetProgress("Cleaning...")
	if err := fs.Root.Remove(zipName); err != nil {
		model.SetProgress("")
		return pd.NotOk(pd.New(err, pd.FSError, pd.ErrFSRootFileRemove))
	}

	model.SetProgress("")
	return pd.Ok()
}

func DownloadFile(model *Model, filename, src, dest string) pd.Result {
	am := model.App()
	fs := am.FileSystem()
	dm := am.Debug()

	// Try to get existing file size for resume
	resumePos := int64(0)
	if info, err := fs.Root.Stat(dest); err == nil {
		resumePos = info.Size()
	}

	// Verify if resume is possible
	if resumePos > 0 {
		headReq, err := http.NewRequest("HEAD", src, nil)
		if err != nil {
			return pd.NotOk(pd.New(err, pd.NetworkError, pd.ErrNetworkDownloadRequest))
		}

		client := &http.Client{}
		resp, err := client.Do(headReq)
		if err != nil {
			return pd.NotOk(pd.New(err, pd.NetworkError, pd.ErrNetworkDownloadRequest))
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return pd.NotOk(pd.New(fmt.Errorf("bad HEAD status: %s", resp.Status), pd.NetworkError, pd.ErrNetworkStatusNotOk))
		}

		// File is already complete
		if resp.ContentLength > 0 && resumePos == resp.ContentLength {
			return pd.Ok()
		}

		// Server doesn't support resume
		if resp.Header.Get("Accept-Ranges") != "bytes" {
			resumePos = 0 // Force full download
		}
	}

	// Open file for writing
	var out io.WriteCloser
	var err error
	if resumePos > 0 {
		out, err = fs.Root.OpenFile(dest, os.O_WRONLY|os.O_APPEND, 0644)
	} else {
		// For full download, remove existing file if it exists
		if err := fs.Root.Remove(dest); err != nil && !os.IsNotExist(err) {
			return pd.NotOk(pd.New(err, pd.FSError, pd.ErrFSRootFileRemove))
		}
		out, err = fs.Root.Create(dest)
	}
	if err != nil {
		return pd.NotOk(pd.New(err, pd.FSError, pd.ErrFSRootFileNew))
	}
	defer func() {
		if cerr := out.Close(); cerr != nil {
			dm.SetToast(pd.New(cerr, pd.FSError, pd.ErrFSRootFileClose), pd.FileManager)
		}
	}()

	// Create request
	req, err := http.NewRequest("GET", src, nil)
	if err != nil {
		return pd.NotOk(pd.New(err, pd.NetworkError, pd.ErrNetworkDownloadRequest))
	}
	if resumePos > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", resumePos))
	}

	// Execute request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return pd.NotOk(pd.New(err, pd.NetworkError, pd.ErrNetworkDownloadRequest))
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			dm.SetToast(pd.New(cerr, pd.FSError, pd.ErrFSRootFileClose), pd.FileManager)
		}
	}()

	// Validate status
	expectedStatus := http.StatusOK
	if resumePos > 0 {
		expectedStatus = http.StatusPartialContent
	}
	if resp.StatusCode != expectedStatus {
		return pd.NotOk(pd.New(
			fmt.Errorf("unexpected status: %s (expected %d)", resp.Status, expectedStatus),
			pd.NetworkError,
			pd.ErrNetworkStatusNotOk,
		))
	}

	// Determine total size
	totalSize := resp.ContentLength
	if resumePos > 0 {
		if cr := resp.Header.Get("Content-Range"); cr != "" {
			parts := strings.Split(cr, "/")
			if len(parts) == 2 {
				if size, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
					totalSize = size
				}
			}
		}
	}

	// Set up progress tracking
	p := &ProgressTracker{
		model:       model,
		filename:    filename,
		totalSize:   totalSize,
		currentSize: resumePos,
		startTime:   time.Now(),
	}

	// Stream to file
	if _, err := io.Copy(out, io.TeeReader(resp.Body, p)); err != nil {
		// Clean up partial file on error
		if resumePos == 0 {
			_ = fs.Root.Remove(dest)
		}
		return pd.NotOk(pd.New(err, pd.FSError, pd.ErrFSRootFileWrite))
	}

	return pd.Ok()
}

func unzipToDir(model *Model, src, dest string) pd.Result {
	dm := model.App().Debug()
	r, err := zip.OpenReader(src)
	if err != nil {
		return pd.NotOk(pd.New(err, pd.FSError, pd.ErrFSRootFileRead))
	}
	defer func() {
		if cerr := r.Close(); cerr != nil {
			dm.SetToast(pd.New(cerr, pd.FSError, pd.ErrFSRootFileClose), pd.FileManager)
		}
	}()

	if err := os.MkdirAll(dest, 0755); err != nil {
		return pd.NotOk(pd.New(err, pd.FSError, pd.ErrFSDirNew))
	}

	for _, f := range r.File {
		if result := extractFile(f, dest, dm); !result.Ok {
			return result
		}
	}
	return pd.Ok()
}

func extractFile(f *zip.File, dest string, dm *pd.Debug) pd.Result {
	if f.FileInfo().IsDir() {
		return pd.Ok()
	}

	rc, err := f.Open()
	if err != nil {
		return pd.NotOk(pd.New(err, pd.FSError, pd.ErrFSFileInvalid))
	}
	defer func() {
		if cerr := rc.Close(); cerr != nil {
			dm.SetToast(pd.New(cerr, pd.FSError, pd.ErrFSRootFileClose), pd.FileManager)
		}
	}()

	targetPath := filepath.Join(dest, f.Name)
	if !strings.HasPrefix(targetPath, filepath.Clean(dest)+string(os.PathSeparator)) {
		return pd.NotOk(pd.New(fmt.Errorf("illegal path: %s", f.Name), pd.FSError, pd.ErrFSFileInvalid))
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return pd.NotOk(pd.New(err, pd.FSError, pd.ErrFSDirNew))
	}

	out, err := os.Create(targetPath)
	if err != nil {
		return pd.NotOk(pd.New(err, pd.FSError, pd.ErrFSFileInvalid))
	}
	defer func() {
		if cerr := out.Close(); cerr != nil {
			dm.SetToast(pd.New(cerr, pd.FSError, pd.ErrFSRootFileClose), pd.FileManager)
		}
	}()

	if _, err := io.Copy(out, rc); err != nil {
		return pd.NotOk(pd.New(err, pd.FSError, pd.ErrFSFileWrite))
	}

	if err := os.Chmod(targetPath, f.Mode()); err != nil {
		return pd.NotOk(pd.New(err, pd.FSError, pd.ErrFSFilePerm))
	}

	return pd.Ok()
}
