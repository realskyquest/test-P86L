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

package app

import (
	"context"
	"p86l"
	"p86l/configs"
	pd "p86l/internal/debug"
	"time"

	"github.com/google/go-github/v71/github"
)

type prereleaseResult struct {
	result     pd.Result
	prerelease bool
}

type gameFileResult struct {
	gameFile bool
}

type launcherReleaseResult struct {
	result  pd.Result
	release *github.RepositoryRelease
}

func (p *playContent) handlePrerelease(model *p86l.Model, value bool) {
	am := model.App()
	dm := am.Debug()
	data := model.Data()

	data.SetUsePreRelease(dm, value)
	result := data.Save(am)

	if !result.Ok {
		p.prResult <- prereleaseResult{
			result:     result,
			prerelease: false,
		}
		return
	}
	p.prResult <- prereleaseResult{
		result:     pd.Ok(),
		prerelease: data.File().UsePreRelease,
	}
}

func (p *playButtons) handleGameFile(model *p86l.Model) {
	am := model.App()
	fs := am.FileSystem()

	if result := fs.ExistsRoot(model.GameExecutablePath()); !result.Ok {
		p.gFResult <- gameFileResult{
			gameFile: false,
		}
		return
	}
	p.gFResult <- gameFileResult{
		gameFile: true,
	}
}

func (p *playButtons) fetchLauncherRelease(model *p86l.Model) {
	am := model.App()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	release, _, err := am.GithubClient().Repositories.GetLatestRelease(ctx, configs.CompanyName, configs.AppName)

	if err != nil {
		p.lRResult <- launcherReleaseResult{
			result:  pd.NotOk(pd.New(err, pd.NetworkError, pd.ErrNetworkLatestInvalid)),
			release: nil,
		}
		return
	}
	p.lRResult <- launcherReleaseResult{
		result:  pd.Ok(),
		release: release,
	}
}

// -- Button handlers: ran as goroutine --

func (p *playButtons) handleGameDownload(model *p86l.Model, InstallOrUpdate string) {
	if !model.Cache().IsValid() {
		return
	}

	p.progress = true

	am := model.App()
	dm := am.Debug()
	log := dm.Log()
	data := model.Data()
	cache := model.Cache()

	cacheAssets := cache.File().Repo.Assets
	if data.File().UsePreRelease {
		cacheAssets = cache.File().PreRepo.Assets
	}

	downloadAsset := func(assetName string, asset *github.ReleaseAsset) {
		downloadUrl := asset.GetBrowserDownloadURL()
		log.Info().Any("Asset", []string{assetName, downloadUrl}).Str("playButtons", InstallOrUpdate).Msg(pd.NetworkManager)
		result := p86l.DownloadGame(model, assetName, downloadUrl, data.File().UsePreRelease)
		if !result.Ok {
			dm.SetPopup(result.Err, pd.NetworkManager)
		}
	}

	for _, asset := range cacheAssets {
		assetName := asset.GetName()
		if data.File().UsePreRelease {
			if p86l.IsValidPreGameFile(assetName) {
				downloadAsset(assetName, asset)
			}
		} else {
			if p86l.IsValidGameFile(assetName) {
				downloadAsset(assetName, asset)
			}
		}
	}

	p.progress = false
}

func (p *playButtons) handlePlay() {
	p.progress = true
	time.Sleep(5 * time.Second)
	p.progress = false
}

func (p *playButtons) handleLauncher() {

}
