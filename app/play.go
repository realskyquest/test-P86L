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
	data := p.model.Data()

	if p.err != nil {
		p86l.GErr = p.err
		return p.err.Err
	}

	p.content.SetModel(p.model)

	p.preReleaseText.SetValue(p86l.T("settings.prerelease"))
	p.preReleaseToggle.SetOnValueChanged(func(value bool) {
		if value {
			data.SetUsePreRelease(true)
		} else {
			data.SetUsePreRelease(false)
		}
		p.err = data.Save()
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

	installButton basicwidget.Button
	playButton    basicwidget.Button
	updateButton  basicwidget.Button

	state      int
	inProgress bool

	gameRunning bool
	gameMutex   sync.Mutex

	model *p86l.Model
}

func (p *playContent) SetModel(model *p86l.Model) {
	p.model = model
}

func (p *playContent) handleDownload(context *guigui.Context) {
	p.inProgress = true

	context.SetEnabled(&p.installButton, false)
	context.SetEnabled(&p.playButton, false)
	context.SetEnabled(&p.updateButton, false)

	cache := p.model.Cache()
	if p.model.Data().File().UsePreRelease {
		pr, rErr := p86l.GetPreRelease()
		if rErr != nil {
			p86l.E.SetPopup(p86l.E.New(rErr, pd.NetworkError, pd.ErrNetworkDownloadRequest))
		}
		assets := pr.Assets

		for _, asset := range assets {
			if name := asset.GetName(); p86l.IsValidPreGameFile(name) {
				downloadUrl := asset.GetBrowserDownloadURL()
				log.Info().Any("Asset", []string{name, downloadUrl}).Str("Play", "playContent").Msg(pd.NetworkManager)
				err := p86l.DownloadGame(p.model, name, downloadUrl, true)
				if err != nil {
					p86l.E.SetPopup(err)
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
					p86l.E.SetPopup(err)
					break
				}

				if err = p.model.Data().SetGameVersion(cache.File().Repo.GetTagName()); err != nil {
					p86l.E.SetToast(err)
					break
				}
				if err = p.model.Data().Save(); err != nil {
					p86l.E.SetToast(err)
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
	data := p.model.Data()

	if p.state != 1 {
		return
	}

	if err := p86l.FS.IsDirR(p86l.E, p86l.FS.DirGamePath()); err != nil {
		p86l.E.SetPopup(p86l.E.New(fmt.Errorf("game not found"), pd.AppError, pd.ErrGameNotExist))
		return
	}

	p.gameMutex.Lock()
	if p.gameRunning {
		p.gameMutex.Unlock()
		p86l.E.SetPopup(p86l.E.New(fmt.Errorf("game is already running."), pd.AppError, pd.ErrGameRunning))
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

		cmd := exec.Command(filepath.Join(p86l.FS.CompanyDirPath, "build", game, "Project-86.exe"))
		if err := cmd.Start(); err != nil {
			p86l.E.SetPopup(p86l.E.New(err, pd.AppError, pd.ErrGameNotExist))
			return
		}

		log.Info().Str("Game", "starting game").Str("Play", "playContent").Msg(pd.FileManager)
		ebiten.MinimizeWindow()

		if err := cmd.Wait(); err != nil {
			if _, ok := err.(*exec.ExitError); !ok {
				p86l.E.SetPopup(p86l.E.New(err, pd.AppError, pd.ErrGameNotExist))
			}
		}
	}()
}

func (p *playContent) handleUpdate(context *guigui.Context) {
	data := p.model.Data()
	cache := p.model.Cache()

	if p.state != 1 && !cache.IsValid() {
		return
	}

	value, err := p86l.CheckNewerVersion(data.File().GameVersion, cache.File().Repo.GetTagName())
	if err != nil {
		p86l.E.SetPopup(err)
		return
	}
	if value {
		log.Info().Str("Game", "New version found").Str("Play", "playContent").Msg(pd.NetworkManager)
		go p.handleDownload(context)
	} else {
		p86l.E.SetPopup(p86l.E.New(fmt.Errorf("newer version not found"), pd.AppError, pd.ErrGameVersionInvalid))
	}
}

func (p *playContent) Build(context *guigui.Context, appender *guigui.ChildWidgetAppender) error {
	data := p.model.Data()
	cache := p.model.Cache()

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

	if data.File().UsePreRelease {
		p.installButton.SetText(p86l.T("play.prerelease"))
	} else {
		p.installButton.SetText(p86l.T("play.install"))
	}
	p.playButton.SetText(p86l.T("play.play"))
	p.updateButton.SetText(p86l.T("play.update"))

	if data.File().UsePreRelease {
		if err := p86l.FS.IsDirR(p86l.E, filepath.Join(p86l.FS.DirBuildPath(), "pregame", "Project-86.exe")); err == nil {
			// play.
			p.state = 1
		} else {
			// install.
			p.state = 0
		}
	} else {
		if err := p86l.FS.IsDirR(p86l.E, filepath.Join(p86l.FS.DirGamePath(), "Project-86.exe")); err == nil {
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
				p86l.E.SetToast(err)
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
			layout.FlexibleSize(1),
			layout.FlexibleSize(1),
		},
		Heights: []layout.Size{
			layout.FlexibleSize(1),
			layout.FixedSize(2 * u),
			layout.FlexibleSize(1),
		},
	}
	switch p.state {
	case 0:
		appender.AppendChildWidgetWithBounds(&p.installButton, gl.CellBounds(1, 1))
	case 1:
		glI := layout.GridLayout{
			Bounds: gl.CellBounds(1, 1),
			Widths: []layout.Size{
				layout.FlexibleSize(1),
				layout.FlexibleSize(1),
			},
			ColumnGap: u / 2,
		}
		appender.AppendChildWidgetWithBounds(&p.playButton, glI.CellBounds(0, 0))
		appender.AppendChildWidgetWithBounds(&p.updateButton, glI.CellBounds(1, 0))
	}

	return nil
}

type playLinks struct {
	guigui.DefaultWidget

	websiteButton basicwidget.Button
	githubButton  basicwidget.Button
	discordButton basicwidget.Button
	patreonButton basicwidget.Button
}

func (p *playLinks) Build(context *guigui.Context, appender *guigui.ChildWidgetAppender) error {
	img1, err1 := assets.TheImageCache.Get(p86l.E, "ie")
	img2, err2 := assets.TheImageCache.Get(p86l.E, "github")
	img3, err3 := assets.TheImageCache.Get(p86l.E, "discord")
	img4, err4 := assets.TheImageCache.Get(p86l.E, "patreon")

	if err := cmp.Or(err1, err2, err3, err4); err != nil {
		p86l.GErr = err
		return err.Err
	}

	p.websiteButton.SetIcon(img1)
	p.githubButton.SetIcon(img2)
	p.discordButton.SetIcon(img3)
	p.patreonButton.SetIcon(img4)

	p.websiteButton.SetText(p86l.T("play.website"))
	p.githubButton.SetText(p86l.T("play.github"))
	p.discordButton.SetText(p86l.T("play.discord"))
	p.patreonButton.SetText(p86l.T("play.patreon"))

	p.websiteButton.SetOnDown(func() {
		go p86l.OpenBrowser(configs.Website)
	})
	p.githubButton.SetOnDown(func() {
		go p86l.OpenBrowser(configs.Github)
	})
	p.discordButton.SetOnDown(func() {
		go p86l.OpenBrowser(configs.Discord)
	})
	p.patreonButton.SetOnDown(func() {
		go p86l.OpenBrowser(configs.Patreon)
	})

	u := basicwidget.UnitSize(context)
	var gl layout.GridLayout
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
		appender.AppendChildWidgetWithBounds(&p.websiteButton, gl.CellBounds(0, 0))
		appender.AppendChildWidgetWithBounds(&p.githubButton, gl.CellBounds(1, 0))
		appender.AppendChildWidgetWithBounds(&p.discordButton, gl.CellBounds(2, 0))
		appender.AppendChildWidgetWithBounds(&p.patreonButton, gl.CellBounds(3, 0))
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
		appender.AppendChildWidgetWithBounds(&p.websiteButton, gl.CellBounds(0, 0))
		appender.AppendChildWidgetWithBounds(&p.githubButton, gl.CellBounds(0, 1))
		appender.AppendChildWidgetWithBounds(&p.discordButton, gl.CellBounds(1, 0))
		appender.AppendChildWidgetWithBounds(&p.patreonButton, gl.CellBounds(1, 1))
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
		appender.AppendChildWidgetWithBounds(&p.websiteButton, gl.CellBounds(0, 0))
		appender.AppendChildWidgetWithBounds(&p.githubButton, gl.CellBounds(0, 1))
		appender.AppendChildWidgetWithBounds(&p.discordButton, gl.CellBounds(0, 2))
		appender.AppendChildWidgetWithBounds(&p.patreonButton, gl.CellBounds(0, 3))
	}

	return nil
}
