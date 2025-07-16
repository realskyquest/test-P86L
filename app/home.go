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
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/guigui"
	"github.com/hajimehoshi/guigui/basicwidget"
	"github.com/hajimehoshi/guigui/layout"
)

type Home struct {
	guigui.DefaultWidget

	background basicwidget.Background
	stats      homeStats

	model *p86l.Model
}

func (h *Home) SetModel(model *p86l.Model) {
	h.model = model
}

func (h *Home) Build(context *guigui.Context, appender *guigui.ChildWidgetAppender) error {
	h.stats.SetModel(h.model)
	context.SetOpacity(&h.background, 0.9)

	u := basicwidget.UnitSize(context)
	gl := layout.GridLayout{
		Bounds: context.Bounds(h),
		Widths: []layout.Size{
			layout.FlexibleSize(1),
		},
		Heights: []layout.Size{
			layout.FlexibleSize(1),
			layout.FixedSize(u*4 - (u / 4)),
		},
	}
	appender.AppendChildWidgetWithBounds(&h.background, gl.CellBounds(0, 1))
	appender.AppendChildWidgetWithBounds(&h.stats, gl.CellBounds(0, 1))

	return nil
}

type homeStats struct {
	guigui.DefaultWidget

	image basicwidget.Image

	form1           basicwidget.Form
	welcomeText     basicwidget.Text
	welcomeStatText basicwidget.Text

	form2             basicwidget.Form
	downloadsText     basicwidget.Text
	downloadsStatText basicwidget.Text
	versionText       basicwidget.Text
	versionStatText   basicwidget.Text

	model *p86l.Model

	cachedImage *ebiten.Image

	sync sync.Once
	err  *pd.Error
}

func (h *homeStats) SetModel(model *p86l.Model) {
	h.model = model
}

func (h *homeStats) Build(context *guigui.Context, appender *guigui.ChildWidgetAppender) error {
	h.sync.Do(func() {
		images, err := assets.GetIconImages(p86l.E)
		if err != nil {
			h.err = err
		}
		h.cachedImage = ebiten.NewImageFromImage(images[len(images)-1])
	})

	if h.err != nil {
		p86l.GErr = h.err
		return h.err.Err
	}

	cache := h.model.Cache()
	cacheAssets := cache.File().Repo.Assets

	h.image.SetImage(h.cachedImage)

	h.welcomeText.SetValue(p86l.T("home.welcome"))
	h.welcomeStatText.SetValue(p86l.GetUsername())

	h.form1.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget: &h.welcomeText,
		},
		{
			SecondaryWidget: &h.welcomeStatText,
		},
	})

	// --

	h.downloadsText.SetValue(p86l.T("home.downloads"))
	h.versionText.SetValue(p86l.T("home.version"))

	if cache.IsValid() {
		for _, asset := range cacheAssets {
			if name := asset.GetName(); p86l.IsValidGameFile(name) {
				h.downloadsStatText.SetValue(fmt.Sprintf("%d", asset.GetDownloadCount()))
				break
			}
		}
		h.versionStatText.SetValue(cache.File().Repo.GetTagName())
	} else {
		h.downloadsStatText.SetValue("...")
		h.versionStatText.SetValue("...")
	}

	h.form2.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &h.downloadsText,
			SecondaryWidget: &h.downloadsStatText,
		},
		{
			PrimaryWidget:   &h.versionText,
			SecondaryWidget: &h.versionStatText,
		},
	})

	u := basicwidget.UnitSize(context)
	gl := layout.GridLayout{
		Bounds: context.Bounds(h),
		Widths: []layout.Size{
			layout.FixedSize(u*3 - (u / 2)),
			layout.FixedSize(max(h.welcomeText.DefaultSize(context).X, h.welcomeStatText.DefaultSize(context).X) + u),
			layout.FixedSize(u * 8),
			layout.FlexibleSize(1),
		},
		Heights: []layout.Size{
			layout.FixedSize(u / 2),
			layout.FlexibleSize(1),
		},
		ColumnGap: u / 2,
	}
	appender.AppendChildWidgetWithBounds(&h.image, gl.CellBounds(0, 1))
	appender.AppendChildWidgetWithBounds(&h.form1, gl.CellBounds(1, 1))
	appender.AppendChildWidgetWithBounds(&h.form2, gl.CellBounds(2, 1))

	return nil
}
