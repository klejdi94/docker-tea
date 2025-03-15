@echo off
echo Building Docker Tea application...

REM Save current directory and move to root
pushd %~dp0\..

REM Build the application
go build -o docker-tea.exe ./cmd/docker-tea
if %ERRORLEVEL% NEQ 0 (
    echo Failed to build Docker Tea application
    popd
    exit /b %ERRORLEVEL%
)

echo Starting Docker Tea application...
echo.
echo Usage tips:
echo - Use TAB to switch between resources (Containers, Images, Volumes, Networks)
echo - When viewing a container, press 'm' to monitor resource usage in real-time
echo - Press '?' for help and to see all available keyboard shortcuts
echo.

REM Run the application
docker-tea.exe

REM Return to previous directory
popd