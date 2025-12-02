# NWN2 Save Reminder

A Windows application that monitors your Neverwinter Nights 2 multiplayer saves folder and reminds you to save your game.

## Features

- **Automatic Backup**: When you quicksave, the application automatically creates a timestamped backup
- **Save Reminder**: Alerts you every 5 minutes if you haven't saved recently
- **File Watching**: Monitors the saves folder in real-time
- **Terminal Logging**: All activity is logged to the console window

## Requirements

- Windows 10/11
- Go 1.21 or later (for building from source)

## Installation

### Option 1: Build from Source

1. **Install Go** (if not already installed):
   - Download from https://go.dev/dl/
   - Run the installer
   - Verify installation: `go version`

2. **Build the application**:
   ```bash
   go mod download
   go build -o nwn2-save-reminder.exe
   ```

3. **Run the application**:
   ```bash
   .\nwn2-save-reminder.exe
   ```

### Option 2: Use Pre-built Executable

If a pre-built executable is available, simply run `nwn2-save-reminder.exe`.

## How It Works

1. The application watches: `My Documents\Neverwinter Nights 2\saves\multiplayer`
2. When it detects changes to `000000 - quicksave` (any extension):
   - Waits 3 seconds (to ensure the file is fully written)
   - Creates a backup in `backups\YYYY-MM-DD_HH-MM-SS\000000 - quicksave`
   - Resets the alarm timer
3. After 5 minutes without a new save:
   - Plays an alarm sound (system beep by default)
   - Repeats every 5 minutes until you save again

## Configuration

The application uses a `config.json` file in the same directory as the executable. On first run, a default configuration file will be created automatically.

### Config File Location

The config file is created in the same directory as the executable:
```
nwn2-save-reminder.exe
config.json
```

### Configuration Options

Edit `config.json` to customize the behavior:

```json
{
  "alarm_interval": "5m",
  "debounce_delay": "3s",
  "repeat_interval": "5m",
  "alarm_sound_file": "",
  "alarm_volume": 100,
  "verbose_logging": false
}
```

**Settings:**
- `alarm_interval`: Time before first alarm (e.g., `"5m"`, `"300s"`, `"10m"`)
- `debounce_delay`: Wait time after file change before processing (e.g., `"3s"`, `"5s"`)
- `repeat_interval`: Time between repeat alarms (e.g., `"5m"`, `"10m"`)
- `alarm_sound_file`: Path to audio file (empty string = system beep, supports WAV and MP3 formats)
- `alarm_volume`: Alarm volume level (0-100, default: 100)
  - `100` = Full volume (as loud as system allows)
  - `50` = Half volume
  - `0` = Muted (no alarm sound)
- `verbose_logging`: Enable detailed debug logging (`true` or `false`, default: `false`)

**Time Format:**
- Use Go duration format: `"5m"` (5 minutes), `"30s"` (30 seconds), `"1h"` (1 hour)
- Examples: `"5m"`, `"300s"`, `"10m"`, `"1h30m"`

**Sound File Path:**
- Supports both absolute and relative paths
- **Relative paths** are resolved relative to the executable directory (where `nwn2-save-reminder.exe` is located)
- **Absolute paths** work as-is (e.g., `"C:\\Sounds\\alarm.wav"`)
- Examples:
  - `"alarm.wav"` - looks for `alarm.wav` in the same directory as the executable
  - `"sounds\\alarm.wav"` - looks in a `sounds` subdirectory relative to the executable
  - `"C:\\Sounds\\alarm.wav"` - absolute path (Windows)
  - `"D:/Audio/alarm.wav"` - absolute path (alternative Windows format)

### Custom Audio File

To use a custom alarm sound:

1. Place a WAV or MP3 audio file in the same directory as the executable, or provide a full path
2. Edit `config.json` and set `alarm_sound_file`:
   ```json
   {
     "alarm_sound_file": "alarm.wav"
   }
   ```
   Or use an absolute path:
   ```json
   {
     "alarm_sound_file": "C:\\Sounds\\alarm.mp3"
   }
   ```
3. Restart the application (no rebuild needed!)

### Verbose Logging

By default, the application only logs important events (saves, backups, alarms). To see all file system events for debugging:

1. Edit `config.json` and set `verbose_logging` to `true`:
   ```json
   {
     "verbose_logging": true
   }
   ```
2. Restart the application
3. You'll now see detailed logs for every file event detected

**Note:** The configuration is displayed on startup, so you can verify your settings are loaded correctly.

### Alarm Volume

Control the alarm volume with the `alarm_volume` setting:

- **100** (default): Full volume - alarm plays at maximum system volume
- **50**: Half volume - alarm plays at 50% of maximum
- **25**: Quarter volume - quieter alarm
- **0**: Muted - no alarm sound (useful for silent mode)

**Note:** Volume control works best with custom audio files. System beep volume cannot be controlled and will be skipped if volume is set below 10.

## Usage

1. Start the application (double-click or run from command line)
2. A terminal window will open showing logs
3. Play Neverwinter Nights 2 as normal
4. When you quicksave, you'll see a log message and a backup will be created
5. If you don't save for 5 minutes, you'll hear an alarm
6. Press `Ctrl+C` to exit

## Backup Location

Backups are stored in:
```
My Documents\Neverwinter Nights 2\saves\multiplayer\backups\YYYY-MM-DD_HH-MM-SS\
```

Each backup folder contains a timestamp of when the save was made.

## Troubleshooting

**"Saves folder does not exist" error:**
- Make sure you've launched Neverwinter Nights 2 at least once to create the saves folder
- Check that the path is correct: `My Documents\Neverwinter Nights 2\saves\multiplayer`

**Alarm not playing:**
- The default uses Windows system beep
- If you want a custom sound, set `alarmSoundFile` in the code

**Application not detecting saves:**
- Make sure you're saving to the multiplayer folder
- Check that the quicksave file name starts with "000000 - quicksave"
- Verify the terminal window shows "File watcher initialized" message

## License

This is a personal utility tool. Use at your own discretion.

