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
	"fmt"
	"p86l"
	"p86l/assets"
	pd "p86l/internal/debug"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/hajimehoshi/guigui"
	"github.com/hajimehoshi/guigui/basicwidget"
	"github.com/hajimehoshi/guigui/layout"
)

type Home struct {
	guigui.DefaultWidget

	content homeContent

	box basicwidget.Background
}

func (h *Home) AppendChildWidgets(context *guigui.Context, appender *guigui.ChildWidgetAppender) {
	model := context.Model(h, modelKeyModel).(*p86l.Model)
	am := model.App()

	am.RenderBox(appender, &h.box)
	appender.AppendChildWidget(&h.content)
}

func (h *Home) Build(context *guigui.Context) error {
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
	context.SetBounds(&h.box, gl.CellBounds(0, 0), h)
	context.SetBounds(&h.content, gl.CellBounds(0, 1), h)

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

	form2          basicwidget.Form
	playTimeText   basicwidget.Text
	playCountText  basicwidget.Text
	lastPlayedText basicwidget.Text
	lastTimeText   basicwidget.Text

	form3         basicwidget.Form
	downloadsText basicwidget.Text
	countText     basicwidget.Text
	versionText   basicwidget.Text
	latestText    basicwidget.Text

	box1   basicwidget.Background
	box2   basicwidget.Background
	height int
}

func (h *homeContent) AppendChildWidgets(context *guigui.Context, appender *guigui.ChildWidgetAppender) {
	model := context.Model(h, modelKeyModel).(*p86l.Model)
	am := model.App()

	am.RenderBox(appender, &h.box1)
	am.RenderBox(appender, &h.box2)
	appender.AppendChildWidget(&h.background)
	appender.AppendChildWidget(&h.p86lImage)
	appender.AppendChildWidget(&h.form1)
	appender.AppendChildWidget(&h.form2)
	appender.AppendChildWidget(&h.form3)
}

func (h *homeContent) Build(context *guigui.Context) error {
	model := context.Model(h, modelKeyModel).(*p86l.Model)
	am := model.App()
	data := model.Data()
	cache := model.Cache()

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

	h.playTimeText.SetValue(am.T("home.playtime"))
	h.playCountText.SetValue(fmt.Sprintf("%d", data.File().PlayTime))
	h.lastPlayedText.SetValue(am.T("home.lastplayed"))
	if data.File().PlayTime == 0 {
		h.lastTimeText.SetValue("")
	} else {
		h.lastTimeText.SetValue(humanize.RelTime(time.Now(), data.File().LastPlayed, "before", "ago"))
	}

	h.form2.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &h.playTimeText,
			SecondaryWidget: &h.playCountText,
		},
		{
			PrimaryWidget:   &h.lastPlayedText,
			SecondaryWidget: &h.lastTimeText,
		},
	})

	h.downloadsText.SetValue(am.T("home.downloads"))
	h.versionText.SetValue(am.T("home.version"))

	if cache.IsValid() {
		cacheAssets := cache.File().Repo.Assets
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

	h.form3.SetItems([]basicwidget.FormItem{
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
			layout.FlexibleSize(2),
			layout.FlexibleSize(2),
			layout.FlexibleSize(3),
		},
		Heights: []layout.Size{
			layout.FlexibleSize(1),
		},
	}
	context.SetBounds(&h.box1, gl.CellBounds(1, 0).Inset(u/2), h)
	context.SetBounds(&h.box2, gl.CellBounds(2, 0).Inset(u/2), h)
	h.height = max(h.form1.DefaultSizeInContainer(context, context.Bounds(h).Dx()-u).Y, h.form2.DefaultSizeInContainer(context, context.Bounds(h).Dx()).Y, h.form3.DefaultSizeInContainer(context, context.Bounds(h).Dx()).Y)
	context.SetBounds(&h.background, context.Bounds(h), h)
	context.SetBounds(&h.p86lImage, gl.CellBounds(0, 0), h)
	context.SetBounds(&h.form1, gl.CellBounds(1, 0).Inset(u/2), h)
	context.SetBounds(&h.form2, gl.CellBounds(2, 0).Inset(u/2), h)
	context.SetBounds(&h.form3, gl.CellBounds(3, 0).Inset(u/2), h)

	return nil
}

func (h *homeContent) Height() int {
	return h.height
}
