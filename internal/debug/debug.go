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

package debug

import (
	"fmt"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	ErrorManager   = "ErrorManager"
	FileManager    = "FileManager"
	NetworkManager = "NetworkManager"
)

type ErrorType string

const (
	UnknownError ErrorType = "unknown"
	AppError     ErrorType = "app"
	FSError      ErrorType = "filesystem"
	DataError    ErrorType = "data"
	CacheError   ErrorType = "cache"
	NetworkError ErrorType = "network"
)

const (
	// App errors (1001-1999)
	ErrUnknown int = iota + 1001
	ErrBrowserOpen

	ErrLauncherVersionInvalid

	ErrGameVersionInvalid
	ErrGameNotExist
	ErrGameRunning
)

const (
	// Filesystem errors (2001-2999)
	ErrFSOpenFileManagerInvalid int = iota + 2001

	ErrFSDirInvalid
	ErrFSDirNew
	ErrFSDirNotExist
	ErrFSDirRead
	ErrFSDirRename
	ErrFSDirRemove

	ErrFSFileInvalid
	ErrFSNewFileInvalid
	ErrFSFileWrite
	ErrFSFileNotExist

	// -- root --

	ErrFSRootInvalid

	ErrFSRootDirInvalid
	ErrFSRootDirNew

	ErrFSRootFileInvalid
	ErrFSRootFileNew
	ErrFSRootFileNotExist
	ErrFSRootFileRead
	ErrFSRootFileWrite
	ErrFSRootFileRemove
	ErrFSRootFileClose
)

const (
	// Data errors (3001-3999)
	ErrDataLoad int = iota + 3001
	ErrDataSave
	ErrDataReset
	ErrDataLocaleInvalid
)

const (
	// Cache errors (4001-4999)
	ErrCacheLoad int = iota + 4001
	ErrCacheSave
	ErrCacheReset
	ErrCacheInvalid
	ErrCacheRepoInvalid
	ErrCacheBodyInvalid
	ErrCacheURLInvalid
	ErrCacheAssetsInvalid
)

const (
	// // Network errors (5001-5999)
	ErrNetworkRateLimitInvalid int = iota + 5001
	ErrNetworkCacheRequest
	ErrNetworkDownloadRequest
	ErrNetworkStatusNotOk
)

type Error struct {
	err     error
	errType ErrorType
	code    int
}

func New(err error, errType ErrorType, code int) *Error {
	return &Error{
		err:     err,
		errType: errType,
		code:    code,
	}
}

func (e *Error) Error() error {
	return e.err
}

func (e *Error) ErrorType() ErrorType {
	return e.errType
}

func (e *Error) Code() int {
	return e.code
}

func (e *Error) IsType(errType ErrorType) bool {
	return e.errType == errType
}

func (e *Error) IsCode(code int) bool {
	return e.code == code
}

func (e *Error) String() string {
	if e != nil {
		return fmt.Sprintf("Code: ( %d ), Type: ( %s ), Err: ( %s )", e.code, string(e.errType), e.err.Error())
	}
	return ""
}

func (e *Error) LogErr(logStruct, logFunc string) {
	log.Error().Int("Code", e.code).Any("Type", e.errType).Err((e.err)).Str(logStruct, logFunc).Msg(ErrorManager)
}

func (e *Error) LogErrStack(logStruct, logFunc string) {
	log.Error().Stack().Int("Code", e.code).Any("Type", e.errType).Err((e.err)).Str(logStruct, logFunc).Msg(ErrorManager)
}

func (e *Error) LogWarn(logStruct, logFunc string) {
	log.Warn().Int("Code", e.code).Any("Type", e.errType).Err((e.err)).Str(logStruct, logFunc).Msg(ErrorManager)
}

func (e *Error) LogWarnStack(logStruct, logFunc string) {
	log.Warn().Stack().Int("Code", e.code).Any("Type", e.errType).Err((e.err)).Str(logStruct, logFunc).Msg(ErrorManager)
}

type Debug struct {
	log   *zerolog.Logger
	toast *Error
	popup *Error
}

func (d *Debug) Log() *zerolog.Logger {
	return d.log
}

func (d *Debug) Toast() *Error {
	return d.toast
}

func (d *Debug) Popup() *Error {
	return d.popup
}

func (d *Debug) SetLog(logger *zerolog.Logger) {
	d.log = logger
}

func (d *Debug) SetToast(err *Error) {
	err.LogWarnStack("Debug", "SetToast")
	d.toast = err
}

func (d *Debug) SetPopup(err *Error) {
	err.LogWarnStack("Debug", "SetPopup")
	d.popup = err
}
