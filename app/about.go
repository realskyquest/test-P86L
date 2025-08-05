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
	pd "p86l/internal/debug"
	"strings"

	"github.com/hajimehoshi/guigui"
	"github.com/hajimehoshi/guigui/basicwidget"
	"github.com/hajimehoshi/guigui/layout"
)

type About struct {
	guigui.DefaultWidget

	panel   basicwidget.Panel
	content aboutContent

	model *p86l.Model
}

func (a *About) SetModel(m *p86l.Model) {
	a.model = m
	a.content.model = m
}

func (a *About) Build(context *guigui.Context, appender *guigui.ChildWidgetAppender) error {
	bounds := context.Bounds(a)
	contentHeight := a.content.Height()

	var contentSize image.Point
	if bounds.Dy() > contentHeight {
		contentSize = image.Pt(bounds.Dx(), bounds.Dy())
	} else {
		contentSize = image.Pt(bounds.Dx(), contentHeight)
	}
	context.SetSize(&a.content, contentSize, a)
	a.panel.SetContent(&a.content)

	appender.AppendChildWidgetWithBounds(&a.panel, context.Bounds(a))

	return nil
}

type aboutContent struct {
	guigui.DefaultWidget

	aboutText   basicwidget.Text
	form        basicwidget.Form
	leadText    basicwidget.Text
	devText     basicwidget.Text
	leadImage   basicwidget.Image
	devImage    basicwidget.Image
	licenseText basicwidget.Text

	box1   basicwidget.Background
	box2   basicwidget.Background
	box3   basicwidget.Background
	height int
	model  *p86l.Model
}

func (a *aboutContent) Build(context *guigui.Context, appender *guigui.ChildWidgetAppender) error {
	am := a.model.App()

	img1, err1 := assets.TheImageCache.Get("lead")
	img2, err2 := assets.TheImageCache.Get("dev")

	if err := cmp.Or(err1, err2); err != nil {
		am.SetError(pd.NotOk(err))
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

	a.licenseText.SetValue(strings.Join(strings.Fields(am.License()), " "))
	a.licenseText.SetAutoWrap(true)
	a.licenseText.SetScale(0.7)
	a.licenseText.SetHorizontalAlign(basicwidget.HorizontalAlignCenter)
	context.SetOpacity(&a.licenseText, 0.7)

	gl := layout.GridLayout{
		Bounds: context.Bounds(a).Inset(u / 2),
		Widths: []layout.Size{
			layout.FlexibleSize(1),
		},
		Heights: []layout.Size{
			layout.FixedSize(a.aboutText.DefaultSizeInContainer(context, context.Bounds(a).Dx()-u).Y),
			layout.FixedSize(a.form.DefaultSizeInContainer(context, context.Bounds(a).Dx()-u).Y),
			layout.FlexibleSize(1),
			layout.FixedSize(a.licenseText.DefaultSizeInContainer(context, context.Bounds(a).Dx()-u).Y),
		},
		RowGap: u / 2,
	}
	a.height = gl.CellBounds(0, 0).Dy() + gl.CellBounds(0, 1).Dy() + gl.CellBounds(0, 3).Dy() + u*2
	am.RenderBox(appender, &a.box1, gl.CellBounds(0, 0))
	am.RenderBox(appender, &a.box2, gl.CellBounds(0, 1))
	am.RenderBox(appender, &a.box3, gl.CellBounds(0, 3))
	appender.AppendChildWidgetWithBounds(&a.aboutText, gl.CellBounds(0, 0))
	appender.AppendChildWidgetWithBounds(&a.form, gl.CellBounds(0, 1))
	appender.AppendChildWidgetWithBounds(&a.licenseText, gl.CellBounds(0, 3))

	return nil
}

func (a *aboutContent) Height() int {
	return a.height
}
