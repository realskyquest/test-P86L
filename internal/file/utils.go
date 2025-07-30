package file

import (
	"encoding/json"
	"p86l/configs"
	pd "p86l/internal/debug"
	"path/filepath"
)

// /Project-86-Launcher/
func (a *AppFS) PathDirApp() string {
	return configs.AppName
}

// /build/
func (a *AppFS) PathDirBuild() string {
	return "build"
}

// /build/game
func (a *AppFS) PathDirGame() string {
	return filepath.Join(a.PathDirBuild(), "game")
}

// /build/prerelease
func (a *AppFS) PathDirPrerelease() string {
	return filepath.Join(a.PathDirBuild(), "prerelease")
}

// -- files --

// build/game/Project-86.exe
func (a *AppFS) PathFileGame() string {
	return filepath.Join(a.PathDirGame(), "Project-86.exe")
}

// build/game/Project-86.exe
func (a *AppFS) PathFilePrerelease() string {
	return filepath.Join(a.PathDirPrerelease(), "Project-86.exe")
}

// /Project-86-Launcher/log.txt
func (a *AppFS) PathFileLog() string {
	return filepath.Join(a.PathDirApp(), "log.txt")
}

// /Project-86-Launcher/data.json
func (a *AppFS) PathFileData() string {
	return filepath.Join(a.PathDirApp(), configs.DataFile)
}

// /Project-86-Launcher/cache.json
func (a *AppFS) PathFileCache() string {
	return filepath.Join(a.PathDirApp(), configs.CacheFile)
}

// -- Used to convert data to bytes.

func (a *AppFS) EncodeData(appDebug *pd.Debug, d Data) (pd.Result, []byte) {
	b, err := json.Marshal(d)
	if err != nil {
		return pd.NotOk(pd.New(err, pd.FSError, pd.ErrDataSave)), nil
	}
	return pd.Ok(), b
}

func (a *AppFS) EncodeCache(appDebug *pd.Debug, c Cache) (pd.Result, []byte) {
	b, err := json.Marshal(c)
	if err != nil {
		return pd.NotOk(pd.New(err, pd.FSError, pd.ErrDataSave)), nil
	}
	return pd.Ok(), b
}

// -- Used to get data and use it for app.

// Get data.
func (a *AppFS) DecodeData(appDebug *pd.Debug, b []byte) (pd.Result, Data) {
	var d Data
	err := json.Unmarshal(b, &d)
	if err != nil {
		return pd.NotOk(pd.New(err, pd.FSError, pd.ErrDataLoad)), d
	}
	return pd.Ok(), d
}

// Get cache.
func (a *AppFS) DecodeCache(appDebug *pd.Debug, b []byte) (pd.Result, Cache) {
	var c Cache
	err := json.Unmarshal(b, &c)
	if err != nil {
		return pd.NotOk(pd.New(err, pd.FSError, pd.ErrCacheLoad)), c
	}
	return pd.Ok(), c
}
