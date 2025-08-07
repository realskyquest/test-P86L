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
	"flag"
	"fmt"
	"net"
	"os"
	"p86l/app"
	"p86l/configs"
	pd "p86l/internal/debug"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/guigui"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

var version = "dev"

func main() {
	port := flag.Int("instance", 54321, "Prot to use for single-instance locking")
	flag.Parse()

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	result, newRoot, model := app.NewRoot(version)
	am := model.App()
	dm := am.Debug()
	fs := am.FileSystem()

	if version == "dev" {
		newLog := zerolog.New(os.Stdout).With().Timestamp().Logger()
		newLog = newLog.Output(zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: "2006/01/02 15:04:05",
		})
		dm.SetLog(&newLog)

		for _, token := range strings.Split(os.Getenv("P86L_DEBUG"), ",") {
			switch token {
			case "log":
				am.SetLogsEnabled(true)
			case "box":
				am.SetBoxesEnabled(true)
			}
		}
	} else {
		logFile, rErr := fs.Root.Create(filepath.Join(fs.PathDirApp(), "log.txt"))
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

	if !result.Ok {
		result.Err.LogErrStack(am.Debug().Log(), "main", "NewRoot", pd.FileManager)
		os.Exit(1)
	}
	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *port))
	if err != nil {
		dm.Log().Error().Err(fmt.Errorf("Another instance is already running (or port %d is in use): %w", *port, err)).Msg(pd.NetworkManager)
		os.Exit(1)
	}
	defer func() {
		err := l.Close()
		if err != nil {
			dm.Log().Error().Err(fmt.Errorf("Failed to close instance: %w", err)).Msg(pd.ErrorManager)
		}
	}()

	op := &guigui.RunOptions{
		Title:         configs.AppTitle,
		WindowMinSize: configs.AppWindowMinSize,
	}
	if err := guigui.Run(newRoot, op); err != nil {
		if result := am.Error(); !result.Ok {
			result.Err.LogErrStack(am.Debug().Log(), "main", "guigui.Run", pd.ErrorManager)
		}
		os.Exit(1)
	}
}
