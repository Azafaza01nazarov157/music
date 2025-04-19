@echo off
echo Building Music Conveyor...

if exist cmd (
    set "PROJECT_ROOT=."
) else (
    set "PROJECT_ROOT=.."
)

if not exist %PROJECT_ROOT%\bin mkdir %PROJECT_ROOT%\bin

echo Building web app...
go build -o %PROJECT_ROOT%\bin\app.exe %PROJECT_ROOT%\cmd\app
if %ERRORLEVEL% neq 0 (
    echo Web app build failed!
    exit /b %ERRORLEVEL%

)
echo Building audio processor...
go build -o %PROJECT_ROOT%\bin\processor.exe %PROJECT_ROOT%\cmd\processor
if %ERRORLEVEL% neq 0 (
    echo Audio processor build failed!
    exit /b %ERRORLEVEL%
)

echo Build completed successfully!
echo App binary: %PROJECT_ROOT%\bin\app.exe
echo Processor binary: %PROJECT_ROOT%\bin\processor.exe
