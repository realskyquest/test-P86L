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
	"p86l"
	"p86l/assets"
	pd "p86l/internal/debug"

	"github.com/dustin/go-humanize"
	"github.com/hajimehoshi/guigui"
	"github.com/hajimehoshi/guigui/basicwidget"
	"github.com/hajimehoshi/guigui/layout"
)

type Home struct {
	guigui.DefaultWidget

	content homeContent

	box   basicwidget.Background
	model *p86l.Model
}

func (h *Home) SetModel(model *p86l.Model) {
	h.model = model
	h.content.model = model
}

func (h *Home) Build(context *guigui.Context, appender *guigui.ChildWidgetAppender) error {
	am := h.model.App()

	u := basicwidget.UnitSize(context)
	gl := layout.GridLayout{
		Bounds: context.Bounds(h),
		Widths: []layout.Size{
			layout.FlexibleSize(1),
		},
		Heights: []layout.Size{
			layout.FlexibleSize(1),
			layout.FixedSize(h.content.Height() + u),
		},
	}
	am.RenderBox(appender, &h.box, gl.CellBounds(0, 0))
	appender.AppendChildWidgetWithBounds(&h.content, gl.CellBounds(0, 1))

	return nil
}

type homeContent struct {
	guigui.DefaultWidget

	background basicwidget.Background
	p86lImage  basicwidget.Image

	form1           basicwidget.Form
	welcomeText     basicwidget.Text
	usernameText    basicwidget.Text
	downloadedText  basicwidget.Text
	gameVersionText basicwidget.Text

	form2         basicwidget.Form
	downloadsText basicwidget.Text
	countText     basicwidget.Text
	versionText   basicwidget.Text
	latestText    basicwidget.Text

	box1   basicwidget.Background
	box2   basicwidget.Background
	height int
	model  *p86l.Model
}

func (h *homeContent) Build(context *guigui.Context, appender *guigui.ChildWidgetAppender) error {
	am := h.model.App()
	data := h.model.Data()
	cache := h.model.Cache()
	cacheAssets := cache.File().Repo.Assets

	img, err := assets.TheImageCache.Get("p86l")

	if err != nil {
		am.SetError(pd.NotOk(err))
		return err.Error()
	}

	context.SetOpacity(&h.background, 0.9)
	h.p86lImage.SetImage(img)

	h.welcomeText.SetValue(am.T("home.welcome"))
	h.usernameText.SetValue(p86l.GetUsername())

	h.downloadedText.SetValue(am.T("home.downloaded"))
	if version := data.File().GameVersion; version == "" {
		h.gameVersionText.SetValue("")
	} else {
		h.gameVersionText.SetValue(version)
	}

	h.form1.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &h.welcomeText,
			SecondaryWidget: &h.usernameText,
		},
		{
			PrimaryWidget:   &h.downloadedText,
			SecondaryWidget: &h.gameVersionText,
		},
	})

	h.downloadsText.SetValue(am.T("home.downloads"))
	h.versionText.SetValue(am.T("home.version"))

	if cache.IsValid() {
		for _, asset := range cacheAssets {
			if name := asset.GetName(); p86l.IsValidGameFile(name) {
				h.countText.SetValue(humanize.FormatInteger("#,###.", asset.GetDownloadCount()))
				break
			}
		}
		h.latestText.SetValue(cache.File().Repo.GetTagName())
	} else {
		h.countText.SetValue("...")
		h.latestText.SetValue("...")
	}

	h.form2.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &h.downloadsText,
			SecondaryWidget: &h.countText,
		},
		{
			PrimaryWidget:   &h.versionText,
			SecondaryWidget: &h.latestText,
		},
	})

	u := basicwidget.UnitSize(context)
	gl := layout.GridLayout{
		Bounds: context.Bounds(h),
		Widths: []layout.Size{
			layout.FixedSize(u*3 - (u / 2)),
			layout.FlexibleSize(1),
			layout.FlexibleSize(1),
		},
		Heights: []layout.Size{
			layout.FlexibleSize(1),
		},
	}
	am.RenderBox(appender, &h.box1, gl.CellBounds(1, 0).Inset(u/2))
	am.RenderBox(appender, &h.box2, gl.CellBounds(2, 0).Inset(u/2))
	h.height = max(h.form1.DefaultSizeInContainer(context, context.Bounds(h).Dx()-u).Y, h.form2.DefaultSizeInContainer(context, context.Bounds(h).Dx()).Y)
	appender.AppendChildWidgetWithBounds(&h.background, context.Bounds(h))
	appender.AppendChildWidgetWithBounds(&h.p86lImage, gl.CellBounds(0, 0))
	appender.AppendChildWidgetWithBounds(&h.form1, gl.CellBounds(1, 0).Inset(u/2))
	appender.AppendChildWidgetWithBounds(&h.form2, gl.CellBounds(2, 0).Inset(u/2))

	return nil
}

func (h *homeContent) Height() int {
	return h.height
}
