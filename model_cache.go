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
	"context"
	"p86l/configs"
	pd "p86l/internal/debug"
	"p86l/internal/file"
	"time"

	"github.com/google/go-github/v71/github"
)

type TypeRateLimitResult struct {
	result pd.Result
	limit  *github.RateLimits
}

type TypeGithubResult struct {
	result     pd.Result
	release    *github.RepositoryRelease
	prerelease *github.RepositoryRelease
}

type CacheModel struct {
	rateLimitResult chan TypeRateLimitResult
	githubResult    chan TypeGithubResult

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

func (c *CacheModel) IsTimestampValid() bool {
	return time.Now().Before(c.File().Timestamp.Add(c.File().ExpiresIn))
}

// -- Setters for CacheModel --

func (c *CacheModel) SetProgress(value bool) {
	c.progress = value
}

func (c *CacheModel) SetValid(value bool) {
	c.valid = value
}

func (c *CacheModel) SetRateLimits(am *AppModel, rateLimit *github.RateLimits) pd.Result {
	dm := am.Debug()

	c.file.RateLimit = rateLimit
	dm.Log().Info().Str("CacheModel", "SetRateLimits").Msg(pd.FileManager)
	return SaveCache(am, c.file)
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

// -- Common utils --

func (c *CacheModel) fetchRatelimit(model *Model) {
	am := model.App()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	limit, _, err := am.GithubClient().RateLimit.Get(ctx)

	if err != nil {
		c.rateLimitResult <- TypeRateLimitResult{
			result: pd.NotOk(pd.New(err, pd.NetworkError, pd.ErrNetworkRateLimitInvalid)),
			limit:  nil,
		}
		return
	}
	model.App().Debug().SetToast(nil, pd.NetworkManager)
	c.rateLimitResult <- TypeRateLimitResult{
		result: pd.Ok(),
		limit:  limit,
	}
}

func (c *CacheModel) fetchLatestCache(model *Model) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result, release, prerelease := c.getCacheItems(model, ctx)

	if !result.Ok {
		c.githubResult <- TypeGithubResult{
			result:     result,
			release:    nil,
			prerelease: nil,
		}
		return
	}
	c.githubResult <- TypeGithubResult{
		result:     pd.Ok(),
		release:    release,
		prerelease: prerelease,
	}
}

func (c *CacheModel) getCacheItems(model *Model, ctx context.Context) (pd.Result, *github.RepositoryRelease, *github.RepositoryRelease) {
	am := model.App()

	release, _, err := am.GithubClient().Repositories.GetLatestRelease(ctx, configs.RepoOwner, configs.RepoName)
	if err != nil {
		return pd.NotOk(pd.New(err, pd.NetworkError, pd.ErrNetworkLatestInvalid)), nil, nil
	}

	result, prerelease := GetPreRelease(am)
	if !result.Ok {
		return result, nil, nil
	}

	return pd.Ok(), release, prerelease
}

// -- Common --

func (c *CacheModel) Load() {
	c.rateLimitResult = make(chan TypeRateLimitResult, 1)
	c.githubResult = make(chan TypeGithubResult, 1)
}

func (c *CacheModel) Update(model *Model) {
	am := model.App()
	dm := am.Debug()
	rlm := model.RateLimits()
	data := model.Data()

	for range 2 {
		select {
		case rateLimitResult := <-c.rateLimitResult:
			rlm.SetProgress(false)

			if !rateLimitResult.result.Ok {
				dm.SetToast(rateLimitResult.result.Err, pd.NetworkManager)
			} else {
				c.SetRateLimits(am, rateLimitResult.limit)
			}
		case githubResult := <-c.githubResult:
			c.SetProgress(false)

			if !githubResult.result.Ok {
				dm.SetToast(githubResult.result.Err, pd.NetworkManager)
			} else {
				c.SetRepos(am, githubResult.release, githubResult.prerelease, data.File().Locale)
			}
		default:

		}
	}

	if !c.valid && (!rlm.Progress() && !c.Progress()) {
		dm.Log().Info().Str("CacheModel", "Update.missingCache").Msg(pd.AppManager)
		c.Fetch(model)
	}

	if c.valid && !c.IsTimestampValid() && c.file.RateLimit.Core.Remaining > 0 && (!rlm.Progress() && !c.Progress()) {
		dm.Log().Info().Str("CacheModel", "Update.cacheExpired").Msg(pd.AppManager)
		c.Fetch(model)
	}
}

func (c *CacheModel) Fetch(model *Model) {
	log := model.App().Debug().Log()
	rlm := model.RateLimits()
	cache := model.Cache()

	if rlm.Progress() || cache.Progress() {
		return
	}

	log.Info().Str("CacheModel", "Fetch").Msg(pd.NetworkManager)
	rlm.SetProgress(true)
	cache.SetProgress(true)
	go func() {
		c.fetchLatestCache(model)
		c.fetchRatelimit(model)
	}()
}
