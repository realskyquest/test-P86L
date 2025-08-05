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
	"context"
	"image"
	"p86l"
	"p86l/assets"
	"p86l/configs"
	pd "p86l/internal/debug"
	"sync"
	"time"

	"github.com/google/go-github/v71/github"
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

	model    *p86l.Model
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

func (r *rootBackground) Build(context *guigui.Context, appender *guigui.ChildWidgetAppender) error {
	am := r.model.App()

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
	appender.AppendChildWidgetWithPosition(&r.bgImage, imgPosition)

	if r.model.Mode() != "home" {
		appender.AppendChildWidgetWithBounds(&r.background, r.bgBounds)
	}

	return nil
}

type rootPopupContent struct {
	guigui.DefaultWidget

	popup *basicwidget.Popup

	titleText   basicwidget.Text
	closeButton basicwidget.Button

	model *p86l.Model
}

func (r *rootPopupContent) Build(context *guigui.Context, appender *guigui.ChildWidgetAppender) error {
	dm := r.model.App().Debug()
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
	appender.AppendChildWidgetWithBounds(&r.titleText, gl.CellBounds(0, 0))
	{
		gl := layout.GridLayout{
			Bounds: gl.CellBounds(0, 1),
			Widths: []layout.Size{
				layout.FlexibleSize(1),
				layout.FixedSize(r.closeButton.DefaultSize(context).X),
			},
		}
		appender.AppendChildWidgetWithBounds(&r.closeButton, gl.CellBounds(1, 0))
	}

	return nil
}

// -- Channels --

type ratelimitResult struct {
	result pd.Result
	limit  *github.RateLimits
}

type githubResult struct {
	result     pd.Result
	release    *github.RepositoryRelease
	prerelease *github.RepositoryRelease
}

func (r *Root) fetchRatelimit() {
	am := r.model.App()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	limit, _, err := am.GithubClient().RateLimit.Get(ctx)

	if err != nil {
		r.rlResult <- ratelimitResult{
			result: pd.NotOk(pd.New(err, pd.NetworkError, pd.ErrNetworkRateLimitInvalid)),
			limit:  nil,
		}
		return
	}
	r.rlResult <- ratelimitResult{
		result: pd.Ok(),
		limit:  limit,
	}
}

func (r *Root) fetchLatestCache() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result, release, prerelease := r.getCacheItems(ctx)

	if !result.Ok {
		r.ghResult <- githubResult{
			result:     result,
			release:    nil,
			prerelease: nil,
		}
		return
	}
	r.ghResult <- githubResult{
		result:     pd.Ok(),
		release:    release,
		prerelease: prerelease,
	}
}

func (r *Root) getCacheItems(ctx context.Context) (pd.Result, *github.RepositoryRelease, *github.RepositoryRelease) {
	am := r.model.App()

	release, _, err := am.GithubClient().Repositories.GetLatestRelease(ctx, configs.RepoOwner, configs.RepoName)
	if err != nil {
		return pd.NotOk(pd.New(err, pd.NetworkError, pd.ErrNetworkLatestInvalid)), nil, nil
	}

	result, prerelease := p86l.GetPreRelease(am)
	if !result.Ok {
		return result, nil, nil
	}

	return pd.Ok(), release, prerelease
}
