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

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type ErrorType string

type Result struct {
	Err *Error
	Ok  bool
}

func NotOk(err *Error) Result {
	return Result{Err: err, Ok: false}
}

func Ok() Result {
	return Result{Ok: true}
}

// -- Custom Error --

type Error struct {
	err     error
	errType ErrorType
	code    int
}

func New(err error, errType ErrorType, code int) *Error {
	return &Error{
		err:     errors.WithStack(err),
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

func (e *Error) Equal(other *Error) bool {
	if e == nil || other == nil {
		return e == nil && other == nil
	}
	if e.err.Error() != other.err.Error() {
		return false
	}
	return e.errType == other.errType && e.code == other.code
}

func (e *Error) String() string {
	if e != nil {
		return fmt.Sprintf("Code: ( %d ), Type: ( %s ), Err: ( %s )", e.code, string(e.errType), e.err.Error())
	}
	return ""
}

func (e *Error) LogErr(log *zerolog.Logger, logStruct, logFunc, manager string) {
	log.Error().Int("Code", e.code).Any("Type", e.errType).Err(e.err).Str(logStruct, logFunc).Msg(manager)
}

func (e *Error) LogErrStack(log *zerolog.Logger, logStruct, logFunc, manager string) {
	log.Error().Stack().Int("Code", e.code).Any("Type", e.errType).Err(e.err).Str(logStruct, logFunc).Msg(manager)
}

func (e *Error) LogWarn(log *zerolog.Logger, logStruct, logFunc, manager string) {
	log.Warn().Int("Code", e.code).Any("Type", e.errType).Err(e.err).Str(logStruct, logFunc).Msg(manager)
}

func (e *Error) LogWarnStack(log *zerolog.Logger, logStruct, logFunc, manager string) {
	log.Warn().Stack().Int("Code", e.code).Any("Type", e.errType).Err(e.err).Str(logStruct, logFunc).Msg(manager)
}

// -- ErrorManager --

type Debug struct {
	log             *zerolog.Logger
	toast           *Error
	lastLoggedToast *Error
	popup           *Error
	lastLoggedPopup *Error
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

func (d *Debug) SetToast(err *Error, manager string) {
	if err == nil {
		return
	}
	if d.toast == nil || !d.toast.Equal(err) {
		if d.lastLoggedToast == nil || !d.lastLoggedToast.Equal(err) {
			err.LogWarnStack(d.log, "Debug", "SetToast", manager)
			d.lastLoggedToast = err
		}
	}
	d.toast = err
}

func (d *Debug) SetPopup(err *Error, manager string) {
	if err == nil {
		return
	}
	if d.popup == nil || !d.popup.Equal(err) {
		if d.lastLoggedPopup == nil || !d.lastLoggedPopup.Equal(err) {
			err.LogWarnStack(d.log, "Debug", "SetPopup", manager)
			d.lastLoggedPopup = err
		}
	}
	d.popup = err
}
