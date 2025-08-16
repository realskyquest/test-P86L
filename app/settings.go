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
	"p86l/configs"
	pd "p86l/internal/debug"
	"path/filepath"
	"sync"

	"github.com/hajimehoshi/guigui"
	"github.com/hajimehoshi/guigui/basicwidget"
	"golang.org/x/text/language"
)

type Settings struct {
	guigui.DefaultWidget

	panel   basicwidget.Panel
	content settingsContent
}

func (s *Settings) AppendChildWidgets(context *guigui.Context, appender *guigui.ChildWidgetAppender) {
	appender.AppendChildWidget(&s.panel)
}

func (s *Settings) Build(context *guigui.Context) error {
	context.SetSize(&s.content, image.Pt(context.ActualSize(s).X, s.content.Height()), s)
	s.panel.SetContent(&s.content)
	context.SetBounds(&s.panel, context.Bounds(s), s)

	return nil
}

type settingsContent struct {
	guigui.DefaultWidget

	form basicwidget.Form

	localeText            basicwidget.Text
	localeDropdownList    basicwidget.DropdownList[language.Tag]
	colorModeText         basicwidget.Text
	colorModeToggle       basicwidget.Toggle
	scaleText             basicwidget.Text
	scaleSegmentedControl basicwidget.SegmentedControl[int]

	companyFolderText    basicwidget.Text
	companyFolderButton  basicwidget.Button
	launcherFolderText   basicwidget.Text
	launcherFolderButton basicwidget.Button

	dataButton  basicwidget.Button
	cacheButton basicwidget.Button

	box    basicwidget.Background
	height int

	sync   sync.Once
	result pd.Result
}

func (s *settingsContent) AppendChildWidgets(context *guigui.Context, appender *guigui.ChildWidgetAppender) {
	model := context.Model(s, modelKeyModel).(*p86l.Model)
	am := model.App()

	am.RenderBox(appender, &s.box)
	appender.AppendChildWidget(&s.form)
}

func (s *settingsContent) Build(context *guigui.Context) error {
	model := context.Model(s, modelKeyModel).(*p86l.Model)
	am := model.App()
	dm := am.Debug()
	fs := am.FileSystem()
	data := model.Data()
	cache := model.Cache()

	s.sync.Do(func() {
		s.result = pd.Ok()
	})

	if !s.result.Ok {
		am.SetError(s.result)
		return s.result.Err.Error()
	}

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
		if item.Value == language.English {
			data.SetLocale(am, context, language.English)
			context.SetAppLocales(nil)
			s.result = data.Save(am)
			return
		}
		data.SetLocale(am, context, item.Value)
		cache.SetChangelog(am, item.Value.String())
		result := data.Save(am)
		if !s.result.Ok {
			s.result = result
		}
	})
	if !s.localeDropdownList.IsPopupOpen() {
		if locales := context.AppendAppLocales(nil); len(locales) > 0 {
			s.localeDropdownList.SelectItemByValue(locales[0])
		} else {
			s.localeDropdownList.SelectItemByValue(language.English)
		}
	}

	s.colorModeToggle.SetOnValueChanged(func(value bool) {
		if value {
			data.SetColorMode(dm, context, guigui.ColorModeDark)
		} else {
			data.SetColorMode(dm, context, guigui.ColorModeLight)
		}
		result := data.Save(am)
		if !result.Ok {
			s.result = result
		}
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
			Text:  "50%",
			Value: 0,
		},
		{
			Text:  "75%",
			Value: 1,
		},
		{
			Text:  "100%",
			Value: 2,
		},
		{
			Text:  "125%",
			Value: 3,
		},
		{
			Text:  "150%",
			Value: 4,
		},
	})
	s.scaleSegmentedControl.SetOnItemSelected(func(index int) {
		item, ok := s.scaleSegmentedControl.ItemByIndex(index)
		if !ok {
			data.SetAppScale(dm, context, 2)
			return
		}
		data.SetAppScale(dm, context, item.Value)
		result := data.Save(am)
		if !result.Ok {
			s.result = result
		}
	})
	s.scaleSegmentedControl.SelectItemByValue(data.File().AppScale)

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

	s.dataButton.SetText(am.T("settings.resetdata"))
	s.cacheButton.SetText(am.T("settings.resetcache"))

	s.dataButton.SetOnDown(func() {
		d := p86l.NewData()
		data.SetFile(am, d)
		result := p86l.LoadB(am, context, model, "data")
		if !result.Ok {
			dm.SetPopup(result.Err, pd.FileManager)
		}
	})
	s.cacheButton.SetOnDown(func() {
		dm.Log().Info().Str("settingsContent", "cacheButton.SetOnDown").Msg(pd.AppManager)
		cache.SetValid(false)
		cache.Fetch(model)
	})

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
		{
			PrimaryWidget:   &s.companyFolderText,
			SecondaryWidget: &s.companyFolderButton,
		},
		{
			PrimaryWidget:   &s.launcherFolderText,
			SecondaryWidget: &s.launcherFolderButton,
		},
		{
			SecondaryWidget: &s.dataButton,
		},
		{
			SecondaryWidget: &s.cacheButton,
		},
	})
	u := basicwidget.UnitSize(context)
	context.SetBounds(&s.box, context.Bounds(s).Inset(u/2), s)
	s.height = s.form.DefaultSizeInContainer(context, context.Bounds(s).Inset(u/2).Dx()-u).Y + u
	context.SetBounds(&s.form, context.Bounds(s).Inset(u/2), s)

	return nil
}

func (c *settingsContent) Height() int {
	return c.height
}
