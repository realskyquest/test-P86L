//go:generate goversioninfo -icon=../../assets/p86l.ico -manifest=app.manifest

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

package main

import (
	"os"
	"p86l/app"
	"p86l/configs"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/guigui"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

var version = "dev"

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
}

func main() {
	newRoot, err := app.NewRoot(version)
	if err != nil {
		err.LogErrStack("main", "NewRoot")
		os.Exit(1)
	}

	am := newRoot.Model().App()
	dm := am.Debug()
	fs := am.FileSystem()

	if version == "dev" {
		for _, token := range strings.Split(os.Getenv("P86L_DEBUG"), ",") {
			switch token {
			case "log":
				newLog := zerolog.New(os.Stdout).With().Timestamp().Logger()
				newLog = newLog.Output(zerolog.ConsoleWriter{
					Out:        os.Stderr,
					TimeFormat: "2006/01/02 15:04:05",
				})

				dm.SetLog(&newLog)
				am.SetLogsEnabled(true)
			case "box":
				am.SetBoxesEnabled(true)
			}
		}
	} else {
		logFile, rErr := fs.Root.Create(filepath.Join(fs.DirAppPath(), "log.txt"))
		if rErr != nil {
			am.Debug().Log().Error().Err(rErr).Msg("Failed to create log file")
			os.Exit(1)
		}

		multi := zerolog.MultiLevelWriter(os.Stdout, logFile)
		newLog := zerolog.New(multi).With().Timestamp().Logger()
		dm.SetLog(&newLog)
		am.SetLogsEnabled(true)
	}

	if !am.LogsEnabled() {
		zerolog.SetGlobalLevel(zerolog.Disabled)
	}

	op := &guigui.RunOptions{
		Title:         configs.AppTitle,
		WindowMinSize: configs.AppWindowMinSize,
	}
	if rErr := guigui.Run(newRoot, op); rErr != nil {
		err := am.Error()
		if err != nil {
			err.LogErrStack("main", "guigui.Run")
		}
		os.Exit(1)
	}
}
