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

package file_test

import (
	pd "p86l/internal/debug"
	"p86l/internal/file"
	"testing"

	"github.com/hajimehoshi/guigui"
)

func setup(t *testing.T) (*pd.Debug, *file.AppFS) {
	e := &pd.Debug{}
	result, a := file.NewFS("test")
	if !result.Ok {
		t.Fatalf("%s", result.Err.String())
	}

	return e, a
}

func TestInit(t *testing.T) {
	_, fs := setup(t)
	t.Logf("%#v", fs)
}

func TestSaveFiles(t *testing.T) {
	e, fs := setup(t)

	exampleData := file.Data{
		Locale:    "fr",
		AppScale:  2,
		ColorMode: guigui.ColorModeDark,
	}

	result, b := fs.EncodeData(e, exampleData)
	if !result.Ok {
		t.Fatalf("%s", result.Err.String())
	}
	result = fs.Save(e, fs.PathFileData(), b)
	if !result.Ok {
		t.Fatalf("%s", result.Err.String())
	}
}

func TestLoadFiles(t *testing.T) {
	e, fs := setup(t)

	result, b := fs.Load(e, fs.PathFileData())
	if !result.Ok {
		t.Fatalf("%s", result.Err.String())
	}
	result, d := fs.DecodeData(e, b)
	if !result.Ok {
		t.Fatalf("%s", result.Err.String())
	}

	t.Logf("%#v", d)
}

func TestStatFile(t *testing.T) {
	_, fs := setup(t)

	if result := fs.ExistsRoot(fs.PathFileData()); !result.Ok {
		t.Fatalf("%s", result.Err.String())
	}
}
