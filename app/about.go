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
	"p86l"
	"p86l/assets"

	"github.com/hajimehoshi/guigui"
	"github.com/hajimehoshi/guigui/basicwidget"
	"github.com/hajimehoshi/guigui/layout"
)

type About struct {
	guigui.DefaultWidget

	aboutText   basicwidget.Text
	credits     aboutCredits
	licenseText basicwidget.Text

	model *p86l.Model
}

func (a *About) SetModel(m *p86l.Model) {
	a.model = m
	a.credits.model = m
}

func (a *About) Build(context *guigui.Context, appender *guigui.ChildWidgetAppender) error {
	am := a.model.App()

	a.aboutText.SetValue(am.T("about.info"))
	a.aboutText.SetAutoWrap(true)
	a.aboutText.SetHorizontalAlign(basicwidget.HorizontalAlignCenter)
	a.aboutText.SetVerticalAlign(basicwidget.VerticalAlignMiddle)

	a.licenseText.SetValue(am.License())
	a.licenseText.SetAutoWrap(true)
	a.licenseText.SetScale(0.7)
	a.licenseText.SetHorizontalAlign(basicwidget.HorizontalAlignCenter)
	a.licenseText.SetVerticalAlign(basicwidget.VerticalAlignBottom)
	context.SetOpacity(&a.licenseText, 0.7)

	u := basicwidget.UnitSize(context)
	gl := layout.GridLayout{
		Bounds: context.Bounds(a).Inset(u / 2),
		Widths: []layout.Size{
			layout.FlexibleSize(1),
		},
		Heights: []layout.Size{
			layout.FixedSize(u * 6),
			layout.FixedSize(u * 5),
			layout.FlexibleSize(1),
		},
		RowGap: u / 2,
	}
	appender.AppendChildWidgetWithBounds(&a.aboutText, gl.CellBounds(0, 0))
	appender.AppendChildWidgetWithBounds(&a.credits, gl.CellBounds(0, 1))
	appender.AppendChildWidgetWithBounds(&a.licenseText, gl.CellBounds(0, 2))

	return nil
}

type aboutCredits struct {
	guigui.DefaultWidget

	leadImg  basicwidget.Image
	devImg   basicwidget.Image
	leadText basicwidget.Text
	devText  basicwidget.Text

	model *p86l.Model
}

func (a *aboutCredits) Build(context *guigui.Context, appender *guigui.ChildWidgetAppender) error {
	am := a.model.App()

	img1, err1 := assets.TheImageCache.Get("lead")
	img2, err2 := assets.TheImageCache.Get("dev")

	if err := cmp.Or(err1, err2); err != nil {
		am.SetError(err)
		return err.Error()
	}

	a.leadImg.SetImage(img1)
	a.devImg.SetImage(img2)

	a.leadText.SetValue(am.T("about.lead"))
	a.leadText.SetScale(1.2)
	a.leadText.SetHorizontalAlign(basicwidget.HorizontalAlignCenter)

	a.devText.SetValue(am.T("about.dev"))
	a.devText.SetScale(1.2)
	a.devText.SetHorizontalAlign(basicwidget.HorizontalAlignCenter)

	var gl layout.GridLayout
	var bLeadText breakWidget
	var bLeadImage breakWidget
	var bDevText breakWidget
	var bDevImage breakWidget

	u := basicwidget.UnitSize(context)

	if breakSize(context, 760) {
		gl = layout.GridLayout{
			Bounds: context.Bounds(a),
			Widths: []layout.Size{
				layout.FlexibleSize(1),
				layout.FixedSize(max(a.leadText.DefaultSize(context).X, a.devText.DefaultSize(context).X)),
				layout.FixedSize(u * 2),
				layout.FixedSize(max(a.leadText.DefaultSize(context).X, a.devText.DefaultSize(context).X)),
				layout.FixedSize(u * 2),
				layout.FlexibleSize(1),
			},
			Heights: []layout.Size{
				layout.FixedSize(u * 2),
			},
			ColumnGap: u,
		}
		bLeadText.Set(1, 0)
		bLeadImage.Set(2, 0)
		bDevText.Set(3, 0)
		bDevImage.Set(4, 0)
	} else {
		gl = layout.GridLayout{
			Bounds: context.Bounds(a),
			Widths: []layout.Size{
				layout.FlexibleSize(1),
				layout.FixedSize(max(a.leadText.DefaultSize(context).X, a.devText.DefaultSize(context).X)),
				layout.FixedSize(u * 2),
				layout.FlexibleSize(1),
			},
			Heights: []layout.Size{
				layout.FixedSize(u * 2),
				layout.FixedSize(u * 2),
			},
			ColumnGap: u,
			RowGap:    u / 2,
		}
		bLeadText.Set(1, 0)
		bLeadImage.Set(2, 0)
		bDevText.Set(1, 1)
		bDevImage.Set(2, 1)
	}
	appender.AppendChildWidgetWithBounds(&a.leadText, gl.CellBounds(bLeadText.Get()))
	appender.AppendChildWidgetWithBounds(&a.leadImg, gl.CellBounds(bLeadImage.Get()))
	appender.AppendChildWidgetWithBounds(&a.devText, gl.CellBounds(bDevText.Get()))
	appender.AppendChildWidgetWithBounds(&a.devImg, gl.CellBounds(bDevImage.Get()))

	return nil
}
