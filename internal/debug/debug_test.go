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
	"testing"
)

func TestResultOk(t *testing.T) {
	result := Ok()
	if result.Ok {
		t.Logf("%s", result.Err.String())
		return
	}
	t.Fatalf("Result should should ok")
}

func TestResultNotOk(t *testing.T) {
	result := NotOk(New(fmt.Errorf("result"), AppError, ErrUnknown))
	if !result.Ok {
		t.Logf("%s", result.Err.String())
		return
	}
	t.Fatalf("Result should not be ok")
}

func TestError(t *testing.T) {
	err := New(fmt.Errorf("error"), AppError, ErrUnknown)
	if err.code == ErrUnknown {
		t.Logf("%s", err.String())
		return
	}
	t.Fatalf("Err should be unknown")
}
