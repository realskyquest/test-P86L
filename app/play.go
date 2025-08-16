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
	})

	if !p.result.Ok {
		am.SetError(p.result)
		return p.result.Err.Error()
	}

	p.prereleaseText.SetValue(am.T("settings.prerelease"))
	p.prereleaseToggle.SetOnValueChanged(func(value bool) {
		if value == data.File().UsePreRelease {
			return
		}
		data.SetUsePreRelease(dm, value)
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
	height  int
}

func (p *playButtons) AppendChildWidgets(context *guigui.Context, appender *guigui.ChildWidgetAppender) {
	for i := range p.buttons {
		appender.AppendChildWidget(&p.buttons[i])
	}
}

func (p *playButtons) Build(context *guigui.Context) error {
	model := context.Model(p, modelKeyModel).(*p86l.Model)
	am := model.App()
	//dm := am.Debug()
	data := model.Data()
	//cache := model.Cache()
	play := model.Play()

	buttonTexts := []string{"play.install", "play.update", "play.play", "play.launcher"}
	buttonActions := []func(){
		func() { go play.HandleGameDownload(model, "handleInstall") },
		func() { go play.HandleGameDownload(model, "handleUpdate") },
		func() { go play.HandlePlay(model) },
		func() { go play.HandlePlay(model) },
	}

	for i := range p.buttons {
		// Text
		if t := buttonTexts[i]; t == "play.install" {
			if data.File().UsePreRelease {
				p.buttons[i].SetText(am.T("play.prerelease"))
			} else {
				p.buttons[i].SetText(am.T(t))
			}
		} else {
			p.buttons[i].SetText(am.T(t))
		}
		// Actions
		p.buttons[i].SetOnDown(buttonActions[i])

		for range 1 {
			select {
			case availableResult := <-play.GameAvailable().Available:
				play.SetGameAvailable(model, false, false)
				context.SetEnabled(&p.buttons[0], !availableResult) // Install
				context.SetEnabled(&p.buttons[2], availableResult)  // Play

				if !availableResult { // Update
					context.SetEnabled(&p.buttons[1], false)
				} else {
					canUpdate := play.CanUpdate(model)
					context.SetEnabled(&p.buttons[1], canUpdate)
				}
			default:

			}
		}

		if !play.CanInteract() {
			context.SetEnabled(&p.buttons[i], false)
		}
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
	case breakSize(context, 600):
		widths = []layout.Size{
			layout.FlexibleSize(1),
			layout.FlexibleSize(1),
			layout.FixedSize(u / 2),
			layout.FlexibleSize(1),
			layout.FlexibleSize(1),
		}
		heights = []layout.Size{
			layout.FixedSize(u * 2),
			layout.FixedSize(u * 2),
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
	case breakSize(context, 600):
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
		p.buttons[i].SetOnDown(buttonActions[i])
	}

	u := basicwidget.UnitSize(context)
	gl := layout.GridLayout{
		Bounds: context.Bounds(p),
		Widths: []layout.Size{
			layout.FixedSize(u * 2),
			layout.FixedSize(u * 2),
			layout.FixedSize(u * 2),
			layout.FixedSize(u * 2),
			layout.FlexibleSize(1),
		},
		Heights: []layout.Size{
			layout.FixedSize(u * 2),
			layout.FixedSize(u * 2),
			layout.FixedSize(u * 2),
			layout.FixedSize(u * 2),
		},
		ColumnGap: u / 2,
	}

	p.height = u * 2
	for i := range p.buttons {
		bounds := gl.CellBounds(i, 0)
		context.SetBounds(&p.buttons[i], bounds, p)
	}

	return nil
}

func (p *playLinks) Height() int {
	return p.height
}
