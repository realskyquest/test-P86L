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

func DownloadGame(model *Model, filename, src string, preRelease bool) *pd.Error {
	am := model.App()
	dm := am.Debug()
	fs := am.FileSystem()

	var game string
	if preRelease {
		game = "pregame"
	} else {
		game = "game"
	}
	gameZip := fmt.Sprintf("%s.zip", game)

	model.SetProgress("Downloading...")

	err := DownloadFile(model, filename, src, gameZip)
	if err != nil {
		model.SetProgress("")
		return err
	}

	if fs.IsDirR(fs.DirGamePath()) == nil {
		model.SetProgress("Removing old files...")

		rErr := os.RemoveAll(filepath.Join(fs.CompanyDirPath, "build", game))
		if rErr != nil {
			model.SetProgress("")
			return pd.New(rErr, pd.FSError, pd.ErrFSDirRemove)
		}
	}

	model.SetProgress("Extracting...")

	err = unzip(dm, filepath.Join(fs.CompanyDirPath, gameZip), filepath.Join(fs.CompanyDirPath, "build", game))
	if err != nil {
		model.SetProgress("")
		return err
	}

	model.SetProgress("Cleaning...")

	rErr := fs.Root.Remove(gameZip)
	if rErr != nil {
		model.SetProgress("")
		return pd.New(rErr, pd.FSError, pd.ErrFSRootFileRemove)
	}

	model.SetProgress("")

	return nil
}

func DownloadLauncher() {
	// TODO:
}

func DownloadFile(model *Model, filename, src, dest string) *pd.Error {
	am := model.App()
	dm := am.Debug()
	fs := am.FileSystem()

	var resumePos int64 = 0
	if info, err := fs.Root.Stat(dest); err == nil {
		resumePos = info.Size()
	}

	if resumePos > 0 {
		// Make a HEAD request to get the file size without downloading.
		headReq, err := http.NewRequest("HEAD", src, nil)
		if err != nil {
			return pd.New(err, pd.NetworkError, pd.ErrNetworkDownloadRequest)
		}

		client := &http.Client{}
		headResp, err := client.Do(headReq)
		if err != nil {
			return pd.New(err, pd.NetworkError, pd.ErrNetworkDownloadRequest)
		}
		err = headResp.Body.Close()
		if err != nil {
			return pd.New(err, pd.NetworkError, pd.ErrNetworkDownloadRequest)
		}

		if headResp.StatusCode == http.StatusOK {
			contentLength := headResp.ContentLength
			// If the file size matches, skip download.
			if contentLength > 0 && resumePos == contentLength {
				// File is already fully downloaded.
				return nil
			}
		}
	}

	var out io.WriteCloser
	var err error
	if resumePos > 0 {
		out, err = fs.Root.OpenFile(dest, os.O_WRONLY|os.O_APPEND, 0644)
	} else {
		out, err = fs.Root.Create(dest)
	}
	if err != nil {
		return pd.New(err, pd.FSError, pd.ErrFSRootFileNew)
	}
	defer func() {
		err := out.Close()
		if err != nil {
			dm.SetToast(pd.New(err, pd.FSError, pd.ErrFSRootFileClose))
		}
	}()

	req, err := http.NewRequest("GET", src, nil)
	if err != nil {
		return pd.New(err, pd.NetworkError, pd.ErrNetworkDownloadRequest)
	}

	if resumePos > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", resumePos))
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return pd.New(err, pd.NetworkError, pd.ErrNetworkDownloadRequest)
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			dm.SetToast(pd.New(err, pd.FSError, pd.ErrFSRootFileClose))
		}
	}()

	expectedStatus := http.StatusOK
	if resumePos > 0 {
		expectedStatus = http.StatusPartialContent
	}

	if resp.StatusCode != expectedStatus {
		if resumePos > 0 && resp.StatusCode != http.StatusPartialContent {
			err := out.Close()
			if err != nil {
				return pd.New(err, pd.FSError, pd.ErrFSRootFileClose)
			}
			return DownloadFileFromScratch(model, filename, src, dest)
		}
		return pd.New(fmt.Errorf("bad status: %s", resp.Status), pd.NetworkError, pd.ErrNetworkStatusNotOk)
	}

	var totalSize int64
	if resumePos > 0 {
		if contentRange := resp.Header.Get("Content-Range"); contentRange != "" {
			if parts := strings.Split(contentRange, "/"); len(parts) == 2 {
				if size, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
					totalSize = size
				}
			}
		}
	} else {
		totalSize = resp.ContentLength
	}

	p := &ProgressTracker{
		model:       model,
		filename:    filename,
		totalSize:   totalSize,
		currentSize: resumePos,
		startTime:   time.Now(),
	}

	_, err = io.Copy(out, io.TeeReader(resp.Body, p))
	if err != nil {
		return pd.New(err, pd.FSError, pd.ErrFSRootFileWrite)
	}

	return nil
}

func DownloadFileFromScratch(model *Model, filename, src, dest string) *pd.Error {
	am := model.App()
	dm := am.Debug()
	fs := am.FileSystem()

	// Remove existing file
	err := fs.Root.Remove(dest)
	if err != nil {
		return pd.New(err, pd.FSError, pd.ErrFSRootFileRemove)
	}

	// Create new file
	out, err := fs.Root.Create(dest)
	if err != nil {
		return pd.New(err, pd.FSError, pd.ErrFSRootFileNew)
	}
	defer func() {
		err := out.Close()
		if err != nil {
			dm.SetToast(pd.New(err, pd.FSError, pd.ErrFSRootFileClose))
		}
	}()

	resp, err := http.Get(src)
	if err != nil {
		return pd.New(err, pd.NetworkError, pd.ErrNetworkDownloadRequest)
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			dm.SetToast(pd.New(err, pd.FSError, pd.ErrFSRootFileClose))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return pd.New(fmt.Errorf("bad status: %s", resp.Status), pd.NetworkError, pd.ErrNetworkStatusNotOk)
	}

	p := &ProgressTracker{
		model:       model,
		filename:    filename,
		totalSize:   resp.ContentLength,
		currentSize: 0,
		startTime:   time.Now(),
	}

	_, err = io.Copy(out, io.TeeReader(resp.Body, p))
	if err != nil {
		err := fs.Root.Remove(dest)
		if err != nil {
			return pd.New(err, pd.FSError, pd.ErrFSRootFileRemove)
		}
		return pd.New(err, pd.FSError, pd.ErrFSRootFileWrite)
	}

	return nil
}

func unzip(dm *pd.Debug, src, dest string) *pd.Error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return pd.New(err, pd.FSError, pd.ErrFSRootFileRead)
	}
	defer func() {
		err := r.Close()
		if err != nil {
			dm.SetToast(pd.New(err, pd.FSError, pd.ErrFSRootFileClose))
		}
	}()

	if err := os.MkdirAll(dest, 0755); err != nil {
		return pd.New(err, pd.FSError, pd.ErrFSDirNew)
	}

	// Find common root directory if it exists
	var rootDir string
	if len(r.File) > 0 {
		firstPath := r.File[0].Name
		rootDir = strings.Split(firstPath, "/")[0] + "/"

		// Verify all files have this common root
		for _, f := range r.File {
			if !strings.HasPrefix(f.Name, rootDir) {
				rootDir = "" // No common root
				break
			}
		}
	}

	extractAndWriteFile := func(f *zip.File) *pd.Error {
		// Skip the root directory itself
		if f.Name == rootDir || f.Name == strings.TrimSuffix(rootDir, "/") {
			return nil
		}

		rc, err := f.Open()
		if err != nil {
			return pd.New(err, pd.FSError, pd.ErrFSFileInvalid)
		}
		defer func() {
			err := rc.Close()
			if err != nil {
				dm.SetToast(pd.New(err, pd.FSError, pd.ErrFSRootFileClose))
			}
		}()

		// Strip the root directory from the path
		relPath := f.Name
		if rootDir != "" {
			relPath = strings.TrimPrefix(f.Name, rootDir)
		}

		path := filepath.Join(dest, relPath)

		// Check for ZipSlip vulnerability.
		if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) {
			return pd.New(fmt.Errorf("illegal file path: %s", path), pd.FSError, pd.ErrFSFileInvalid)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(path, f.Mode()); err != nil {
				return pd.New(err, pd.FSError, pd.ErrFSDirNew)
			}
		} else {
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return pd.New(err, pd.FSError, pd.ErrFSDirNew)
			}

			fi, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return pd.New(err, pd.FSError, pd.ErrFSFileInvalid)
			}
			defer func() {
				err := fi.Close()
				if err != nil {
					dm.SetToast(pd.New(err, pd.FSError, pd.ErrFSRootFileClose))
				}
			}()

			_, err = io.Copy(fi, rc)
			if err != nil {
				return pd.New(err, pd.FSError, pd.ErrFSFileWrite)
			}
		}
		return nil
	}

	for _, f := range r.File {
		err := extractAndWriteFile(f)
		if err != nil {
			return err
		}
	}

	return nil
}
