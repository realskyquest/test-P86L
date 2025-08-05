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

	"github.com/google/go-github/v71/github"
)

type CacheModel struct {
	progress  bool
	valid     bool
	file      file.Cache
	changelog string
}

func (c *CacheModel) Progress() bool {
	return c.progress
}

func (c *CacheModel) File() *file.Cache {
	return &c.file
}

func (c *CacheModel) IsValid() bool {
	return c.valid
}

func (c *CacheModel) Changelog() string {
	return c.changelog
}

// -- Setters for CacheModel --

func (c *CacheModel) SetProgress(value bool) {
	c.progress = value
}

func (c *CacheModel) SetValid(value bool) {
	c.valid = value
}

func (c *CacheModel) SetRepos(am *AppModel, repo, preRepo *github.RepositoryRelease, locale string) pd.Result {
	dm := am.Debug()

	c.file.Repo = repo
	c.file.PreRepo = preRepo
	c.file.Timestamp = time.Now()
	c.file.ExpiresIn = time.Hour
	if result := c.file.Validate(repo); !result.Ok {
		c.valid = false
	} else {
		c.valid = true
	}
	dm.Log().Info().Str("CacheModel", "SetRepos").Msg(pd.FileManager)
	return SaveCache(am, c.file)
}

func (c *CacheModel) SetChangelog(am *AppModel, locale string) {
	if !c.valid {
		return
	}

	if body := c.file.Repo.GetBody(); body != "" && locale != "en" {
		go func() {
			c.changelog = am.Translate(body, locale)
		}()
	}
}
