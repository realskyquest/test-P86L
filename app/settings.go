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
	"p86l/configs"
	pd "p86l/internal/debug"
	"path/filepath"

	"github.com/hajimehoshi/guigui"
	"github.com/hajimehoshi/guigui/basicwidget"
	"github.com/hajimehoshi/guigui/layout"
	"golang.org/x/text/language"
)

type Settings struct {
	guigui.DefaultWidget

	data  settingsData
	open  settingsOpen
	reset settingsReset

	model *p86l.Model
	err   *pd.Error
}

func (s *Settings) SetModel(model *p86l.Model) {
	s.model = model
}

func (s *Settings) Build(context *guigui.Context, appender *guigui.ChildWidgetAppender) error {
	am := s.model.App()

	if s.err != nil {
		am.SetError(s.err)
		return s.err.Error()
	}

	s.data.model = s.model
	s.open.model = s.model
	s.reset.model = s.model

	u := basicwidget.UnitSize(context)
	gl := layout.GridLayout{
		Bounds: context.Bounds(s).Inset(u / 2),
		Heights: []layout.Size{
			layout.FixedSize(u * 7),
			layout.FixedSize(u * 4),
			layout.FixedSize(u * 5),
		},
	}
	appender.AppendChildWidgetWithBounds(&s.data, gl.CellBounds(0, 0))
	appender.AppendChildWidgetWithBounds(&s.open, gl.CellBounds(0, 1))
	appender.AppendChildWidgetWithBounds(&s.reset, gl.CellBounds(0, 2))

	return nil
}

type settingsData struct {
	guigui.DefaultWidget

	form                  basicwidget.Form
	localeText            basicwidget.Text
	localeDropdownList    basicwidget.DropdownList[language.Tag]
	colorModeText         basicwidget.Text
	colorModeToggle       basicwidget.Toggle
	scaleText             basicwidget.Text
	scaleSegmentedControl basicwidget.SegmentedControl[int]

	model *p86l.Model

	err *pd.Error
}

func (s *settingsData) Build(context *guigui.Context, appender *guigui.ChildWidgetAppender) error {
	am := s.model.App()
	dm := am.Debug()
	data := s.model.Data()
	cache := s.model.Cache()

	s.localeText.SetValue(am.T("settings.locale"))
	s.colorModeText.SetValue(am.T("settings.colormode"))
	s.scaleText.SetValue(am.T("settings.appscale"))

	s.localeDropdownList.SetItems(localeItems)
	s.localeDropdownList.SetOnItemSelected(func(index int) {
		item, ok := s.localeDropdownList.ItemByIndex(index)
		if !ok {
			context.SetAppLocales(nil)
			return
		}
		if item.ID == language.English {
			data.SetLocale(am, context, language.English)
			context.SetAppLocales(nil)
			s.err = data.Save(am)
			return
		}
		data.SetLocale(am, context, item.ID)
		cache.SetChangelog(am, item.ID.String())
		s.err = data.Save(am)
	})
	if !s.localeDropdownList.IsPopupOpen() {
		if locales := context.AppendAppLocales(nil); len(locales) > 0 {
			s.localeDropdownList.SelectItemByID(locales[0])
		} else {
			s.localeDropdownList.SelectItemByID(language.English)
		}
	}

	s.colorModeToggle.SetOnValueChanged(func(value bool) {
		if value {
			data.SetColorMode(dm, context, guigui.ColorModeDark)
		} else {
			data.SetColorMode(dm, context, guigui.ColorModeLight)
		}
		s.err = data.Save(am)
	})
	switch context.ColorMode() {
	case guigui.ColorModeLight:
		s.colorModeToggle.SetValue(false)
	case guigui.ColorModeDark:
		s.colorModeToggle.SetValue(true)
	default:
		s.colorModeToggle.SetValue(false)
	}

	s.scaleSegmentedControl.SetItems([]basicwidget.SegmentedControlItem[int]{
		{
			Text: "50%",
			ID:   0,
		},
		{
			Text: "75%",
			ID:   1,
		},
		{
			Text: "100%",
			ID:   2,
		},
		{
			Text: "125%",
			ID:   3,
		},
		{
			Text: "150%",
			ID:   4,
		},
	})
	s.scaleSegmentedControl.SetOnItemSelected(func(index int) {
		item, ok := s.scaleSegmentedControl.ItemByIndex(index)
		if !ok {
			data.SetAppScale(dm, context, 2)
			return
		}
		data.SetAppScale(dm, context, item.ID)
		s.err = data.Save(am)
	})
	s.scaleSegmentedControl.SelectItemByID(data.File().AppScale)

	if s.err != nil {
		am.SetError(s.err)
		return s.err.Error()
	}

	s.form.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &s.localeText,
			SecondaryWidget: &s.localeDropdownList,
		},
		{
			PrimaryWidget:   &s.colorModeText,
			SecondaryWidget: &s.colorModeToggle,
		},
		{
			PrimaryWidget:   &s.scaleText,
			SecondaryWidget: &s.scaleSegmentedControl,
		},
	})

	appender.AppendChildWidgetWithBounds(&s.form, context.Bounds(s))
	return nil
}

type settingsOpen struct {
	guigui.DefaultWidget

	form                 basicwidget.Form
	companyFolderText    basicwidget.Text
	companyFolderButton  basicwidget.Button
	launcherFolderText   basicwidget.Text
	launcherFolderButton basicwidget.Button

	model *p86l.Model
}

func (s *settingsOpen) Build(context *guigui.Context, appender *guigui.ChildWidgetAppender) error {
	am := s.model.App()
	dm := am.Debug()
	fs := am.FileSystem()

	s.companyFolderText.SetValue(am.T("settings.company"))
	s.companyFolderButton.SetText(am.T("common.open"))
	s.launcherFolderText.SetValue(am.T("settings.launcher"))
	s.launcherFolderButton.SetText(am.T("common.open"))

	s.companyFolderButton.SetOnDown(func() {
		go func() {
			fs.OpenFileManager(dm, fs.CompanyDirPath)
		}()
	})
	s.launcherFolderButton.SetOnDown(func() {
		go func() {
			fs.OpenFileManager(dm, filepath.Join(fs.CompanyDirPath, configs.AppName))
		}()
	})

	s.form.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   &s.companyFolderText,
			SecondaryWidget: &s.companyFolderButton,
		},
		{
			PrimaryWidget:   &s.launcherFolderText,
			SecondaryWidget: &s.launcherFolderButton,
		},
	})

	appender.AppendChildWidgetWithBounds(&s.form, context.Bounds(s))
	return nil
}

type settingsReset struct {
	guigui.DefaultWidget

	form        basicwidget.Form
	dataButton  basicwidget.Button
	cacheButton basicwidget.Button

	model *p86l.Model
}

func (s *settingsReset) Build(context *guigui.Context, appender *guigui.ChildWidgetAppender) error {
	am := s.model.App()
	dm := am.Debug()
	data := s.model.Data()
	cache := s.model.Cache()

	s.dataButton.SetText(am.T("settings.resetdata"))
	s.cacheButton.SetText(am.T("settings.resetcache"))

	s.dataButton.SetOnDown(func() {
		d := p86l.NewData()
		data.SetFile(am, d)
		err := p86l.LoadB(am, context, s.model, "data")
		if err != nil {
			dm.SetPopup(err)
		}
	})
	s.cacheButton.SetOnDown(func() {
		cache.SetValid(false)
		cache.SetProgress(false)
	})

	s.form.SetItems([]basicwidget.FormItem{
		{
			PrimaryWidget:   nil,
			SecondaryWidget: &s.dataButton,
		},
		{
			PrimaryWidget:   nil,
			SecondaryWidget: &s.cacheButton,
		},
	})

	appender.AppendChildWidgetWithBounds(&s.form, context.Bounds(s))
	return nil
}
