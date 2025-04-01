@echo off
echo Building Go project...
if not exist bin mkdir bin
go mod tidy
go build -o bin\app.exe main.go
if %ERRORLEVEL% NEQ 0 (
    echo Build failed!
    exit /b 1
)
echo Build completed. Binary is in bin\app.exe
