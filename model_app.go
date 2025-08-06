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

	translator "github.com/Conight/go-googletrans"
	"github.com/google/go-github/v71/github"
	"github.com/hajimehoshi/guigui"
	"github.com/hajimehoshi/guigui/basicwidget"
	"github.com/hashicorp/go-version"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

type AppModel struct {
	license      string
	plainVersion string
	version      *version.Version
	logs         bool
	box          bool

	githubClient *github.Client

	d  *pd.Debug
	fs *file.AppFS

	i18nBundle    *i18n.Bundle
	i18nLocalizer *i18n.Localizer
	translator    *translator.Translator

	result pd.Result
}

// License returns the license text for the application.
func (a *AppModel) License() string {
	if a.license == "" {
		a.license = `Project-86-Launcher: A Launcher developed for Project-86 for managing game files.
Copyright (C) 2025 Project 86 Community

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.`
	}
	return a.license
}

// PlainVersion returns the plain version string of the application.
func (a *AppModel) PlainVersion() string {
	return a.plainVersion
}

// Version returns the version of the application.
func (a *AppModel) Version() *version.Version {
	return a.version
}

// -- start of debug --

// LogsEnabled returns whether logging is enabled in the application.
func (a *AppModel) LogsEnabled() bool {
	return a.logs
}

// BoxesEnabled returns whether boxes is enabled in the application,
// renders a background on widgets to test widget bounds.
func (a *AppModel) BoxesEnabled() bool {
	return a.box
}

func (a *AppModel) RenderBox(appender *guigui.ChildWidgetAppender, widget *basicwidget.Background) {
	if a.BoxesEnabled() {
		appender.AppendChildWidget(widget)
	}
}

// -- end of debug --

// GithubClient returns the GitHub client for API interactions.
func (a *AppModel) GithubClient() *github.Client {
	if a.githubClient == nil {
		a.githubClient = github.NewClient(nil)
	}
	return a.githubClient
}

// Debug returns the debug model for logging and error handling.
func (a *AppModel) Debug() *pd.Debug {
	if a.d == nil {
		a.d = &pd.Debug{}
	}
	return a.d
}

// FileSystem returns the application filesystem model.
func (a *AppModel) FileSystem() *file.AppFS {
	return a.fs
}

// I18nBundle returns the i18n bundle for internationalization.
func (a *AppModel) I18nBundle() *i18n.Bundle {
	return a.i18nBundle
}

// I18nLocalizer returns the i18n localizer for localization.
func (a *AppModel) I18nLocalizer() *i18n.Localizer {
	return a.i18nLocalizer
}

// T gets the localized string for a given key.
func (a *AppModel) T(key string) string {
	keyMsg, err := a.i18nLocalizer.Localize(&i18n.LocalizeConfig{
		MessageID: key,
	})
	if err != nil {
		a.d.Log().Error().Err(err).Str("Key", key).Msg("Localization error")
		return key // Fallback to key if translation fails.
	}
	return keyMsg
}

// Translator returns the translator for language translation.
func (a *AppModel) Translator() *translator.Translator {
	if a.translator == nil {
		a.translator = translator.New()
	}
	return a.translator
}

// Translate translates a text to the target language using the translator;
// if the translation fails, it returns "?".
func (a *AppModel) Translate(text, targetLang string) string {
	result, err := a.translator.Translate(text, "auto", targetLang)
	if err != nil {
		return "?"
	}
	return result.Text
}

// Error returns the last error encountered in the application.
func (a *AppModel) Error() pd.Result {
	return a.result
}

// -- Setters for AppModel --

func (a *AppModel) SetFileSystem() pd.Result {
	result, fs := file.NewFS()
	if !result.Ok {
		return result
	}
	a.fs = fs
	return pd.Ok()
}

// SetPlainVersion sets the plain version of the application.
func (a *AppModel) SetPlainVersion(value string) {
	a.plainVersion = value
}

// SetVersion sets the version of the application.
func (a *AppModel) SetVersion(value string) pd.Result {
	v, err := version.NewVersion(value)
	if err != nil {
		return pd.NotOk(pd.New(err, pd.AppError, pd.ErrLauncherVersionInvalid))
	}
	a.version = v
	return pd.Ok()
}

// SetLogsEnabled sets whether logging is enabled in the application.
func (a *AppModel) SetLogsEnabled(enabled bool) {
	a.logs = enabled
}

// SetBoxesEnabled sets whether boxes is enabled in the application.
func (a *AppModel) SetBoxesEnabled(enabled bool) {
	a.box = enabled
}

// Set i18nBundle.
func (a *AppModel) SetI18nBundle(bundle *i18n.Bundle) {
	a.i18nBundle = bundle
}

// Set i18nLocalizer.
func (a *AppModel) SetI18nLocalizer(localizer *i18n.Localizer) {
	a.i18nLocalizer = localizer
}

// SetLocale sets the locale for the application.
func (a *AppModel) SetLocale(locale string) {
	a.i18nLocalizer = i18n.NewLocalizer(a.i18nBundle, locale)
}

// Set error for app.
func (a *AppModel) SetError(result pd.Result) {
	a.result = result
}
