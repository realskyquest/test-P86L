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
	"image"
	"p86l"
	"p86l/assets"

	"github.com/hajimehoshi/guigui"
	"github.com/hajimehoshi/guigui/basicwidget"
	"github.com/hajimehoshi/guigui/layout"
)

type About struct {
	guigui.DefaultWidget

	panel       basicwidget.Panel
	content     aboutContent
	licenseText basicwidget.Text

	box   basicwidget.Background
	model *p86l.Model
}

func (a *About) SetModel(m *p86l.Model) {
	a.model = m
	a.content.model = m
}

func (a *About) Build(context *guigui.Context, appender *guigui.ChildWidgetAppender) error {
	am := a.model.App()

	a.panel.SetAutoBorder(true)
	context.SetSize(&a.content, image.Pt(context.ActualSize(a).X, a.content.Height()), a)
	a.panel.SetContent(&a.content)

	a.licenseText.SetValue(am.License())
	a.licenseText.SetAutoWrap(true)
	a.licenseText.SetScale(0.7)
	a.licenseText.SetHorizontalAlign(basicwidget.HorizontalAlignCenter)
	a.licenseText.SetVerticalAlign(basicwidget.VerticalAlignBottom)
	context.SetOpacity(&a.licenseText, 0.7)

	u := basicwidget.UnitSize(context)
	gl := layout.GridLayout{
		Bounds: context.Bounds(a),
		Widths: []layout.Size{
			layout.FlexibleSize(1),
		},
		Heights: []layout.Size{
			layout.FlexibleSize(1),
			layout.FixedSize(int(float64(a.licenseText.DefaultSizeInContainer(context, context.Bounds(a).Dx()-u).Y) * 0.7)),
		},
	}
	am.RenderBox(appender, &a.box, gl.CellBounds(0, 1))
	appender.AppendChildWidgetWithBounds(&a.panel, gl.CellBounds(0, 0))
	appender.AppendChildWidgetWithBounds(&a.licenseText, gl.CellBounds(0, 1))

	return nil
}

type aboutContent struct {
	guigui.DefaultWidget

	aboutText basicwidget.Text
	form      basicwidget.Form
	leadText  basicwidget.Text
	devText   basicwidget.Text
	leadImage basicwidget.Image
	devImage  basicwidget.Image

	box1   basicwidget.Background
	box2   basicwidget.Background
	height int
	model  *p86l.Model
}

func (a *aboutContent) Build(context *guigui.Context, appender *guigui.ChildWidgetAppender) error {
	am := a.model.App()

	img1, err1 := assets.TheImageCache.Get("lead")
	img2, err2 := assets.TheImageCache.Get("dev")

	if err := cmp.Or(err1, err2); err != nil {
		am.SetError(err)
		return err.Error()
	}

	a.aboutText.SetValue(am.T("about.info"))
	a.aboutText.SetAutoWrap(true)
	a.aboutText.SetHorizontalAlign(basicwidget.HorizontalAlignCenter)
	a.aboutText.SetVerticalAlign(basicwidget.VerticalAlignMiddle)

	u := basicwidget.UnitSize(context)

	a.leadImage.SetImage(img1)
	context.SetSize(&a.leadImage, image.Pt(u*4, u*4), a)

	a.devImage.SetImage(img2)
	context.SetSize(&a.devImage, image.Pt(u*4, u*4), a)

	a.leadText.SetValue(am.T("about.lead"))
	a.leadText.SetScale(1.4)

	a.devText.SetValue(am.T("about.dev"))
	a.devText.SetScale(1.4)

	a.form.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &a.leadImage,
			SecondaryWidget: &a.leadText,
		},
		{
			PrimaryWidget:   &a.devImage,
			SecondaryWidget: &a.devText,
		},
	})

	gl := layout.GridLayout{
		Bounds: context.Bounds(a).Inset(u / 2),
		Widths: []layout.Size{
			layout.FlexibleSize(1),
		},
		Heights: []layout.Size{
			layout.FixedSize(a.aboutText.DefaultSizeInContainer(context, context.Bounds(a).Dx()-u).Y),
			layout.FixedSize(a.form.DefaultSizeInContainer(context, context.Bounds(a).Dx()-u).Y),
		},
		RowGap: u / 2,
	}
	am.RenderBox(appender, &a.box1, gl.CellBounds(0, 0))
	am.RenderBox(appender, &a.box2, gl.CellBounds(0, 1))
	a.height = gl.CellBounds(0, 0).Dy() + gl.CellBounds(0, 1).Dy() + u*2
	appender.AppendChildWidgetWithBounds(&a.aboutText, gl.CellBounds(0, 0))
	appender.AppendChildWidgetWithBounds(&a.form, gl.CellBounds(0, 1))

	return nil
}

func (a *aboutContent) Height() int {
	return a.height
}
