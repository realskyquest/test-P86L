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
	"os/exec"
	pd "p86l/internal/debug"
	"path/filepath"
	"sync"

	"github.com/google/go-github/v71/github"
	"github.com/shirou/gopsutil/v4/process"
)

type gameAvailable struct {
	Progress  bool
	Available chan bool
}

type PlayModel struct {
	progress    string        // Used to display message in sidebar.
	canInteract bool          // Used to disable buttons in play.
	available   gameAvailable // Used to display whether game files are present.
	exeMutex    sync.Mutex
	runningPID  int32
}

// -- Getters --

func (p *PlayModel) Progress() string {
	return p.progress
}

func (p *PlayModel) CanInteract() bool {
	return p.canInteract
}

func (p *PlayModel) GameAvailable() gameAvailable {
	return p.available
}

func (p *PlayModel) RunningPID() int32 {
	return p.runningPID
}

// -- Setters --

func (p *PlayModel) SetProgress(value string) {
	p.progress = value
}

func (p *PlayModel) SetInteract(value bool) {
	p.canInteract = value
}

func (p *PlayModel) SetGameAvailable(model *Model, init, progress bool) {
	am := model.App()
	fs := am.FileSystem()

	if init {
		p.available.Available = make(chan bool, 1)
		return
	}

	if progress {
		p.available.Progress = true
	} else {
		p.available.Progress = false
		return
	}

	if result := fs.ExistsRoot(model.GameExecutablePath()); !result.Ok {
		p.available.Available <- false
		return
	}
	p.available.Available <- true
}

func (p *PlayModel) SetPID(pid int32) {
	p.runningPID = pid
}

// -- Common utils --

func (p *PlayModel) CanUpdate(model *Model) bool {
	dm := model.App().Debug()
	data := model.Data()
	cache := model.Cache()

	if !cache.IsValid() {
		return false
	}
	if data.File().GameVersion == "" {
		return false
	}
	result, uValue := CheckNewerVersion(data.File().GameVersion, cache.File().Repo.GetTagName())
	if !result.Ok {
		dm.SetToast(result.Err, pd.FileManager)
		return false
	}
	if uValue {
		return true
	}

	return false
}

func (p *PlayModel) isExeRunning(exeName string) (bool, int32) {
	processes, err := process.ProcessesWithContext(context.Background())
	if err != nil {
		return false, -1
	}

	for _, p := range processes {
		name, err := p.NameWithContext(context.Background())
		if err == nil && name == exeName {
			return true, p.Pid
		}

		exePath, err := p.ExeWithContext(context.Background())
		if err == nil && filepath.Base(exePath) == exeName {
			return true, p.Pid
		}
	}

	return false, -1
}

func (p *PlayModel) runExe(model *Model) (int32, error) {
	p.exeMutex.Lock()
	defer p.exeMutex.Unlock()

	exeName := "Project-86.exe"

	if isRunning, pid := p.isExeRunning(exeName); isRunning {
		p.runningPID = pid
		return -1, fmt.Errorf("Executable already running (PID %d)", pid)
	}

	cmd := exec.Command(filepath.Join(model.App().FileSystem().CompanyDirPath, model.GameExecutablePath()))
	if err := cmd.Start(); err != nil {
		return -1, fmt.Errorf("Failed to start executable: %w", err)
	}

	go func() {
		_ = cmd.Wait()
		p.exeMutex.Lock()
		p.runningPID = -1
		p.exeMutex.Unlock()
	}()

	p.runningPID = int32(cmd.Process.Pid)
	return p.runningPID, nil
}

// -- Common --

func (p *PlayModel) Load(model *Model) {
	p.SetInteract(true)
	p.SetGameAvailable(model, true, false)
	p.SetPID(-1)
}

func (p *PlayModel) HandleGameDownload(model *Model, installOrUpdate string) {
	am := model.App()
	dm := am.Debug()
	log := dm.Log()
	data := model.Data()
	cache := model.Cache()

	if !cache.IsValid() {
		return
	}

	p.canInteract = false

	cacheAssets := cache.File().Repo.Assets
	if data.File().UsePreRelease {
		cacheAssets = cache.file.PreRepo.Assets
	}

	downloadAsset := func(assetName string, asset *github.ReleaseAsset, notPrerelease bool) {
		downloadUrl := asset.GetBrowserDownloadURL()
		log.Info().Any("Asset", []string{assetName, downloadUrl}).Str("HandleGameDownload", installOrUpdate).Msg(pd.NetworkManager)
		if result := DownloadGame(model, assetName, downloadUrl, data.File().UsePreRelease); !result.Ok {
			dm.SetPopup(result.Err, pd.NetworkManager)
		} else {
			if notPrerelease {
				result := data.SetGameVersion(dm, cache.File().Repo.GetTagName())
				if !result.Ok {
					dm.SetToast(result.Err, pd.FileManager)
				}
			}
		}
	}

	for _, asset := range cacheAssets {
		assetName := asset.GetName()
		if data.File().UsePreRelease {
			if IsValidPreGameFile(assetName) {
				downloadAsset(assetName, asset, false)
			}
		} else {
			if IsValidGameFile(assetName) {
				downloadAsset(assetName, asset, true)
			}
		}
	}

	p.canInteract = true
}

func (p *PlayModel) HandlePlay(model *Model) {
	dm := model.App().Debug()
	pid, err := p.runExe(model)
	switch {
	case err != nil && pid == -1:
		dm.SetPopup(pd.New(fmt.Errorf("Already running: %w", err), pd.AppError, pd.ErrGameRunning), pd.FileManager)
	case err != nil:
		dm.SetPopup(pd.New(fmt.Errorf("Start failed: %w", err), pd.AppError, pd.ErrGameRunning), pd.FileManager)
	default:
		dm.SetPopup(pd.New(fmt.Errorf("Launched PID %d", pid), pd.UnknownError, pd.ErrUnknown), pd.FileManager)
	}
}
