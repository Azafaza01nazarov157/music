@echo off
cd /d %~dp0\..

echo Building Go project...
go build -o bin\app.exe main.go

if exist bin\app.exe (
    echo Running Go project...
    bin\app.exe
) else (
    echo Error: build failed!
)
