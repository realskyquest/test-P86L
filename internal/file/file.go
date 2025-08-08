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

package file

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"p86l/configs"
	pd "p86l/internal/debug"
	"path/filepath"

	"github.com/skratchdot/open-golang/open"
)

// Used to make folders.
func mkdirAll(path string) pd.Result {
	_, err := os.Stat(path)
	if !errors.Is(err, fs.ErrNotExist) && err != nil {
		return pd.Ok()
	}
	err = os.MkdirAll(path, 0755)
	if err != nil {
		return pd.NotOk(pd.New(err, pd.FSError, pd.ErrFSDirNew))
	}
	return pd.Ok()
}

type AppFS struct {
	CompanyDirPath string
	Root           *os.Root
}

// NewFS returns a filesystem manager.
func NewFS(extra ...string) (pd.Result, *AppFS) {
	// Handles the company path and a path for debugging.
	var companyPath string
	if len(extra) == 1 && extra[0] != "" {
		if result, cPath := GetCompanyPath(extra[0]); !result.Ok {
			return result, nil
		} else {
			companyPath = cPath
		}
	} else {
		if result, cPath := GetCompanyPath(); !result.Ok {
			return result, nil
		} else {
			companyPath = cPath
		}
	}

	// Makes the path for company and app.
	result := mkdirAll(filepath.Join(companyPath, configs.AppName))
	if !result.Ok {
		return result, nil
	}

	// Creates a virtual filesystem thats in company path, that protects/restricts changes outside of it.
	root, err := os.OpenRoot(companyPath)
	if err != nil {
		return pd.NotOk(pd.New(err, pd.FSError, pd.ErrFSRootInvalid)), nil
	}

	return pd.Ok(), &AppFS{CompanyDirPath: companyPath, Root: root}
}

// Opens the filemanager app with the given path.
func (a *AppFS) OpenFileManager(dm *pd.Debug, path string) pd.Result {
	if err := open.Run(path); err != nil {
		return pd.NotOk(pd.New(err, pd.FSError, pd.ErrFSOpenFileManagerInvalid))
	}
	dm.Log().Info().Str("Path", path).Str("AppFS", "OpenFileManager").Msg("FileManager")
	return pd.Ok()
}

// Path returns a absolute path, .../Project-86-Community/*
func (a *AppFS) Path(components ...string) string {
	return filepath.Join(append([]string{a.CompanyDirPath}, components...)...)
}

// PathRoot returns a relative path.
func (a *AppFS) PathRoot(components ...string) string {
	return filepath.Join(components...)
}

// PathLauncher returns a relative path, Project-86-Launcher/*
func (a *AppFS) PathLauncher(components ...string) string {
	return filepath.Join(append([]string{a.PathDirApp()}, components...)...)
}

// TODO: List?

// Load returns bytes from a file.
func (a *AppFS) Load(dm *pd.Debug, loadFile string) (pd.Result, []byte) {
	if result := a.ExistsRoot(loadFile); !result.Ok {
		return result, nil
	}

	file, err := a.Root.Open(loadFile)
	if err != nil {
		return pd.NotOk(pd.New(err, pd.FSError, pd.ErrFSRootFileInvalid)), nil
	}
	defer func() {
		if err := file.Close(); err != nil {
			dm.SetToast(pd.New(err, pd.FSError, pd.ErrFSRootFileClose), pd.FileManager)
		}
	}()
	b, err := io.ReadAll(file)
	if err != nil {
		return pd.NotOk(pd.New(err, pd.FSError, pd.ErrFSRootFileRead)), nil
	}

	return pd.Ok(), b
}

// Exists returns a result based on absolute path.
func (a *AppFS) Exists(components ...string) pd.Result {
	path := filepath.Join(append([]string{a.CompanyDirPath}, components...)...)
	_, err := os.Stat(path)
	if err == nil {
		return pd.Ok()
	}
	return pd.NotOk(pd.New(err, pd.FSError, pd.ErrFSFileNotExist))
}

// ExistsRoot returns a result based on relative.
func (a *AppFS) ExistsRoot(components ...string) pd.Result {
	path := filepath.Join(components...)
	_, err := a.Root.Stat(path)
	if err == nil {
		return pd.Ok()
	}
	return pd.NotOk(pd.New(err, pd.FSError, pd.ErrFSRootFileNotExist))
}

// ExistsLauncher returns a result based on relative of launcher.
func (a *AppFS) ExistsLauncher(components ...string) pd.Result {
	path := filepath.Join(append([]string{a.PathDirApp()}, components...)...)
	_, err := a.Root.Stat(path)
	if err == nil {
		return pd.Ok()
	}
	return pd.NotOk(pd.New(err, pd.FSError, pd.ErrFSRootFileNotExist))
}

// Save writes a file.
func (a *AppFS) Save(dm *pd.Debug, saveFile string, bytes []byte) pd.Result {
	file, err := a.Root.Create(saveFile)
	if err != nil {
		return pd.NotOk(pd.New(err, pd.FSError, pd.ErrFSRootFileNew))
	}
	defer func() {
		if err := file.Close(); err != nil {
			dm.SetToast(pd.New(err, pd.FSError, pd.ErrFSRootFileClose), pd.FileManager)
		}
	}()
	_, err = file.Write(bytes)
	if err != nil {
		return pd.NotOk(pd.New(err, pd.FSError, pd.ErrFSRootFileWrite))
	}

	return pd.Ok()
}
