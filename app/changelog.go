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

	"github.com/hajimehoshi/guigui"
	"github.com/hajimehoshi/guigui/basicwidget"
	"github.com/hajimehoshi/guigui/layout"
)

type Changelog struct {
	guigui.DefaultWidget

	panel   basicwidget.Panel
	content changelogContent

	model *p86l.Model
}

func (c *Changelog) SetModel(model *p86l.Model) {
	c.model = model
	c.content.model = model
}

func (c *Changelog) Build(context *guigui.Context, appender *guigui.ChildWidgetAppender) error {
	context.SetSize(&c.content, image.Pt(context.ActualSize(c).X, c.content.Height()), c)
	c.panel.SetContent(&c.content)

	appender.AppendChildWidgetWithBounds(&c.panel, context.Bounds(c))

	return nil
}

type changelogContent struct {
	guigui.DefaultWidget

	text       basicwidget.Text
	form       basicwidget.Form
	viewText   basicwidget.Text
	viewButton basicwidget.Button

	box1   basicwidget.Background
	box2   basicwidget.Background
	height int
	model  *p86l.Model
}

func (c *changelogContent) Build(context *guigui.Context, appender *guigui.ChildWidgetAppender) error {
	am := c.model.App()
	dm := am.Debug()

	c.text.SetHorizontalAlign(basicwidget.HorizontalAlignCenter)
	c.text.SetVerticalAlign(basicwidget.VerticalAlignMiddle)
	c.text.SetAutoWrap(true)

	// IsValid
	if cache := c.model.Cache(); cache.IsValid() {
		// If not english, get translation
		if locale := c.model.Data().File().Locale; locale != "en" {
			// If there is no translation, output "..."
			if changelog := cache.Changelog(); changelog == "" {
				c.text.SetValue("...")
			} else {
				c.text.SetValue(changelog)
			}
		} else {
			c.text.SetValue(cache.File().Repo.GetBody())
		}
		c.viewText.SetValue(cache.File().Repo.GetHTMLURL())
		context.SetEnabled(&c.viewButton, true)
	} else {
		c.text.SetValue("...")
		c.viewText.SetValue("?")
		context.SetEnabled(&c.viewButton, false)
	}

	c.viewButton.SetOnDown(func() {
		if value := c.viewText.Value(); value != "?" {
			go p86l.OpenBrowser(dm, value)
		}
	})
	c.viewButton.SetText(am.T("changelog.view"))

	c.form.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &c.viewText,
			SecondaryWidget: &c.viewButton,
		},
	})

	u := basicwidget.UnitSize(context)
	gl := layout.GridLayout{
		Bounds: context.Bounds(c).Inset(u / 2),
		Heights: []layout.Size{
			layout.FixedSize(c.text.DefaultSizeInContainer(context, context.Bounds(c).Dx()-u).Y),
			layout.FixedSize(c.form.DefaultSizeInContainer(context, context.Bounds(c).Dx()-u).Y),
		},
		RowGap: u / 2,
	}
	am.RenderBox(appender, &c.box1, gl.CellBounds(0, 0))
	am.RenderBox(appender, &c.box2, gl.CellBounds(0, 1))
	c.height = gl.CellBounds(0, 0).Dy() + gl.CellBounds(0, 1).Dy() + u*2
	appender.AppendChildWidgetWithBounds(&c.text, gl.CellBounds(0, 0))
	appender.AppendChildWidgetWithBounds(&c.form, gl.CellBounds(0, 1))

	return nil
}

func (c *changelogContent) Height() int {
	return c.height
}
