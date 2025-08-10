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
	"p86l"
	"p86l/app"
	"p86l/configs"
	pd "p86l/internal/debug"
	"strings"

	"github.com/hajimehoshi/guigui"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

var version = "dev"

func main() {
	port := flag.Int("instance", 54321, "Port to use for single-instance locking")
	flag.Parse()

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	result, newRoot, model := app.NewRoot(version)
	if !result.Ok {
		fmt.Println("P86L - app.NewRoot - %w", result.Err)
		os.Exit(1)
	}
	am := model.App()
	dm := am.Debug()
	fs := am.FileSystem()
	defer func() {
		if err := fs.Root.Close(); err != nil {
			fmt.Println("Failed to close root: %w", err)
		}
	}()

	if version == "dev" {
		output := zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: "2006/01/02 15:04:05",
		}
		newLog := zerolog.New(output).With().Timestamp().Logger()
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
		if err := p86l.RotateLogFiles(fs.Root, fs.CompanyDirPath, fs.PathDirLogs()); err != nil {
			fmt.Println("Log rotation warning: %w", err)
			os.Exit(1)
		}

		logFile, err := p86l.NewLogFile(fs.Root, fs.PathDirLogs())
		if err != nil {
			fmt.Println("Failed to create new log file: %w", err)
			os.Exit(1)
		}
		defer func() {
			if err := logFile.Close(); err != nil {
				fmt.Println("Failed to close log file: %w", err)
			}
		}()

		multi := zerolog.MultiLevelWriter(os.Stdout, logFile)
		newLog := zerolog.New(multi).With().Timestamp().Logger()
		dm.SetLog(&newLog)
		am.SetLogsEnabled(true)
	}

	// After this use zerolog, its now available.
	if !am.LogsEnabled() {
		zerolog.SetGlobalLevel(zerolog.Disabled)
	}
	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *port))
	if err != nil {
		dm.Log().Error().Err(fmt.Errorf("Another instance is already running (or port %d is in use): %w", *port, err)).Msg(pd.NetworkManager)
		os.Exit(1)
	}
	defer func() {
		if err := l.Close(); err != nil {
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
