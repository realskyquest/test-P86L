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

package p86l

import (
	pd "p86l/internal/debug"
	"p86l/internal/file"
	"time"

	"github.com/hajimehoshi/guigui"
	"github.com/hashicorp/go-version"
	"golang.org/x/text/language"
)

type DataModel struct {
	validGameVersion bool
	file             file.Data
}

func NewData() file.Data {
	return file.Data{
		V:              0, // TODO: Set data version via env??
		WindowX:        0,
		WindowY:        0,
		WindowWidth:    0,
		WindowHeight:   0,
		WindowMaximize: false,
		Locale:         language.English.String(),
		AppScale:       2,
		ColorMode:      guigui.ColorModeLight,
		UsePreRelease:  false,
		PlayTime:       0,
		LastPlayed:     time.Now(),
		GameVersion:    "",
	}
}

func (d *DataModel) File() *file.Data {
	return &d.file
}

func (d *DataModel) IsValidGameVersion() bool {
	return d.validGameVersion
}

func (d *DataModel) GetAppScaleI(scale float64) int {
	switch scale {
	case 0.5: // 50%
		return 0
	case 0.75: // 75%
		return 1
	case 1.0: // 100%
		return 2
	case 1.25: // 125%
		return 3
	case 1.50: // 150%
		return 4
	}

	return -1
}

func (d *DataModel) GetAppScaleF(scale int) float64 {
	switch scale {
	case 0: // 50%
		return 0.5
	case 1: // 75%
		return 0.75
	case 2: // 100%
		return 1.0
	case 3: // 125%
		return 1.25
	case 4: // 150%
		return 1.50
	}

	return -1
}

// -- Setters for DataModel --

func (d *DataModel) SetPosition(x, y int) {
	d.file.WindowX = x
	d.file.WindowY = y
}

func (d *DataModel) SetSize(width, height int) {
	d.file.WindowWidth = width
	d.file.WindowHeight = height
}

func (d *DataModel) SetLocale(am *AppModel, context *guigui.Context, locale language.Tag) {
	am.Debug().Log().Info().Any("Translation", locale).Str("DataModel", "SetLocale").Msg(pd.FileManager)
	d.file.Locale = locale.String()
	context.SetAppLocales([]language.Tag{locale})
	am.SetLocale(locale.String())
}

func (d *DataModel) SetAppScale(dm *pd.Debug, context *guigui.Context, scale int) {
	dm.Log().Info().Any("Scaling", scale).Str("DataModel", "SetAppScale").Msg(pd.FileManager)
	d.file.AppScale = scale
	context.SetAppScale(d.GetAppScaleF(scale))
}

func (d *DataModel) SetColorMode(dm *pd.Debug, context *guigui.Context, mode guigui.ColorMode) {
	dm.Log().Info().Any("Theme", mode).Str("DataModel", "SetColorMode").Msg(pd.FileManager)
	d.file.ColorMode = mode
	context.SetColorMode(mode)
}

func (d *DataModel) SetUsePreRelease(dm *pd.Debug, value bool) {
	dm.Log().Info().Any("Pre-release", value).Str("DateModel", "SetUsePreRelease").Msg(pd.FileManager)
	d.file.UsePreRelease = value
}

func (d *DataModel) SetPlayTime(dm *pd.Debug, value int, timestamp time.Time) {
	dm.Log().Info().Int("PlayTime", value).Str("DataModel", "SetPlayTime").Msg(pd.FileManager)
	dm.Log().Info().Str("LastPlayed", timestamp.String()).Str("DataModel", "SetPlayTime").Msg(pd.FileManager)
	d.file.PlayTime = value
	d.file.LastPlayed = timestamp
}

func (d *DataModel) SetGameVersion(dm *pd.Debug, ver string) pd.Result {
	if ver == "" {
		return pd.Ok()
	}

	_, err := version.NewVersion(ver)
	if err != nil {
		return pd.NotOk(pd.New(err, pd.AppError, pd.ErrGameVersionInvalid))
	}

	dm.Log().Info().Any("Game Version", ver).Str("DateModel", "SetGameVersion").Msg(pd.FileManager)
	d.file.GameVersion = ver
	return pd.Ok()
}

func (d *DataModel) SetFile(am *AppModel, file file.Data) pd.Result {
	am.Debug().Log().Info().Str("DataModel", "SetFile").Msg(pd.FileManager)
	d.file = file
	result := d.Save(am)
	if !result.Ok {
		return result
	}
	return pd.Ok()
}

func (d *DataModel) Save(am *AppModel) pd.Result {
	am.Debug().Log().Info().Str("DataModel", "Save").Msg(pd.FileManager)
	result := SaveData(am, d.file)
	if !result.Ok {
		return result
	}
	return pd.Ok()
}
