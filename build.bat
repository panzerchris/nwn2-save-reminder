@echo off
echo Building NWN2 Save Reminder...
go mod download
go build -o nwn2-save-reminder.exe
if %ERRORLEVEL% EQU 0 (
    echo Build successful! Run nwn2-save-reminder.exe to start.
) else (
    echo Build failed!
    pause
)

