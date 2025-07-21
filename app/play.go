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
	"cmp"
	gctx "context"
	"fmt"
	"os/exec"
	"p86l"
	"p86l/assets"
	"p86l/configs"
	pd "p86l/internal/debug"
	"path/filepath"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/guigui"
	"github.com/hajimehoshi/guigui/basicwidget"
	"github.com/hajimehoshi/guigui/layout"
	"github.com/rs/zerolog/log"
)

type Play struct {
	guigui.DefaultWidget

	content playContent
	links   playLinks

	formPre          basicwidget.Form
	preReleaseText   basicwidget.Text
	preReleaseToggle basicwidget.Toggle

	model *p86l.Model

	err *pd.Error
}

func (p *Play) SetModel(model *p86l.Model) {
	p.model = model
}

func (p *Play) Build(context *guigui.Context, appender *guigui.ChildWidgetAppender) error {
	am := p.model.App()
	dm := am.Debug()
	data := p.model.Data()

	if p.err != nil {
		am.SetError(p.err)
		return p.err.Error()
	}

	p.content.model = p.model
	p.links.model = p.model

	p.preReleaseText.SetValue(am.T("settings.prerelease"))
	p.preReleaseToggle.SetOnValueChanged(func(value bool) {
		if value {
			data.SetUsePreRelease(dm, true)
		} else {
			data.SetUsePreRelease(dm, false)
		}
		p.err = data.Save(am)
	})
	if data.File().UsePreRelease {
		p.preReleaseToggle.SetValue(true)
	} else {
		p.preReleaseToggle.SetValue(false)
	}
	p.formPre.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &p.preReleaseText,
			SecondaryWidget: &p.preReleaseToggle,
		},
	})

	u := basicwidget.UnitSize(context)

	var linksSize int
	if breakSize(context, 1024) {
		linksSize = u * 2
	} else if breakSize(context, 640) {
		linksSize = u*5 - (u / 2)
	} else {
		linksSize = u * 6
	}

	gl := layout.GridLayout{
		Bounds: context.Bounds(p).Inset(u / 2),
		Heights: []layout.Size{
			layout.FlexibleSize(1),
			layout.FixedSize(linksSize),
			layout.FixedSize(u * 2),
		},
		RowGap: u / 2,
	}
	appender.AppendChildWidgetWithBounds(&p.content, gl.CellBounds(0, 0))
	appender.AppendChildWidgetWithBounds(&p.links, gl.CellBounds(0, 1))
	appender.AppendChildWidgetWithBounds(&p.formPre, gl.CellBounds(0, 2))

	return nil
}

type playContent struct {
	guigui.DefaultWidget

	installButton  basicwidget.Button
	playButton     basicwidget.Button
	updateButton   basicwidget.Button
	launcherButton basicwidget.Button

	state      int
	inProgress bool

	gameRunning bool
	gameMutex   sync.Mutex

	model *p86l.Model

	sync sync.Once
}

func (p *playContent) handleDownload(context *guigui.Context) {
	am := p.model.App()
	dm := am.Debug()

	p.inProgress = true

	context.SetEnabled(&p.installButton, false)
	context.SetEnabled(&p.playButton, false)
	context.SetEnabled(&p.updateButton, false)

	cache := p.model.Cache()
	if p.model.Data().File().UsePreRelease {
		pr, rErr := p86l.GetPreRelease(am)
		if rErr != nil {
			dm.SetPopup(pd.New(rErr, pd.NetworkError, pd.ErrNetworkDownloadRequest))
		}
		assets := pr.Assets

		for _, asset := range assets {
			if name := asset.GetName(); p86l.IsValidPreGameFile(name) {
				downloadUrl := asset.GetBrowserDownloadURL()
				log.Info().Any("Asset", []string{name, downloadUrl}).Str("Play", "playContent").Msg(pd.NetworkManager)
				err := p86l.DownloadGame(p.model, name, downloadUrl, true)
				if err != nil {
					dm.SetPopup(err)
					break
				}

				break
			}
		}
	} else {
		assets := cache.File().Repo.Assets

		for _, asset := range assets {
			if name := asset.GetName(); p86l.IsValidGameFile(name) {
				downloadUrl := asset.GetBrowserDownloadURL()
				log.Info().Any("Asset", []string{name, downloadUrl}).Str("Play", "playContent").Msg(pd.NetworkManager)
				err := p86l.DownloadGame(p.model, name, downloadUrl, false)
				if err != nil {
					dm.SetPopup(err)
					break
				}

				if err = p.model.Data().SetGameVersion(dm, cache.File().Repo.GetTagName()); err != nil {
					dm.SetToast(err)
					break
				}
				if err = p.model.Data().Save(am); err != nil {
					dm.SetToast(err)
				}
				break
			}
		}
	}

	context.SetEnabled(&p.installButton, true)
	context.SetEnabled(&p.playButton, true)
	context.SetEnabled(&p.updateButton, true)

	p.inProgress = false
}

func (p *playContent) handlePlay() {
	am := p.model.App()
	dm := am.Debug()
	fs := am.FileSystem()
	data := p.model.Data()

	if p.state != 1 {
		return
	}

	if err := fs.IsDirR(fs.DirGamePath()); err != nil {
		dm.SetPopup(pd.New(fmt.Errorf("game not found"), pd.AppError, pd.ErrGameNotExist))
		return
	}

	p.gameMutex.Lock()
	if p.gameRunning {
		p.gameMutex.Unlock()
		dm.SetPopup(pd.New(fmt.Errorf("game is already running."), pd.AppError, pd.ErrGameRunning))
		return
	}
	p.gameRunning = true
	p.gameMutex.Unlock()

	go func() {
		defer func() {
			p.gameMutex.Lock()
			p.gameRunning = false
			p.gameMutex.Unlock()
			log.Info().Str("Game", "game has closed").Str("Play", "playContent").Msg(pd.FileManager)
		}()

		var game string
		if data.File().UsePreRelease {
			game = "pregame"
		} else {
			game = "game"
		}

		cmd := exec.Command(filepath.Join(fs.CompanyDirPath, "build", game, "Project-86.exe"))
		if err := cmd.Start(); err != nil {
			dm.SetPopup(pd.New(err, pd.AppError, pd.ErrGameNotExist))
			return
		}

		log.Info().Str("Game", "starting game").Str("Play", "playContent").Msg(pd.FileManager)
		ebiten.MinimizeWindow()

		if err := cmd.Wait(); err != nil {
			if _, ok := err.(*exec.ExitError); !ok {
				dm.SetPopup(pd.New(err, pd.AppError, pd.ErrGameNotExist))
			}
		}
	}()
}

func (p *playContent) handleUpdate(context *guigui.Context) {
	am := p.model.App()
	dm := am.Debug()
	data := p.model.Data()
	cache := p.model.Cache()

	if p.state != 1 && !cache.IsValid() {
		return
	}

	value, err := p86l.CheckNewerVersion(data.File().GameVersion, cache.File().Repo.GetTagName())
	if err != nil {
		dm.SetPopup(err)
		return
	}
	if value {
		log.Info().Str("Game", "New version found").Str("Play", "playContent").Msg(pd.NetworkManager)
		go p.handleDownload(context)
	} else {
		dm.SetPopup(pd.New(fmt.Errorf("newer version not found"), pd.AppError, pd.ErrGameVersionInvalid))
	}
}

func (p *playContent) Build(context *guigui.Context, appender *guigui.ChildWidgetAppender) error {
	am := p.model.App()
	dm := am.Debug()
	fs := am.FileSystem()
	data := p.model.Data()
	cache := p.model.Cache()

	p.sync.Do(func() {
		context.SetEnabled(&p.launcherButton, false)
		go func() {
			ctx := gctx.Background()
			release, _, rErr := am.GithubClient().Repositories.GetLatestRelease(ctx, configs.CompanyName, configs.AppName)
			if rErr != nil {
				log.Error().Any("Launcher release", rErr).Msg(pd.NetworkManager)
				return
			}
			if am.Version() != nil {
				value, err := p86l.CheckNewerVersion(am.PlainVersion(), release.GetTagName())
				if err != nil {
					dm.SetToast(err)
					return
				}
				log.Info().Str("playContent", "Build").Msg("Launcher Updater")
				if value {
					context.SetEnabled(&p.launcherButton, true)
				}
			}
		}()
	})

	p.installButton.SetOnDown(func() {
		if p.state != 0 && !cache.IsValid() {
			return
		}
		go p.handleDownload(context)
	})
	p.playButton.SetOnDown(p.handlePlay)
	p.updateButton.SetOnDown(func() {
		p.handleUpdate(context)
	})
	p.launcherButton.SetOnDown(func() {

	})

	if data.File().UsePreRelease {
		p.installButton.SetText(am.T("play.prerelease"))
	} else {
		p.installButton.SetText(am.T("play.install"))
	}
	p.playButton.SetText(am.T("play.play"))
	p.updateButton.SetText(am.T("play.update"))
	p.launcherButton.SetText(am.T("play.launcher"))

	if data.File().UsePreRelease {
		if err := fs.IsDirR(filepath.Join(fs.DirBuildPath(), "pregame", "Project-86.exe")); err == nil {
			// play.
			p.state = 1
		} else {
			// install.
			p.state = 0
		}
	} else {
		if err := fs.IsDirR(filepath.Join(fs.DirGamePath(), "Project-86.exe")); err == nil {
			// play.
			p.state = 1
		} else {
			// install.
			p.state = 0
		}
	}

	// if downloading not in progress
	if !p.inProgress {
		// cache not valid.
		if cache.IsValid() {
			context.SetEnabled(&p.installButton, true)
			context.SetEnabled(&p.updateButton, true)
		} else {
			context.SetEnabled(&p.installButton, false)
			context.SetEnabled(&p.updateButton, false)
		}

		// enable update.
		if cache.IsValid() {
			value, err := p86l.CheckNewerVersion(data.File().GameVersion, cache.File().Repo.GetTagName())
			if err != nil {
				dm.SetToast(err)
			}
			if value {
				context.SetEnabled(&p.updateButton, true)
			} else {
				context.SetEnabled(&p.updateButton, false)
			}
		}
	}

	u := basicwidget.UnitSize(context)
	gl := layout.GridLayout{
		Bounds: context.Bounds(p),
		Widths: []layout.Size{
			layout.FlexibleSize(1),
			layout.FlexibleSize(2),
			layout.FlexibleSize(1),
		},
		Heights: []layout.Size{
			layout.FlexibleSize(1),
			layout.FixedSize(2 * u),
			layout.FlexibleSize(1),
		},
	}
	glI := layout.GridLayout{
		Bounds: gl.CellBounds(1, 1),
		Widths: []layout.Size{
			layout.FlexibleSize(1),
			layout.FlexibleSize(1),
			layout.FlexibleSize(1),
		},
		ColumnGap: u / 2,
	}
	switch p.state {
	case 0:
		appender.AppendChildWidgetWithBounds(&p.installButton, gl.CellBounds(1, 1))
	case 1:
		appender.AppendChildWidgetWithBounds(&p.playButton, glI.CellBounds(0, 0))
		appender.AppendChildWidgetWithBounds(&p.updateButton, glI.CellBounds(1, 0))
		appender.AppendChildWidgetWithBounds(&p.launcherButton, glI.CellBounds(2, 0))
	}

	return nil
}

type playLinks struct {
	guigui.DefaultWidget

	websiteButton basicwidget.Button
	githubButton  basicwidget.Button
	discordButton basicwidget.Button
	patreonButton basicwidget.Button

	model *p86l.Model
}

func (p *playLinks) Build(context *guigui.Context, appender *guigui.ChildWidgetAppender) error {
	am := p.model.App()
	dm := am.Debug()

	img1, err1 := assets.TheImageCache.Get("ie")
	img2, err2 := assets.TheImageCache.Get("github")
	img3, err3 := assets.TheImageCache.Get("discord")
	img4, err4 := assets.TheImageCache.Get("patreon")

	if err := cmp.Or(err1, err2, err3, err4); err != nil {
		am.SetError(err)
		return err.Error()
	}

	p.websiteButton.SetIcon(img1)
	p.githubButton.SetIcon(img2)
	p.discordButton.SetIcon(img3)
	p.patreonButton.SetIcon(img4)

	p.websiteButton.SetText(am.T("play.website"))
	p.githubButton.SetText(am.T("play.github"))
	p.discordButton.SetText(am.T("play.discord"))
	p.patreonButton.SetText(am.T("play.patreon"))

	p.websiteButton.SetOnDown(func() {
		go p86l.OpenBrowser(dm, configs.Website)
	})
	p.githubButton.SetOnDown(func() {
		go p86l.OpenBrowser(dm, configs.Github)
	})
	p.discordButton.SetOnDown(func() {
		go p86l.OpenBrowser(dm, configs.Discord)
	})
	p.patreonButton.SetOnDown(func() {
		go p86l.OpenBrowser(dm, configs.Patreon)
	})

	var gl layout.GridLayout
	var bWebsiteButton breakWidget
	var bGithubButton breakWidget
	var bDiscordButton breakWidget
	var bPatreonButton breakWidget

	u := basicwidget.UnitSize(context)

	if breakSize(context, 1024) {
		gl = layout.GridLayout{
			Bounds: context.Bounds(p),
			Heights: []layout.Size{
				layout.FixedSize(u * 2),
				layout.FixedSize(u * 2),
				layout.FixedSize(u * 2),
				layout.FixedSize(u * 2),
			},
			Widths: []layout.Size{
				layout.FlexibleSize(1),
				layout.FlexibleSize(1),
				layout.FlexibleSize(1),
				layout.FlexibleSize(1),
			},
			RowGap:    u / 2,
			ColumnGap: u / 2,
		}
		bWebsiteButton.Set(0, 0)
		bGithubButton.Set(1, 0)
		bDiscordButton.Set(2, 0)
		bPatreonButton.Set(3, 0)
	} else if breakSize(context, 640) {
		gl = layout.GridLayout{
			Bounds: context.Bounds(p),
			Heights: []layout.Size{
				layout.FixedSize(u * 2),
				layout.FixedSize(u * 2),
			},
			Widths: []layout.Size{
				layout.FlexibleSize(1),
				layout.FlexibleSize(1),
			},
			RowGap:    u / 2,
			ColumnGap: u / 2,
		}
		bWebsiteButton.Set(0, 0)
		bGithubButton.Set(0, 1)
		bDiscordButton.Set(1, 0)
		bPatreonButton.Set(1, 1)
	} else {
		gl = layout.GridLayout{
			Bounds: context.Bounds(p),
			Heights: []layout.Size{
				layout.FlexibleSize(1),
				layout.FlexibleSize(1),
				layout.FlexibleSize(1),
				layout.FlexibleSize(1),
			},
			RowGap:    u / 2,
			ColumnGap: u / 2,
		}
		bWebsiteButton.Set(0, 0)
		bGithubButton.Set(0, 1)
		bDiscordButton.Set(0, 2)
		bPatreonButton.Set(0, 3)
	}
	appender.AppendChildWidgetWithBounds(&p.websiteButton, gl.CellBounds(bWebsiteButton.Get()))
	appender.AppendChildWidgetWithBounds(&p.githubButton, gl.CellBounds(bGithubButton.Get()))
	appender.AppendChildWidgetWithBounds(&p.discordButton, gl.CellBounds(bDiscordButton.Get()))
	appender.AppendChildWidgetWithBounds(&p.patreonButton, gl.CellBounds(bPatreonButton.Get()))

	return nil
}
