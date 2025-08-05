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
	"cmp"
	"image"
	"p86l"
	"p86l/assets"
	p86lLocale "p86l/assets/locale"
	pd "p86l/internal/debug"
	"runtime"
	"slices"
	"sync"
	"time"

	"golang.org/x/text/language"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/guigui"
	"github.com/hajimehoshi/guigui/basicwidget"
	"github.com/hajimehoshi/guigui/layout"
	i18n "github.com/nicksnyder/go-i18n/v2/i18n"
)

type Root struct {
	guigui.DefaultWidget

	background rootBackground
	sidebar    Sidebar
	home       Home
	play       Play
	changelog  Changelog
	settings   Settings
	about      About

	popup         basicwidget.Popup
	popupContent  rootPopupContent
	popupDebounce bool

	lastTick int64
	model    p86l.Model

	locales           []language.Tag
	faceSourceEntries []basicwidget.FaceSourceEntry

	rlResult chan ratelimitResult
	ghResult chan githubResult

	sync   sync.Once
	result pd.Result
}

func NewRoot(version string) (pd.Result, *Root) {
	r := &Root{}
	am := r.model.App()

	am.SetPlainVersion(version)
	result := am.SetFileSystem()
	if !result.Ok {
		return result, nil
	}

	return pd.Ok(), r
}

func (r *Root) Model() *p86l.Model {
	return &r.model
}

func (r *Root) runApp() pd.Result {
	am := r.model.App()

	r.rlResult = make(chan ratelimitResult, 1)
	r.ghResult = make(chan githubResult, 1)

	iconImages, err1 := assets.GetIconImages()
	bundle, err2 := p86lLocale.GetLocales(language.English)

	if err := cmp.Or(err1, err2); err != nil {
		return pd.NotOk(err)
	}

	ebiten.SetWindowIcon(iconImages)
	am.SetI18nBundle(bundle)
	am.SetI18nLocalizer(i18n.NewLocalizer(bundle, "en"))

	return pd.Ok()
}

func (r *Root) updateFontFaceSources(context *guigui.Context) {
	r.locales = slices.Delete(r.locales, 0, len(r.locales))
	r.locales = context.AppendLocales(r.locales)

	r.faceSourceEntries = slices.Delete(r.faceSourceEntries, 0, len(r.faceSourceEntries))
	r.faceSourceEntries = AppendRecommendedFaceSourceEntries(r.faceSourceEntries, r.locales)
	basicwidget.SetFaceSources(r.faceSourceEntries)
}

func (r *Root) Build(context *guigui.Context, appender *guigui.ChildWidgetAppender) error {
	am := r.model.App()
	dm := am.Debug()
	log := dm.Log()
	data := r.model.Data()

	r.sync.Do(func() {
		var gpuInfo ebiten.DebugInfo
		ebiten.ReadDebugInfo(&gpuInfo)

		result1 := r.runApp()
		result2 := p86l.LoadB(am, context, &r.model, "data")
		result3 := p86l.LoadB(am, context, &r.model, "cache")

		if result := cmp.Or(result1, result2, result3); !result.Ok {
			r.result = result
			return
		}

		if launcherVersion := am.PlainVersion(); launcherVersion != "dev" {
			result4 := am.SetVersion(launcherVersion)
			if !result4.Ok {
				result4.Err.LogWarn(log, "Root.sync", "Build", pd.FileManager)
			}
		}

		log.Info().Msg("..:: GuiGui GUI Framework Alpha ::..")
		log.Info().Str("Version", am.PlainVersion()).Msg("P86L - Project 86 Launcher")
		log.Info().Str("Detected OS", runtime.GOOS).Msg("Operating System")
		log.Info().Str("Graphics API", gpuInfo.GraphicsLibrary.String()).Msg("GPU")
		log.Warn().Str("LICENSE", am.License()).Msg("README")

		r.model.Cache().SetChangelog(am, data.File().Locale)

		if data.File().WindowX > 0 || data.File().WindowY > 0 {
			ebiten.SetWindowPosition(data.File().WindowX, data.File().WindowY)
		}
		if data.File().WindowWidth > 0 || data.File().WindowHeight > 0 {
			ebiten.SetWindowSize(data.File().WindowWidth, data.File().WindowHeight)
		}
		if data.File().WindowMaximize {
			ebiten.MaximizeWindow()
		}

		r.result = pd.Ok()
	})

	if !r.result.Ok {
		am.SetError(r.result)
		return r.result.Err.Error()
	}
	r.updateFontFaceSources(context)

	if ebiten.IsWindowBeingClosed() {
		log.Info().Msg("P86L Closing")
		r.result = data.Save(am)
	}

	r.popupContent.model = &r.model
	r.background.model = &r.model

	r.sidebar.SetModel(&r.model)
	r.home.SetModel(&r.model)
	r.play.SetModel(&r.model)
	r.changelog.SetModel(&r.model)
	r.settings.SetModel(&r.model)
	r.about.SetModel(&r.model)

	u := basicwidget.UnitSize(context)
	gl := layout.GridLayout{
		Bounds: context.Bounds(r),
		Widths: []layout.Size{
			layout.FixedSize(8 * u),
			layout.FlexibleSize(1),
		},
	}
	r.background.SetSidebar(&r.sidebar)
	r.background.SetBgBounds(gl.CellBounds(1, 0))
	appender.AppendChildWidgetWithBounds(&r.background, context.Bounds(r))

	appender.AppendChildWidgetWithBounds(&r.sidebar, gl.CellBounds(0, 0))

	switch r.model.Mode() {
	case "home":
		appender.AppendChildWidgetWithBounds(&r.home, gl.CellBounds(1, 0))
	case "play":
		appender.AppendChildWidgetWithBounds(&r.play, gl.CellBounds(1, 0))
	case "changelog":
		appender.AppendChildWidgetWithBounds(&r.changelog, gl.CellBounds(1, 0))
	case "settings":
		appender.AppendChildWidgetWithBounds(&r.settings, gl.CellBounds(1, 0))
	case "about":
		appender.AppendChildWidgetWithBounds(&r.about, gl.CellBounds(1, 0))
	}

	// -- popup --

	r.popup.SetOnClosed(func(reason basicwidget.PopupClosedReason) {
		dm.SetPopup(nil, pd.UnknownManager)
		r.popupDebounce = false
	})
	if dm.Popup() != nil && !r.popupDebounce {
		r.popupDebounce = true
		r.popup.Open(context)
	}

	r.popupContent.popup = &r.popup
	r.popup.SetContent(&r.popupContent)
	r.popup.SetBackgroundBlurred(true)
	r.popup.SetCloseByClickingOutside(true)
	r.popup.SetAnimationDuringFade(false)

	appBounds := context.AppBounds()
	contentSize := image.Pt(int(12*u), int(6*u))
	popupPosition := image.Point{
		X: appBounds.Min.X + (appBounds.Dx()-contentSize.X)/2,
		Y: appBounds.Min.Y + (appBounds.Dy()-contentSize.Y)/2,
	}
	popupBounds := image.Rectangle{
		Min: popupPosition,
		Max: popupPosition.Add(contentSize),
	}
	context.SetSize(&r.popupContent, popupBounds.Size(), r)
	appender.AppendChildWidgetWithBounds(&r.popup, popupBounds)

	return nil
}

func (r *Root) Tick(context *guigui.Context) error {
	am := r.model.App()
	dm := am.Debug()
	log := dm.Log()
	rlm := r.model.Ratelimit()
	data := r.model.Data()
	cache := r.model.Cache()

	x, y := ebiten.WindowPosition()
	width, height := ebiten.WindowSize()
	maximized := ebiten.IsWindowMaximized()
	if !maximized {
		data.SetPosition(x, y)
		data.SetSize(width, height)
	}
	data.File().WindowMaximize = maximized

	for range 2 {
		select {
		case rlResult := <-r.rlResult:
			rlm.SetProgress(false)
			if !rlResult.result.Ok {
				dm.SetToast(rlResult.result.Err, pd.NetworkManager)
				rlm.SetLimit(nil)
			} else {
				rlm.SetLimit(rlResult.limit)
			}
		case ghResult := <-r.ghResult:
			cache.SetProgress(false)
			if !ghResult.result.Ok {
				dm.SetToast(ghResult.result.Err, pd.NetworkManager)
			} else {
				cache.SetRepos(am, ghResult.release, ghResult.prerelease, data.File().Locale)
			}
		default:

		}
	}

	currentTick := ebiten.Tick()
	if currentTick-r.lastTick >= int64(ebiten.TPS()*5) {
		r.lastTick = currentTick

		if cache.IsValid() && ((!rlm.Progress() && rlm.Limit() == nil) || (rlm.Limit() != nil && time.Now().After(rlm.Limit().Core.Reset.Time))) {
			rlm.SetProgress(true)
			log.Info().Str("Ratelimit", "ratelimit refresh triggered").Msg(pd.NetworkManager)
			go r.fetchRatelimit()
		}

		if !cache.Progress() && !cache.IsValid() || time.Now().After(cache.File().Timestamp.Add(cache.File().ExpiresIn)) {
			cache.SetProgress(true)
			log.Info().Str("Cache", "cache refresh triggered").Msg(pd.NetworkManager)
			go r.fetchLatestCache()
		}
	}

	return nil
}
