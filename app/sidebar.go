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
	"time"

	"github.com/dustin/go-humanize"
	"github.com/hajimehoshi/guigui"
	"github.com/hajimehoshi/guigui/basicwidget"
	"github.com/hajimehoshi/guigui/layout"
)

type Sidebar struct {
	guigui.DefaultWidget

	panel        basicwidget.Panel
	panelContent sidebarContent
}

func (s *Sidebar) SetModel(model *p86l.Model) {
	s.panelContent.SetModel(model)
}

func (s *Sidebar) Build(context *guigui.Context, appender *guigui.ChildWidgetAppender) error {
	context.SetOpacity(&s.panel, 0.9)
	s.panel.SetStyle(basicwidget.PanelStyleSide)
	s.panel.SetBorder(basicwidget.PanelBorder{
		End: true,
	})
	context.SetSize(&s.panelContent, context.ActualSize(s), s)
	s.panel.SetContent(&s.panelContent)

	appender.AppendChildWidgetWithBounds(&s.panel, context.Bounds(s))

	return nil
}

type sidebarContent struct {
	guigui.DefaultWidget

	list  basicwidget.List[string]
	stats sidebarStats

	model *p86l.Model
}

func (s *sidebarContent) SetModel(model *p86l.Model) {
	s.model = model
}

func (s *sidebarContent) Build(context *guigui.Context, appender *guigui.ChildWidgetAppender) error {
	am := s.model.App()

	s.stats.model = s.model
	s.list.SetStyle(basicwidget.ListStyleSidebar)

	items := []basicwidget.ListItem[string]{
		{
			Text: am.T("home.title"),
			ID:   "home",
		},
		{
			Text: am.T("play.title"),
			ID:   "play",
		},
		{
			Text: am.T("changelog.title"),
			ID:   "changelog",
		},
		{
			Text: am.T("settings.title"),
			ID:   "settings",
		},
		{
			Text: am.T("about.title"),
			ID:   "about",
		},
	}

	s.list.SetItems(items)
	s.list.SelectItemByID(s.model.Mode())
	s.list.SetItemHeight(basicwidget.UnitSize(context))
	s.list.SetOnItemSelected(func(index int) {
		item, ok := s.list.ItemByIndex(index)
		if !ok {
			s.model.SetMode("")
			return
		}
		if item.ID == s.model.Mode() {
			return
		}
		s.model.SetMode(item.ID)
	})

	gl := layout.GridLayout{
		Bounds: context.Bounds(s),
		Heights: []layout.Size{
			layout.FlexibleSize(2),
			layout.FlexibleSize(3),
		},
	}
	appender.AppendChildWidgetWithBounds(&s.list, gl.CellBounds(0, 0))
	appender.AppendChildWidgetWithBounds(&s.stats, gl.CellBounds(0, 1))

	return nil
}

type sidebarStats struct {
	guigui.DefaultWidget

	progressText   basicwidget.Text
	toastTextInput basicwidget.TextInput
	versionText    basicwidget.Text
	ratelimitText  basicwidget.Text

	model *p86l.Model
}

func (s *sidebarStats) Build(context *guigui.Context, appender *guigui.ChildWidgetAppender) error {
	am := s.model.App()
	dm := am.Debug()
	rlm := s.model.Ratelimit()

	s.progressText.SetValue(s.model.Progress())
	s.progressText.SetAutoWrap(true)

	if toast := dm.Toast(); toast != nil {
		s.toastTextInput.SetValue(toast.String())
	} else {
		s.toastTextInput.SetValue("")
	}
	s.toastTextInput.SetMultiline(true)
	s.toastTextInput.SetAutoWrap(true)
	s.toastTextInput.SetEditable(false)

	s.versionText.SetValue(am.PlainVersion())
	s.versionText.SetHorizontalAlign(basicwidget.HorizontalAlignCenter)
	s.versionText.SetVerticalAlign(basicwidget.VerticalAlignMiddle)

	if limit := rlm.Limit(); limit == nil {
		s.ratelimitText.SetValue("...")
	} else {
		rLeft := limit.Core.Remaining
		rTime := humanize.RelTime(time.Now(), limit.Core.Reset.Time, "remaining", "ago")
		s.ratelimitText.SetValue(fmt.Sprintf("%d / 60 requests - %s", rLeft, rTime))
	}
	s.ratelimitText.SetAutoWrap(true)
	s.ratelimitText.SetHorizontalAlign(basicwidget.HorizontalAlignCenter)

	u := basicwidget.UnitSize(context)
	gl := layout.GridLayout{
		Bounds: context.Bounds(s),
		Widths: []layout.Size{
			layout.FixedSize(u / 4),
			layout.FlexibleSize(1),
			layout.FixedSize(u / 2),
		},
		Heights: []layout.Size{
			layout.FlexibleSize(1),
			layout.FlexibleSize(1),
			layout.FixedSize(s.versionText.DefaultSize(context).Y),
			layout.FixedSize(u * 2),
		},
		RowGap: u / 2,
	}
	appender.AppendChildWidgetWithBounds(&s.progressText, gl.CellBounds(1, 0))
	appender.AppendChildWidgetWithBounds(&s.toastTextInput, gl.CellBounds(1, 1))
	appender.AppendChildWidgetWithBounds(&s.versionText, gl.CellBounds(1, 2))
	appender.AppendChildWidgetWithBounds(&s.ratelimitText, gl.CellBounds(1, 3))

	return nil
}
