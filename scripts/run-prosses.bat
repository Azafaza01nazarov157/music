@echo off
cd /d %~dp0\..

echo Building Go project...
go build -o bin\processor.exe ./cmd/processor

if exist bin\processor.exe (
    echo Running Go project...
    bin\processor.exe
) else (
    echo Error: build failed!
)
