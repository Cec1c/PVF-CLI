@echo off
setlocal
cd /d "%~dp0"
set "GOCACHE=%~dp0.cache\go-build"
go test ./...
if errorlevel 1 exit /b 1
go vet ./...
if errorlevel 1 exit /b 1
go build -trimpath -o dist\pvf.exe ./cmd/pvf
exit /b %errorlevel%
