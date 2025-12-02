package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
)

const (
	quicksaveName    = "000000 - quicksave"
	backupFolderName = "backups"
	configFileName   = "config.json"
)

// Config holds all configuration settings
type Config struct {
	AlarmInterval  string `json:"alarm_interval"`   // Time before first alarm (e.g., "5m", "300s")
	DebounceDelay  string `json:"debounce_delay"`   // Wait time after file change (e.g., "3s")
	RepeatInterval string `json:"repeat_interval"`   // Time between repeat alarms (e.g., "5m")
	AlarmSoundFile string `json:"alarm_sound_file"`  // Path to audio file (empty = system beep)
	AlarmVolume    int    `json:"alarm_volume"`     // Alarm volume (0-100, default: 100)
	VerboseLogging bool   `json:"verbose_logging"`  // Enable verbose/debug logging
}

// DefaultConfig returns the default configuration
func DefaultConfig() Config {
	return Config{
		AlarmInterval:  "5m",
		DebounceDelay:  "3s",
		RepeatInterval: "5m",
		AlarmSoundFile: "",
		AlarmVolume:    100,
		VerboseLogging: false,
	}
}

type SaveReminder struct {
	savesPath      string
	backupsPath    string
	watcher        *fsnotify.Watcher
	lastSaveTime   time.Time
	alarmTimer     *time.Timer
	repeatTimer    *time.Ticker
	alarmActive    bool
	debounceTimer  *time.Timer
	config         Config
	verbose        bool
}

func main() {
	// Load configuration
	config, err := loadConfig()
	if err != nil {
		log.Printf("WARNING: Could not load config, using defaults: %v", err)
		config = DefaultConfig()
	}
	
	// Get the Documents folder path (handles custom locations)
	documentsPath, err := getDocumentsFolder()
	if err != nil {
		log.Printf("WARNING: Could not determine Documents folder, using default: %v", err)
		// Fallback to standard location
		documentsPath = filepath.Join(os.Getenv("USERPROFILE"), "Documents")
	}
	
	// Get the saves folder path
	savesPath := filepath.Join(documentsPath, "Neverwinter Nights 2", "saves", "multiplayer")
	
	log.Printf("NWN2 Save Reminder starting...")
	log.Printf("Documents folder: %s", documentsPath)
	log.Printf("Watching folder: %s", savesPath)
	log.Printf("Configuration loaded from: %s", getConfigPath())
	
	// Print configuration
	printConfig(config)
	
	// Check if folder exists
	if _, err := os.Stat(savesPath); os.IsNotExist(err) {
		log.Printf("ERROR: Saves folder does not exist: %s", savesPath)
		log.Printf("")
		log.Printf("Please make sure:")
		log.Printf("1. Neverwinter Nights 2 has been launched at least once")
		log.Printf("2. You have created a multiplayer save at least once")
		log.Printf("3. The folder path is correct")
		pauseBeforeExit("")
		os.Exit(1)
	}
	
	// Create backups folder
	backupsPath := filepath.Join(savesPath, backupFolderName)
	if err := os.MkdirAll(backupsPath, 0755); err != nil {
		log.Printf("ERROR: Failed to create backups folder: %v", err)
		pauseBeforeExit("")
		os.Exit(1)
	}
	
	// Create watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("ERROR: Failed to create file watcher: %v", err)
		pauseBeforeExit("")
		os.Exit(1)
	}
	defer watcher.Close()
	
	reminder := &SaveReminder{
		savesPath:   savesPath,
		backupsPath: backupsPath,
		watcher:     watcher,
		config:      config,
		verbose:     config.VerboseLogging,
	}
	
	// Find the quicksave folder
	quicksaveFolder := filepath.Join(savesPath, quicksaveName)
	if _, err := os.Stat(quicksaveFolder); os.IsNotExist(err) {
		log.Printf("WARNING: Quicksave folder does not exist yet: %s", quicksaveFolder)
		log.Printf("The watcher will start monitoring once the folder is created.")
	} else {
		log.Printf("Found quicksave folder: %s", quicksaveFolder)
	}
	
	// Add both the saves folder (to detect new folders) and quicksave folder (to detect changes)
	if err := watcher.Add(savesPath); err != nil {
		log.Printf("ERROR: Failed to add saves folder to watcher: %v", err)
		pauseBeforeExit("")
		os.Exit(1)
	}
	
	// Also watch the quicksave folder if it exists (for changes within it)
	if _, err := os.Stat(quicksaveFolder); err == nil {
		if err := watcher.Add(quicksaveFolder); err != nil {
			log.Printf("WARNING: Failed to add quicksave folder to watcher: %v", err)
		} else {
			log.Printf("Watching quicksave folder for changes")
		}
	}
	
	// List existing save folders for debugging
	log.Printf("")
	log.Printf("Current save folders:")
	files, err := os.ReadDir(savesPath)
	if err != nil {
		log.Printf("Warning: Could not read folder contents: %v", err)
	} else {
		if len(files) == 0 {
			log.Printf("  (folder is empty)")
		} else {
			for _, file := range files {
				if file.IsDir() && file.Name() != backupFolderName {
					log.Printf("  - %s (folder)", file.Name())
				}
			}
		}
	}
	log.Printf("")
	log.Printf("File watcher initialized. Waiting for save file changes...")
	log.Printf("Press Ctrl+C to exit")
	if config.VerboseLogging {
		log.Printf("(Verbose logging enabled: All file events will be logged)")
	}
	
	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	
	// Process events in a goroutine
	go reminder.processEvents()
	
	// Wait for interrupt signal
	<-sigChan
	log.Printf("")
	log.Printf("Shutting down...")
	reminder.cleanup()
	log.Printf("Goodbye!")
	pauseBeforeExit("")
}

// getConfigPath returns the path to the config file (in the same directory as the executable)
func getConfigPath() string {
	exePath, err := os.Executable()
	if err != nil {
		// Fallback to current directory
		return configFileName
	}
	exeDir := filepath.Dir(exePath)
	return filepath.Join(exeDir, configFileName)
}

// getExecutableDir returns the directory where the executable is located
func getExecutableDir() string {
	exePath, err := os.Executable()
	if err != nil {
		// Fallback to current directory
		wd, _ := os.Getwd()
		return wd
	}
	return filepath.Dir(exePath)
}

// resolveSoundPath resolves the sound file path, supporting both absolute and relative paths
// Relative paths are resolved relative to the executable directory
func (sr *SaveReminder) resolveSoundPath(path string) string {
	// If path is empty, return empty
	if path == "" {
		return ""
	}
	
	// Check if it's an absolute path (works on both Windows and Unix)
	if filepath.IsAbs(path) {
		// Try absolute path as-is
		if _, err := os.Stat(path); err == nil {
			return path
		}
		return ""
	}
	
	// Relative path - try relative to executable directory first (most common case)
	exeDir := getExecutableDir()
	relativePath := filepath.Join(exeDir, path)
	if _, err := os.Stat(relativePath); err == nil {
		return relativePath
	}
	
	// Also try relative to current working directory (for command-line usage)
	if _, err := os.Stat(path); err == nil {
		return path
	}
	
	return ""
}

// loadConfig loads configuration from a JSON file, or creates a default one if it doesn't exist
func loadConfig() (Config, error) {
	configPath := getConfigPath()
	
	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create default config file
		defaultConfig := DefaultConfig()
		if err := saveConfig(defaultConfig); err != nil {
			return defaultConfig, fmt.Errorf("failed to create default config file: %v", err)
		}
		log.Printf("Created default config file: %s", configPath)
		return defaultConfig, nil
	}
	
	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return DefaultConfig(), fmt.Errorf("failed to read config file: %v", err)
	}
	
	// Parse JSON
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return DefaultConfig(), fmt.Errorf("failed to parse config file: %v", err)
	}
	
	// Validate and set defaults for empty values
	if config.AlarmInterval == "" {
		config.AlarmInterval = "5m"
	}
	if config.DebounceDelay == "" {
		config.DebounceDelay = "3s"
	}
	if config.RepeatInterval == "" {
		config.RepeatInterval = "5m"
	}
	// AlarmSoundFile can be empty (uses system beep)
	// Validate alarm volume (0-100)
	if config.AlarmVolume < 0 {
		config.AlarmVolume = 0
	} else if config.AlarmVolume > 100 {
		config.AlarmVolume = 100
	}
	// If volume is 0 (unset in JSON), default to 100
	if config.AlarmVolume == 0 && config.AlarmSoundFile == "" {
		// Only set to 100 if it's actually 0 and no sound file (might be intentional mute)
		// But if it's 0 in JSON, it means user set it, so keep it
	}
	// VerboseLogging defaults to false if not set
	
	return config, nil
}

// printConfig prints the current configuration in a readable format
func printConfig(config Config) {
	log.Printf("")
	log.Printf("=== Configuration ===")
	log.Printf("Alarm Interval:    %s", config.AlarmInterval)
	log.Printf("Debounce Delay:   %s", config.DebounceDelay)
	log.Printf("Repeat Interval:   %s", config.RepeatInterval)
	if config.AlarmSoundFile != "" {
		log.Printf("Alarm Sound File: %s", config.AlarmSoundFile)
	} else {
		log.Printf("Alarm Sound File: (system beep)")
	}
	log.Printf("Alarm Volume:      %d%%", config.AlarmVolume)
	log.Printf("Verbose Logging:   %v", config.VerboseLogging)
	log.Printf("===================")
	log.Printf("")
}

// saveConfig saves the configuration to a JSON file
func saveConfig(config Config) error {
	configPath := getConfigPath()
	
	// Marshal to JSON with indentation for readability
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}
	
	// Write to file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}
	
	return nil
}

// getDocumentsFolder gets the actual Documents folder path on Windows
// This handles cases where the Documents folder has been moved to a custom location
func getDocumentsFolder() (string, error) {
	if runtime.GOOS != "windows" {
		// On non-Windows, use standard home directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(homeDir, "Documents"), nil
	}
	
	// On Windows, use PowerShell to get the actual Documents folder path
	// This uses the Windows Shell API to get the real location, even if moved
	cmd := exec.Command("powershell", "-Command", "[Environment]::GetFolderPath('MyDocuments')")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get Documents folder: %v", err)
	}
	
	// Clean up the output (remove newlines and whitespace)
	path := strings.TrimSpace(string(output))
	if path == "" {
		return "", fmt.Errorf("Documents folder path is empty")
	}
	
	return path, nil
}

// pauseBeforeExit pauses execution so the user can read error messages
// when running as a double-clickable executable on Windows
func pauseBeforeExit(message string) {
	if runtime.GOOS == "windows" {
		if message != "" {
			fmt.Println("")
			fmt.Println(message)
		}
		// Use cmd to pause (works even when double-clicked)
		// The pause command will print its own "Press any key to continue . . ." message
		cmd := exec.Command("cmd", "/C", "pause")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
	}
}

func (sr *SaveReminder) processEvents() {
	for {
		select {
		case event, ok := <-sr.watcher.Events:
			if !ok {
				return
			}
			
			// Log all file events for debugging (only if verbose)
			if sr.verbose {
				log.Printf("File event detected: %s (op: %s)", event.Name, event.Op.String())
			}
			
			// Check if this event is related to the quicksave folder
			if sr.isQuicksaveRelated(event.Name) {
				if sr.verbose {
					log.Printf("Quicksave-related change detected: %s", event.Name)
				}
				sr.handleQuicksaveChange(event)
			} else {
				if sr.verbose {
					log.Printf("Ignored (not quicksave): %s", filepath.Base(event.Name))
				}
			}
			
		case err, ok := <-sr.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)
		}
	}
}

func (sr *SaveReminder) cleanup() {
	// Stop all timers
	sr.resetAlarmTimers()
	
	// Close watcher
	if sr.watcher != nil {
		sr.watcher.Close()
	}
}

func (sr *SaveReminder) isQuicksaveRelated(filePath string) bool {
	// Check if the path contains the quicksave folder
	// This handles both the folder itself and files within it
	relPath, err := filepath.Rel(sr.savesPath, filePath)
	if err != nil {
		return false
	}
	
	// Check if path starts with "000000 - quicksave" (the folder name)
	parts := strings.Split(relPath, string(filepath.Separator))
	if len(parts) > 0 && parts[0] == quicksaveName {
		return true
	}
	
	return false
}

func (sr *SaveReminder) handleQuicksaveChange(event fsnotify.Event) {
	// Skip if it's the folder itself being created/removed (we want file changes inside)
	info, err := os.Stat(event.Name)
	if err == nil && info.IsDir() {
		// If the quicksave folder was just created, add it to the watcher
		if event.Op&fsnotify.Create != 0 {
			quicksaveFolder := filepath.Join(sr.savesPath, quicksaveName)
			if event.Name == quicksaveFolder {
				log.Printf("Quicksave folder created, adding to watcher...")
				if err := sr.watcher.Add(quicksaveFolder); err != nil {
					log.Printf("Warning: Failed to add quicksave folder to watcher: %v", err)
				}
			}
		}
		return
	}
	
	// Only process write/create events for files (not remove, not directories)
	if event.Op&fsnotify.Write == 0 && event.Op&fsnotify.Create == 0 {
		return
	}
	
	// Cancel existing debounce timer if any
	if sr.debounceTimer != nil {
		sr.debounceTimer.Stop()
	}
	
	// Parse debounce delay from config
	debounceDelay, err := time.ParseDuration(sr.config.DebounceDelay)
	if err != nil {
		log.Printf("Warning: Invalid debounce_delay in config, using 3s: %v", err)
		debounceDelay = 3 * time.Second
	}
	
	// Start debounce timer
	sr.debounceTimer = time.AfterFunc(debounceDelay, func() {
		sr.processQuicksave(filepath.Join(sr.savesPath, quicksaveName))
	})
	
	log.Printf("Detected change in quicksave folder, waiting %v before processing...", debounceDelay)
}

func (sr *SaveReminder) processQuicksave(quicksaveFolderPath string) {
	log.Printf("Processing quicksave folder: %s", quicksaveFolderPath)
	
	// Check if folder exists
	if _, err := os.Stat(quicksaveFolderPath); os.IsNotExist(err) {
		log.Printf("Quicksave folder no longer exists, skipping backup")
		return
	}
	
	// Create backup of the entire folder
	if err := sr.createBackup(quicksaveFolderPath); err != nil {
		log.Printf("Error creating backup: %v", err)
		return
	}
	
	// Reset alarm timers
	sr.resetAlarmTimers()
	
	// Update last save time
	sr.lastSaveTime = time.Now()
	log.Printf("Save processed successfully. Alarm timer reset.")
	
	// Start new alarm timer
	sr.startAlarmTimer()
}

func (sr *SaveReminder) createBackup(quicksaveFolderPath string) error {
	// Create timestamp folder
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	backupFolderName := fmt.Sprintf("%s - %s", timestamp, quicksaveName)
	destFolder := filepath.Join(sr.backupsPath, backupFolderName)
	
	if err := os.MkdirAll(destFolder, 0755); err != nil {
		return fmt.Errorf("error creating backup folder: %v", err)
	}
	
	// Copy the entire quicksave folder recursively
	return sr.copyDirectory(quicksaveFolderPath, destFolder)
}

func (sr *SaveReminder) copyDirectory(src, dst string) error {
	// Get source info
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("error reading source: %v", err)
	}
	
	// Create destination directory
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return fmt.Errorf("error creating destination directory: %v", err)
	}
	
	// Read source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("error reading source directory: %v", err)
	}
	
	// Copy each entry
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		
		if entry.IsDir() {
			// Recursively copy subdirectories
			if err := sr.copyDirectory(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Copy file
			if err := sr.copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	
	log.Printf("Backup created: %s", dst)
	return nil
}

func (sr *SaveReminder) copyFile(src, dst string) error {
	// Read source file
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("error reading source file: %v", err)
	}
	
	// Write to destination
	if err := os.WriteFile(dst, data, 0644); err != nil {
		return fmt.Errorf("error writing backup file: %v", err)
	}
	
	return nil
}

func (sr *SaveReminder) resetAlarmTimers() {
	// Stop and clear existing timers
	if sr.alarmTimer != nil {
		sr.alarmTimer.Stop()
		sr.alarmTimer = nil
	}
	if sr.repeatTimer != nil {
		sr.repeatTimer.Stop()
		sr.repeatTimer = nil
	}
	sr.alarmActive = false
}

func (sr *SaveReminder) startAlarmTimer() {
	// Parse alarm interval from config
	alarmInterval, err := time.ParseDuration(sr.config.AlarmInterval)
	if err != nil {
		log.Printf("Warning: Invalid alarm_interval in config, using 5m: %v", err)
		alarmInterval = 5 * time.Minute
	}
	
	// Start the initial alarm timer
	sr.alarmTimer = time.AfterFunc(alarmInterval, func() {
		sr.triggerAlarm()
		sr.startRepeatAlarm()
	})
	
	log.Printf("Alarm timer started. Will alert in %v if no new save is made.", alarmInterval)
}

func (sr *SaveReminder) startRepeatAlarm() {
	sr.alarmActive = true
	
	// Parse repeat interval from config
	repeatInterval, err := time.ParseDuration(sr.config.RepeatInterval)
	if err != nil {
		log.Printf("Warning: Invalid repeat_interval in config, using 5m: %v", err)
		repeatInterval = 5 * time.Minute
	}
	
	// Start repeating alarm
	sr.repeatTimer = time.NewTicker(repeatInterval)
	go func() {
		for range sr.repeatTimer.C {
			sr.triggerAlarm()
		}
	}()
}

func (sr *SaveReminder) triggerAlarm() {
	log.Printf("*** ALARM: Time to save! It's been %v since last save. ***", time.Since(sr.lastSaveTime))
	
	// Play alarm sound
	sr.playAlarmSound()
}

func (sr *SaveReminder) playAlarmSound() {
	// Check if volume is 0 (muted)
	if sr.config.AlarmVolume == 0 {
		if sr.verbose {
			log.Printf("Alarm volume is 0, alarm is muted")
		}
		return
	}
	
	if sr.config.AlarmSoundFile != "" {
		// Try to find the audio file
		// Supports both absolute paths and relative paths (relative to executable directory)
		soundPath := sr.resolveSoundPath(sr.config.AlarmSoundFile)
		if soundPath != "" {
			sr.playAudioFile(soundPath)
			return
		}
		log.Printf("Warning: Audio file not found: %s, using system beep instead", sr.config.AlarmSoundFile)
	}
	
	// Default: Use system beep
	// Note: System beep volume can't be easily controlled, but we can skip it if volume is very low
	if sr.config.AlarmVolume < 10 {
		// Very low volume, skip beep
		return
	}
	
	if runtime.GOOS == "windows" {
		// Windows: Use PowerShell to play a beep
		cmd := exec.Command("powershell", "-Command", "[console]::beep(800, 500)")
		if err := cmd.Run(); err != nil {
			// Fallback to console beep
			fmt.Print("\a")
		}
	} else {
		// Unix-like: Use console beep
		fmt.Print("\a")
	}
}

func (sr *SaveReminder) playAudioFile(filePath string) {
	if runtime.GOOS == "windows" {
		// Windows: Use PowerShell with Windows Media Player COM object for volume control
		absPath, err := filepath.Abs(filePath)
		if err != nil {
			absPath = filePath
		}
		// Escape backslashes and quotes for PowerShell
		absPath = strings.ReplaceAll(absPath, `\`, `\\`)
		absPath = strings.ReplaceAll(absPath, `"`, `\"`)
		
		// Calculate volume (Windows Media Player uses 0-100)
		volume := sr.config.AlarmVolume
		if volume > 100 {
			volume = 100
		} else if volume < 0 {
			volume = 0
		}
		
		// Use Windows Media Player COM object for better volume control
		// This works for WAV, MP3, and other formats
		psScript := fmt.Sprintf(`
$player = New-Object -ComObject WMPlayer.OCX
$player.settings.volume = %d
$player.URL = "%s"
$player.controls.play()
while ($player.playState -eq 3) {
	Start-Sleep -Milliseconds 100
}
$player.controls.stop()
$player.close()
`, volume, absPath)
		
		cmd := exec.Command("powershell", "-Command", psScript)
		if err := cmd.Run(); err != nil {
			// Fallback: Try SoundPlayer for WAV files (no volume control)
			ext := strings.ToLower(filepath.Ext(filePath))
			if ext == ".wav" {
				cmd = exec.Command("powershell", "-Command", fmt.Sprintf(`[System.Media.SoundPlayer]::new("%s").PlaySync()`, absPath))
				if err := cmd.Run(); err != nil {
					log.Printf("Error playing audio file: %v", err)
				}
			} else {
				// For other formats, try default program (no volume control)
				cmd = exec.Command("cmd", "/C", "start", "/MIN", filePath)
				if err := cmd.Run(); err != nil {
					log.Printf("Error playing audio file: %v", err)
				}
			}
		}
	} else {
		// Unix-like: Use aplay, paplay, or similar
		// Volume control would require additional tools
		cmd := exec.Command("aplay", filePath)
		if err := cmd.Run(); err != nil {
			// Try alternative
			cmd = exec.Command("paplay", filePath)
			if err := cmd.Run(); err != nil {
				log.Printf("Error playing audio file: %v", err)
			}
		}
	}
}

