@echo off
echo Building NWN2 Save Reminder...
echo Downloading dependencies...
go mod tidy
go mod download
echo Building executable...
go build -o nwn2-save-reminder.exe
if %ERRORLEVEL% EQU 0 (
    echo Build successful! Run nwn2-save-reminder.exe to start.
) else (
    echo Build failed!
    pause
)

