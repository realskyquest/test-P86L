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
	"image"
	"p86l"
	"p86l/assets"
	"p86l/configs"
	pd "p86l/internal/debug"
	"sync"

	"github.com/hajimehoshi/guigui"
	"github.com/hajimehoshi/guigui/basicwidget"
	"github.com/hajimehoshi/guigui/layout"
)

type Play struct {
	guigui.DefaultWidget

	panel   basicwidget.Panel
	content playContent
}

func (p *Play) AppendChildWidgets(context *guigui.Context, appender *guigui.ChildWidgetAppender) {
	appender.AppendChildWidget(&p.panel)
}

func (p *Play) Build(context *guigui.Context) error {
	bounds := context.Bounds(p)
	contentHeight := p.content.Height()

	var contentSize image.Point
	if bounds.Dy() > contentHeight {
		contentSize = image.Pt(bounds.Dx(), bounds.Dy())
	} else {
		contentSize = image.Pt(bounds.Dx(), contentHeight)
	}
	context.SetSize(&p.content, contentSize, p)
	p.panel.SetContent(&p.content)
	context.SetBounds(&p.panel, context.Bounds(p), p)

	return nil
}

type playContent struct {
	guigui.DefaultWidget

	buttons          playButtons
	links            playLinks
	form             basicwidget.Form
	prereleaseText   basicwidget.Text
	prereleaseToggle basicwidget.Toggle

	box1   basicwidget.Background
	box2   basicwidget.Background
	box3   basicwidget.Background
	height int

	prProgress bool
	prResult   chan prereleaseResult

	sync   sync.Once
	result pd.Result
}

func (p *playContent) AppendChildWidgets(context *guigui.Context, appender *guigui.ChildWidgetAppender) {
	model := context.Model(p, modelKeyModel).(*p86l.Model)
	am := model.App()

	am.RenderBox(appender, &p.box1)
	am.RenderBox(appender, &p.box2)
	am.RenderBox(appender, &p.box3)
	appender.AppendChildWidget(&p.buttons)
	appender.AppendChildWidget(&p.links)
	appender.AppendChildWidget(&p.form)
}

func (p *playContent) Build(context *guigui.Context) error {
	model := context.Model(p, modelKeyModel).(*p86l.Model)
	am := model.App()
	dm := am.Debug()
	data := model.Data()

	p.sync.Do(func() {
		p.result = pd.Ok()
		p.prResult = make(chan prereleaseResult, 1)
	})

	if !p.result.Ok {
		am.SetError(p.result)
		return p.result.Err.Error()
	}

	// TODO: Move this to Root
	select {
	case prResult := <-p.prResult:
		p.prProgress = false
		if !prResult.result.Ok {
			dm.SetToast(prResult.result.Err, pd.FileManager)
		}
	default:

	}

	p.prereleaseText.SetValue(am.T("settings.prerelease"))
	p.prereleaseToggle.SetOnValueChanged(func(value bool) {
		if value == data.File().UsePreRelease {
			return
		}
		if !p.prProgress {
			p.prProgress = true
			go p.handlePrerelease(model, value)
		}
	})
	if data.File().UsePreRelease {
		p.prereleaseToggle.SetValue(true)
	} else {
		p.prereleaseToggle.SetValue(false)
	}
	p.form.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &p.prereleaseText,
			SecondaryWidget: &p.prereleaseToggle,
		},
	})

	u := basicwidget.UnitSize(context)
	gl := layout.GridLayout{
		Bounds: context.Bounds(p).Inset(u / 2),
		Widths: []layout.Size{
			layout.FlexibleSize(1),
		},
		Heights: []layout.Size{
			layout.FlexibleSize(1),
			layout.FixedSize(p.buttons.Height()),
			layout.FlexibleSize(1),
			layout.FixedSize(p.links.Height()),
			layout.FixedSize(p.form.DefaultSizeInContainer(context, context.Bounds(p).Dx()-u).Y),
		},
		RowGap: u / 2,
	}
	p.height = (gl.CellBounds(0, 1).Dy() + gl.CellBounds(0, 3).Dy() + gl.CellBounds(0, 4).Dy()) + u*3
	context.SetBounds(&p.box1, gl.CellBounds(0, 1), p)
	context.SetBounds(&p.box2, gl.CellBounds(0, 3), p)
	context.SetBounds(&p.box3, gl.CellBounds(0, 4), p)
	context.SetBounds(&p.buttons, gl.CellBounds(0, 1), p)
	context.SetBounds(&p.links, gl.CellBounds(0, 3), p)
	context.SetBounds(&p.form, gl.CellBounds(0, 4), p)

	return nil
}

func (a *playContent) Height() int {
	return a.height
}

type playButtons struct {
	guigui.DefaultWidget

	buttons [4]basicwidget.Button

	height int

	progress      bool
	progressMutex sync.Mutex

	gFResult   chan gameFileResult
	lRResult   chan launcherReleaseResult
	gFProgress bool

	sync sync.Once
}

func (p *playButtons) AppendChildWidgets(context *guigui.Context, appender *guigui.ChildWidgetAppender) {
	for i := range p.buttons {
		appender.AppendChildWidget(&p.buttons[i])
	}
}

func (p *playButtons) Build(context *guigui.Context) error {
	model := context.Model(p, modelKeyModel).(*p86l.Model)
	am := model.App()
	dm := am.Debug()
	data := model.Data()
	cache := model.Cache()

	p.sync.Do(func() {
		p.gFResult = make(chan gameFileResult, 1)
		p.lRResult = make(chan launcherReleaseResult, 1)

		go p.fetchLauncherRelease(model)
	})

	buttonTexts := []string{"play.install", "play.update", "play.play", "play.launcher"}
	buttonActions := []func(){
		func() { go p.handleGameDownload(model, "handleInstall") },
		func() { go p.handleGameDownload(model, "handleUpdate") },
		func() { go p.handlePlay() },
		func() { go p.handleLauncher() },
	}

	for i := range p.buttons {
		if t := buttonTexts[i]; t == "play.install" {
			if data.File().UsePreRelease {
				p.buttons[i].SetText(am.T("play.prerelease"))
			} else {
				p.buttons[i].SetText(am.T(t))
			}
		} else {
			p.buttons[i].SetText(am.T(t))
		}
		p.buttons[i].SetOnDown(buttonActions[i])
	}

	for range 2 {
		select {
		case gFResult := <-p.gFResult: // Handle game files.
			p.gFProgress = false

			for i := range p.buttons {
				switch i {
				case 0: // Install
					if gFResult.gameFile {
						context.SetEnabled(&p.buttons[i], false)
					} else {
						context.SetEnabled(&p.buttons[i], true)
					}
				case 1: // Update
					if !gFResult.gameFile {
						context.SetEnabled(&p.buttons[i], false)
					} else {
						if !cache.IsValid() {
							context.SetEnabled(&p.buttons[i], false)
						} else {
							if data.File().GameVersion == "" && !cache.IsValid() {
								context.SetEnabled(&p.buttons[i], false)
							} else {
								if result, uValue := p86l.CheckNewerVersion(data.File().GameVersion, cache.File().Repo.GetTagName()); !result.Ok {
									context.SetEnabled(&p.buttons[i], false)
									dm.SetToast(result.Err, pd.FileManager)
								} else {
									if uValue {
										context.SetEnabled(&p.buttons[i], true)
									} else {
										context.SetEnabled(&p.buttons[i], false)
									}
								}
							}
						}
					}
				case 2: // Play
					if gFResult.gameFile {
						context.SetEnabled(&p.buttons[i], true)
					} else {
						context.SetEnabled(&p.buttons[i], false)
					}
				}
			}
		case lRResult := <-p.lRResult: // Handle launcher update.
			// TODO: Move to once?
			// Since we gonna check for updates once, when the app/play page is opened only.
			if !lRResult.result.Ok {
				context.SetEnabled(&p.buttons[3], false)
				dm.SetToast(lRResult.result.Err, pd.NetworkManager)
			} else {
				if launcherVersion := am.PlainVersion(); launcherVersion == "dev" {
					context.SetEnabled(&p.buttons[3], false)
				} else {
					if result, lValue := p86l.CheckNewerVersion(launcherVersion, lRResult.release.GetTagName()); !result.Ok {
						context.SetEnabled(&p.buttons[3], false)
						dm.SetToast(result.Err, pd.FileManager)
					} else {
						if lValue {
							context.SetEnabled(&p.buttons[3], true)
						} else {
							context.SetEnabled(&p.buttons[3], false)
						}
					}
				}
			}
		default:

		}
	}

	for i := range p.buttons {
		if p.progress {
			context.SetEnabled(&p.buttons[i], false)
		}
	}

	if !p.gFProgress {
		p.gFProgress = true
		go p.handleGameFile(model)
	}

	var (
		u         = basicwidget.UnitSize(context)
		widths    []layout.Size
		heights   []layout.Size
		positions [4]breakWidget
	)

	switch {
	case breakSize(context, 900):
		widths = []layout.Size{
			layout.FlexibleSize(1),
			layout.FlexibleSize(1),
			layout.FixedSize(u / 2),
			layout.FlexibleSize(1),
			layout.FixedSize(u / 2),
			layout.FlexibleSize(1),
			layout.FixedSize(u / 2),
			layout.FlexibleSize(1),
			layout.FlexibleSize(1),
		}
		heights = []layout.Size{
			layout.FixedSize(u * 2),
		}
		positions = [4]breakWidget{
			{1, 0},
			{3, 0},
			{5, 0},
			{7, 0},
		}
	case breakSize(context, 700):
		widths = []layout.Size{
			layout.FlexibleSize(1),
			layout.FlexibleSize(1),
			layout.FixedSize(u / 2),
			layout.FlexibleSize(1),
			layout.FlexibleSize(1),
		}
		heights = []layout.Size{
			layout.FixedSize(int(float64(u) * 1.5)),
			layout.FixedSize(int(float64(u) * 1.5)),
		}
		positions = [4]breakWidget{
			{1, 0},
			{1, 1},
			{3, 0},
			{3, 1},
		}
	default:
		widths = []layout.Size{
			layout.FlexibleSize(1),
		}
		heights = []layout.Size{
			layout.FixedSize(u),
			layout.FixedSize(u),
			layout.FixedSize(u),
			layout.FixedSize(u),
		}
		positions = [4]breakWidget{
			{0, 0},
			{0, 1},
			{0, 2},
			{0, 3},
		}
	}

	gl := layout.GridLayout{
		Bounds:  context.Bounds(p),
		Widths:  widths,
		Heights: heights,
		RowGap:  u / 2,
	}

	switch {
	case breakSize(context, 900):
		p.height = gl.CellBounds(positions[0].Get()).Dy()
	case breakSize(context, 700):
		p.height = gl.CellBounds(positions[0].Get()).Dy()*2 + u/2
	default:
		p.height = gl.CellBounds(positions[0].Get()).Dy()*4 + int(float64(u)*1.5)
	}

	for i := range p.buttons {
		bounds := gl.CellBounds(positions[i].Get())
		context.SetBounds(&p.buttons[i], bounds, p)
	}

	return nil
}

func (p *playButtons) Height() int {
	return p.height
}

type playLinks struct {
	guigui.DefaultWidget

	buttons [4]basicwidget.Button

	height int
}

func (p *playLinks) AppendChildWidgets(context *guigui.Context, appender *guigui.ChildWidgetAppender) {
	for i := range p.buttons {
		appender.AppendChildWidget(&p.buttons[i])
	}
}

func (p *playLinks) Build(context *guigui.Context) error {
	model := context.Model(p, modelKeyModel).(*p86l.Model)
	am := model.App()
	dm := am.Debug()

	buttonIcons := []string{"ie", "github", "discord", "patreon"}
	buttonTexts := []string{"play.website", "play.github", "play.discord", "play.patreon"}
	buttonActions := []func(){
		func() { go p86l.OpenBrowser(dm, configs.Website) },
		func() { go p86l.OpenBrowser(dm, configs.Github) },
		func() { go p86l.OpenBrowser(dm, configs.Discord) },
		func() { go p86l.OpenBrowser(dm, configs.Patreon) },
	}

	for i := range p.buttons {
		img, err := assets.TheImageCache.Get(buttonIcons[i])
		if err != nil {
			am.SetError(pd.NotOk(err))
			return err.Error()
		}

		p.buttons[i].SetIcon(img)
		p.buttons[i].SetText(am.T(buttonTexts[i]))
		p.buttons[i].SetOnDown(buttonActions[i])
	}

	var (
		u         = basicwidget.UnitSize(context)
		widths    []layout.Size
		heights   []layout.Size
		positions [4]breakWidget
	)

	switch {
	case breakSize(context, 1024):
		widths = []layout.Size{
			layout.FlexibleSize(1),
			layout.FlexibleSize(1),
			layout.FlexibleSize(1),
			layout.FlexibleSize(1),
		}
		heights = []layout.Size{
			layout.FixedSize(u * 2),
		}
		positions = [4]breakWidget{
			{0, 0},
			{1, 0},
			{2, 0},
			{3, 0},
		}
	case breakSize(context, 640):
		widths = []layout.Size{
			layout.FlexibleSize(1),
			layout.FixedSize(u / 2),
			layout.FlexibleSize(1),
		}
		heights = []layout.Size{
			layout.FixedSize(u * 2),
			layout.FixedSize(u * 2),
		}
		positions = [4]breakWidget{
			{0, 0},
			{0, 1},
			{2, 0},
			{2, 1},
		}
	default:
		widths = []layout.Size{
			layout.FlexibleSize(1),
		}
		heights = []layout.Size{
			layout.FixedSize(int(float64(u) * 1.5)),
			layout.FixedSize(int(float64(u) * 1.5)),
			layout.FixedSize(int(float64(u) * 1.5)),
			layout.FixedSize(int(float64(u) * 1.5)),
		}
		positions = [4]breakWidget{
			{0, 0},
			{0, 1},
			{0, 2},
			{0, 3},
		}
	}

	doColumn := func() int {
		if breakSize(context, 1024) {
			return u / 2
		}
		return 0
	}

	gl := layout.GridLayout{
		Bounds:    context.Bounds(p),
		Widths:    widths,
		Heights:   heights,
		ColumnGap: doColumn(),
		RowGap:    u / 2,
	}

	switch {
	case breakSize(context, 1024):
		p.height = gl.CellBounds(positions[0].Get()).Dy()
	case breakSize(context, 640):
		p.height = gl.CellBounds(positions[0].Get()).Dy()*2 + u/2
	default:
		p.height = gl.CellBounds(positions[0].Get()).Dy()*4 + int(float64(u)*1.5)
	}

	for i := range p.buttons {
		bounds := gl.CellBounds(positions[i].Get())
		context.SetBounds(&p.buttons[i], bounds, p)
	}

	return nil
}

func (p *playLinks) Height() int {
	return p.height
}
