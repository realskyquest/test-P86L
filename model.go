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

import "github.com/google/go-github/v71/github"

type Model struct {
	mode     string
	progress string

	ratelimit RatelimitModel
	app       AppModel
	data      DataModel
	cache     CacheModel
}

// -- Getters for Model --

func (m *Model) Mode() string {
	if m.mode == "" {
		return "home"
	}
	return m.mode
}

func (m *Model) Progress() string {
	return m.progress
}

// -- Setters for Model --

func (m *Model) SetMode(mode string) {
	d := m.app.Debug()
	d.Log().Info().Str("Page", mode).Msg("Sidebar")
	m.mode = mode
}

func (m *Model) SetProgress(progress string) {
	m.progress = progress
}

func (m *Model) GameExecutablePath() string {
	am := m.App()
	data := m.Data()
	if data.File().UsePreRelease {
		return am.FileSystem().PathFilePrerelease()
	}
	return am.FileSystem().PathFileGame()
}

// -- Models --

func (m *Model) App() *AppModel {
	return &m.app
}

func (m *Model) Ratelimit() *RatelimitModel {
	return &m.ratelimit
}

func (m *Model) Data() *DataModel {
	return &m.data
}

func (m *Model) Cache() *CacheModel {
	return &m.cache
}

// -- RatelimitModel --

type RatelimitModel struct {
	progress bool
	limit    *github.RateLimits
}

func (r *RatelimitModel) Progress() bool {
	return r.progress
}

func (r *RatelimitModel) Limit() *github.RateLimits {
	return r.limit
}

func (r *RatelimitModel) SetProgress(value bool) {
	r.progress = value
}

func (r *RatelimitModel) SetLimit(value *github.RateLimits) {
	r.limit = value
}
