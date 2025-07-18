@echo off
setlocal enabledelayedexpansion

:: Set working directory to where the batch file is located
cd /d "%~dp0"

echo Searching for Project-86-Launcher executable...

:: Clear any existing value
set "launcher_exe="

:: Method 1: Direct wildcard match
for /f "delims=" %%f in ('dir /b "Project-86-Launcher-*.exe" 2^>nul') do (
    echo %%f | find /i "old" >nul || (
        set "launcher_exe=%%f"
        goto :found
    )
)

:: Method 2: Fallback using full directory scan
if not defined launcher_exe (
    for /f "delims=" %%f in ('dir /b /s "Project-86-Launcher-*.exe" 2^>nul') do (
        echo %%f | find /i "old" >nul || (
            set "launcher_exe=%%f"
            goto :found
        )
    )
)

:found
if defined launcher_exe (
    echo Found executable: !launcher_exe!
    start "" "!launcher_exe!"
) else (
    echo ERROR: Could not find Project-86-Launcher executable
    echo.
    echo Files in current directory:
    dir /b
)
