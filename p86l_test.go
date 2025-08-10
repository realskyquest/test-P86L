package p86l_test

import (
	"p86l"
	"p86l/internal/file"
	"testing"
	"time"
)

func TestCreateLogFiles(t *testing.T) {
	result, fs := file.NewFS("test")
	if !result.Ok {
		t.Fatalf("Failed to create root: %v", result.Err)
	}
	defer func() {
		err := fs.Root.Close()
		if err != nil {
			t.Fatalf("Failed to close root: %v", err)
		}
	}()

	for range 20 {
		time.Sleep(500 * time.Millisecond)
		logFile, err := p86l.NewLogFile(fs.Root, fs.PathDirLogs())
		if err != nil {
			t.Fatalf("Failed to create log file %s: %v", logFile.Name(), err)
		}
		defer func() {
			err = logFile.Close()
			if err != nil {
				t.Fatalf("Failed to close log file %s: %v", logFile.Name(), err)
			}
		}()
	}
	time.Sleep(time.Second)
}

func TestRotateLogs(t *testing.T) {
	result, fs := file.NewFS("test")
	if !result.Ok {
		t.Fatalf("Failed to create root: %v", result.Err)
	}
	defer func() {
		err := fs.Root.Close()
		if err != nil {
			t.Fatalf("Failed to close root: %v", err)
		}
	}()

	err := p86l.RotateLogFiles(fs.Root, fs.CompanyDirPath, fs.PathDirLogs())
	if err != nil {
		t.Fatalf("Failed to rotate log files: %v", err)
	}
}
