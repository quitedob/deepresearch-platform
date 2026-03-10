@echo off
echo ========================================
echo Building Backend for Linux (amd64)...
echo ========================================

set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=0

go build -trimpath -ldflags="-s -w" -o server ./src/cmd/server

if %ERRORLEVEL% EQU 0 (
    echo [SUCCESS] Backend built: server
) else (
    echo [ERROR] Backend build failed!
    exit /b 1
)

echo Done!
