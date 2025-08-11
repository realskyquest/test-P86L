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

package file

import (
	"errors"
	pd "p86l/internal/debug"
	"time"

	"github.com/google/go-github/v71/github"
	"github.com/hajimehoshi/guigui"
)

type Data struct {
	V              int              `json:"v"`
	WindowX        int              `json:"window_x"`
	WindowY        int              `json:"window_y"`
	WindowWidth    int              `json:"window_width"`
	WindowHeight   int              `json:"window_height"`
	WindowMaximize bool             `json:"window_maximize"`
	Locale         string           `json:"locale"`
	AppScale       int              `json:"app_scale"`
	ColorMode      guigui.ColorMode `json:"color_mode"`
	UsePreRelease  bool             `json:"use_pre_release"`
	PlayTime       int              `json:"play_time"`
	LastPlayed     time.Time        `json:"last_played"`
	GameVersion    string           `json:"game_version"`
}

func (d *Data) Log(dm *pd.Debug) {
	log := dm.Log()

	log.Info().Any("Translation", d.Locale).Msg("FileManager")
	log.Info().Any("Scaling", d.AppScale).Msg("FileManager")
	log.Info().Any("Theme", d.ColorMode).Msg("FileManager")
	if d.GameVersion == "" {
		return
	}
	log.Info().Any("Use Pre-release", d.UsePreRelease).Msg("FileManager")
	log.Info().Any("Play Time", d.PlayTime).Msg("FileManager")
	log.Info().Any("Game Version", d.GameVersion).Msg("FileManager")
}

type Cache struct {
	V         int                       `json:"v"`
	Repo      *github.RepositoryRelease `json:"repo"`
	PreRepo   *github.RepositoryRelease `json:"prerelease_repo"`
	Timestamp time.Time                 `json:"time_stamp"`
	ExpiresIn time.Duration             `json:"expires_in"`
}

// TODO: Is this really needed?
func (c *Cache) Log(dm *pd.Debug) {
	log := dm.Log()

	log.Info().Any("Changelog", c.Repo.GetBody()).Any("Timestamp", c.Timestamp).Any("ExpiresIn", c.ExpiresIn).Msg("FileManager")
}

func (c *Cache) Validate(ghRepo *github.RepositoryRelease) pd.Result {
	if ghRepo == nil {
		return pd.NotOk(pd.New(errors.New("repo is empty"), pd.CacheError, pd.ErrCacheInvalid))
	}
	if ghRepo.GetBody() == "" {
		return pd.NotOk(pd.New(errors.New("body is empty"), pd.CacheError, pd.ErrCacheBodyInvalid))
	}
	if ghRepo.GetHTMLURL() == "" {
		return pd.NotOk(pd.New(errors.New("htmlurl is empty"), pd.CacheError, pd.ErrCacheURLInvalid))
	}
	if len(ghRepo.Assets) < 1 {
		return pd.NotOk(pd.New(errors.New("assets are empty"), pd.CacheError, pd.ErrCacheAssetsInvalid))
	}

	return pd.Ok()
}
