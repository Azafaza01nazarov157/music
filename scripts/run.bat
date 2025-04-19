@echo off
cd /d %~dp0\..
if exist bin\app.exe (
    echo Running Go project...
    bin\app.exe
)
if exist bin\processor.exe (
    echo Running Go project...
    bin\processor.exe
)
 else (
    echo Error: bin\app.exe not found! Please run build.bat first.
)
