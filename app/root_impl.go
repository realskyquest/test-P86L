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
	pd "p86l/internal/debug"
	"sync"

	"github.com/hajimehoshi/guigui"
	"github.com/hajimehoshi/guigui/basicwidget"
	"github.com/hajimehoshi/guigui/layout"
)

type breakWidget struct {
	column int // Width
	row    int // Height
}

func (b *breakWidget) Get() (column, row int) {
	return b.column, b.row
}

// Width, Height
func (b *breakWidget) Set(column, row int) {
	b.column = column
	b.row = row
}

func breakSize(context *guigui.Context, size int) bool {
	scaledWidth := int(float64(context.AppSize().X) / context.AppScale())
	return scaledWidth > size
}

type rootBackground struct {
	guigui.DefaultWidget

	bgImage    basicwidget.Image
	background basicwidget.Background

	sidebar  *Sidebar
	bgBounds image.Rectangle

	sync   sync.Once
	result pd.Result
}

func (r *rootBackground) SetSidebar(sidebar *Sidebar) {
	r.sidebar = sidebar
}

func (r *rootBackground) SetBgBounds(bounds image.Rectangle) {
	r.bgBounds = bounds
}

func (r *rootBackground) AppendChildWidgets(context *guigui.Context, appender *guigui.ChildWidgetAppender) {
	model := context.Model(r, modelKeyModel).(*p86l.Model)
	appender.AppendChildWidget(&r.bgImage)
	if model.Mode() != "home" {
		appender.AppendChildWidget(&r.background)
	}
}

func (r *rootBackground) Build(context *guigui.Context) error {
	model := context.Model(r, modelKeyModel).(*p86l.Model)
	am := model.App()

	r.sync.Do(func() {
		r.result = pd.Ok()
	})

	img, err := assets.TheImageCache.Get("banner")
	if err != nil {
		r.result = pd.NotOk(err)
	}

	if !r.result.Ok {
		am.SetError(r.result)
		return r.result.Err.Error()
	}

	r.bgImage.SetImage(img)
	context.SetOpacity(&r.background, 0.9)

	imgWidth := img.Bounds().Dx()
	imgHeight := img.Bounds().Dy()
	aspectRatio := float64(imgHeight) / float64(imgWidth)

	windowSize := context.ActualSize(r)
	availableWidth := windowSize.X

	newHeight := int(float64(availableWidth) * aspectRatio)

	if newHeight < windowSize.Y {
		newHeight = windowSize.Y
		availableWidth = int(float64(newHeight) / aspectRatio)
	}

	context.SetSize(&r.bgImage, image.Pt(availableWidth, newHeight), r)

	yOffset := 0
	if newHeight > windowSize.Y {
		yOffset = -(newHeight - windowSize.Y) / 2
	}

	imgPosition := image.Pt(00, yOffset)
	context.SetPosition(&r.bgImage, imgPosition)
	if model.Mode() != "home" {
		context.SetBounds(&r.background, r.bgBounds, r)
	}

	return nil
}

type rootPopupContent struct {
	guigui.DefaultWidget

	popup *basicwidget.Popup

	titleText   basicwidget.Text
	closeButton basicwidget.Button
}

func (r *rootPopupContent) AppendChildWidgets(context *guigui.Context, appender *guigui.ChildWidgetAppender) {
	appender.AppendChildWidget(&r.titleText)
	appender.AppendChildWidget(&r.closeButton)
}

func (r *rootPopupContent) Build(context *guigui.Context) error {
	model := context.Model(r, modelKeyModel).(*p86l.Model)
	dm := model.App().Debug()
	u := basicwidget.UnitSize(context)

	r.titleText.SetValue(dm.Popup().String())
	r.titleText.SetAutoWrap(true)
	r.titleText.SetBold(true)
	r.titleText.SetSelectable(true)

	r.closeButton.SetText("Close")
	r.closeButton.SetOnUp(func() {
		r.popup.Close()
	})

	gl := layout.GridLayout{
		Bounds: context.Bounds(r).Inset(u / 2),
		Heights: []layout.Size{
			layout.FlexibleSize(1),
			layout.LazySize(func(row int) layout.Size {
				if row != 1 {
					return layout.FixedSize(0)
				}
				return layout.FixedSize(r.closeButton.DefaultSize(context).Y)
			}),
		},
	}
	context.SetBounds(&r.titleText, gl.CellBounds(0, 0), r)
	{
		gl := layout.GridLayout{
			Bounds: gl.CellBounds(0, 1),
			Widths: []layout.Size{
				layout.FlexibleSize(1),
				layout.FixedSize(r.closeButton.DefaultSize(context).X),
			},
		}
		context.SetBounds(&r.closeButton, gl.CellBounds(1, 0), r)
	}

	return nil
}

