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
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	pd "p86l/internal/debug"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
)

func DownloadGame(model *Model, filename, src string, preRelease bool) pd.Result {
	am := model.App()
	fs := am.FileSystem()
	play := model.Play()
	gameType := "game"
	if preRelease {
		gameType = "pregame"
	}

	play.SetProgress("Downloading...")
	zipName := fmt.Sprintf("%s.zip", gameType)

	err := DownloadFile(
		src,
		filepath.Join(fs.CompanyDirPath, zipName),
		5,
		func(d, t int64, s float64) {
			var remainingStr string
			if s > 0 {
				remaining := float64(t-d) / s
				remainingDuration := time.Duration(remaining) * time.Second
				remainingStr = humanize.RelTime(time.Now(), time.Now().Add(remainingDuration), "remaining", "ago")
			} else {
				remainingStr = "calculating..."
			}

			output := fmt.Sprintf("Downloading %s: %s/%s @ %s/s, %s",
				filename,
				humanize.Bytes(uint64(d)),
				humanize.Bytes(uint64(t)),
				humanize.Bytes(uint64(s)),
				remainingStr,
			)
			play.SetProgress(output)
		},
	)
	if err != nil {
		play.SetProgress("")
		return pd.NotOk(pd.New(err, pd.NetworkError, pd.ErrNetworkDownloadRequest))
	}

	play.SetProgress("Removing old files...")
	gameDir := filepath.Join(fs.CompanyDirPath, "build", gameType)
	if err := os.RemoveAll(gameDir); err != nil {
		play.SetProgress("")
		return pd.NotOk(pd.New(err, pd.FSError, pd.ErrFSDirRemove))
	}

	play.SetProgress("Extracting...")
	srcPath := filepath.Join(fs.CompanyDirPath, zipName)
	if result := unzipToDir(model, srcPath, gameDir); !result.Ok {
		play.SetProgress("")
		return result
	}

	play.SetProgress("Cleaning...")
	if err := fs.Root.Remove(zipName); err != nil {
		play.SetProgress("")
		return pd.NotOk(pd.New(err, pd.FSError, pd.ErrFSRootFileRemove))
	}

	play.SetProgress("")
	return pd.Ok()
}

type ProgressFunc func(downloaded, total int64, speed float64)

type progressReader struct {
	reader     io.Reader
	total      *int64 // Pointer to external downloaded counter
	startSize  int64
	startTime  time.Time
	totalRead  int64
	onProgress func(speed float64)
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	if n > 0 {
		pr.totalRead += int64(n)
		*pr.total = pr.startSize + pr.totalRead

		// Calculate speed and report progress
		duration := time.Since(pr.startTime).Seconds()
		if duration > 0.1 { // Avoid division by zero and excessive updates
			speed := float64(pr.totalRead) / duration
			pr.onProgress(speed)
		}
	}
	return n, err
}

// isProtocolError checks if error is a stream PROTOCOL_ERROR
func isProtocolError(err error) bool {
	return strings.Contains(err.Error(), "PROTOCOL_ERROR")
}

func DownloadFile(downloadURL, filePath string, maxRetries int, progress ProgressFunc) error {
	// Get target filename from URL if not specified
	if filePath == "" {
		filePath = filepath.Base(downloadURL)
	}

	// Create HTTP clients
	clientHTTP2 := &http.Client{}
	clientHTTP11 := &http.Client{
		Transport: &http.Transport{
			TLSNextProto: make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
		},
	}

	currentClient := clientHTTP2
	retryDelay := 2 * time.Second
	var file *os.File

	// Track download metrics
	var (
		fileSize   int64
		totalSize  int64 = -1
		downloaded int64
	)

	// Get initial file size for resume
	if info, err := os.Stat(filePath); err == nil {
		fileSize = info.Size()
		downloaded = fileSize
	}

	// Parse total size from Content-Range header
	parseTotalSize := func(header http.Header) {
		cr := header.Get("Content-Range")
		if cr == "" {
			return
		}
		parts := strings.Split(cr, "/")
		if len(parts) < 2 {
			return
		}
		if size, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
			totalSize = size
		}
	}

	// Progress reporter helper
	reportProgress := func(speed float64) {
		if progress != nil {
			progress(downloaded, totalSize, speed)
		}
	}

	// Initial progress report
	reportProgress(0)

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Create request
		req, err := http.NewRequest("GET", downloadURL, nil)
		if err != nil {
			return fmt.Errorf("request creation failed: %w", err)
		}

		// Set resume headers if needed
		if downloaded > 0 {
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-", downloaded))
		}

		// Execute request
		resp, err := currentClient.Do(req)
		if err != nil {
			if isProtocolError(err) && currentClient == clientHTTP2 {
				currentClient = clientHTTP11 // Switch to HTTP/1.1
			}
			if attempt < maxRetries {
				time.Sleep(retryDelay)
				retryDelay *= 2
				continue
			}
			return fmt.Errorf("request failed: %w", err)
		}

		fmt.Println("Download retry", attempt)

		// Handle response status
		switch resp.StatusCode {
		case http.StatusOK:
			if totalSize == -1 {
				totalSize = resp.ContentLength
			}
		case http.StatusPartialContent:
			parseTotalSize(resp.Header)
		case http.StatusRequestedRangeNotSatisfiable:
			resp.Body.Close()
			reportProgress(0)
			return nil // Already downloaded
		default:
			resp.Body.Close()
			return fmt.Errorf("unexpected HTTP status: %s", resp.Status)
		}

		// Open file in appropriate mode
		if file == nil {
			if downloaded > 0 && resp.StatusCode == http.StatusPartialContent {
				file, err = os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0644)
			} else {
				file, err = os.Create(filePath)
			}
			if err != nil {
				resp.Body.Close()
				return fmt.Errorf("file creation failed: %w", err)
			}
		}

		// Create progress reader
		startTime := time.Now()
		progressReader := &progressReader{
			reader:     resp.Body,
			total:      &downloaded,
			startSize:  downloaded,
			startTime:  startTime,
			onProgress: reportProgress,
		}

		// Download the file
		_, err = io.Copy(file, progressReader)

		// Close response body and handle error
		respCloseErr := resp.Body.Close()
		if err == nil {
			err = respCloseErr
		}

		if err == nil {
			// Final progress report
			if progressReader.totalRead > 0 {
				duration := time.Since(startTime).Seconds()
				speed := float64(progressReader.totalRead) / duration
				reportProgress(speed)
			} else {
				reportProgress(0)
			}
			break
		}

		// Handle copy errors
		if isProtocolError(err) && currentClient == clientHTTP2 {
			currentClient = clientHTTP11 // Switch to HTTP/1.1
		}

		// Update downloaded size for next attempt
		if pos, err := file.Seek(0, io.SeekCurrent); err == nil {
			downloaded = pos
		}

		if attempt < maxRetries {
			reportProgress(0)
			time.Sleep(retryDelay)
			retryDelay *= 2
			continue
		}
		return fmt.Errorf("download failed: %w", err)
	}

	// Close file and return any error
	if file != nil {
		if err := file.Close(); err != nil {
			return fmt.Errorf("file close error: %w", err)
		}
	}
	return nil
}

func unzipToDir(model *Model, src, dest string) pd.Result {
	dm := model.App().Debug()
	r, err := zip.OpenReader(src)
	if err != nil {
		return pd.NotOk(pd.New(err, pd.FSError, pd.ErrFSRootFileRead))
	}
	defer func() {
		if err := r.Close(); err != nil {
			dm.SetToast(pd.New(err, pd.FSError, pd.ErrFSRootFileClose), pd.FileManager)
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
		if err := rc.Close(); err != nil {
			dm.SetToast(pd.New(err, pd.FSError, pd.ErrFSRootFileClose), pd.FileManager)
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
		if err := out.Close(); err != nil {
			dm.SetToast(pd.New(err, pd.FSError, pd.ErrFSRootFileClose), pd.FileManager)
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
