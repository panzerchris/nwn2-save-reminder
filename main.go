package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

const (
	quicksaveName    = "000000 - quicksave"
	backupFolderName = "backups"
	alarmInterval    = 5 * time.Minute
	debounceDelay    = 3 * time.Second
	repeatInterval   = 5 * time.Minute
	alarmSoundFile   = "" // Empty = use system beep, or set path to WAV/MP3 file
)

type SaveReminder struct {
	savesPath      string
	backupsPath    string
	watcher        *fsnotify.Watcher
	lastSaveTime   time.Time
	alarmTimer     *time.Timer
	repeatTimer    *time.Ticker
	alarmActive    bool
	debounceTimer  *time.Timer
}

func main() {
	// Get the saves folder path
	savesPath := filepath.Join(os.Getenv("USERPROFILE"), "Documents", "Neverwinter Nights 2", "saves", "multiplayer")
	
	log.Printf("NWN2 Save Reminder starting...")
	log.Printf("Watching folder: %s", savesPath)
	
	// Check if folder exists
	if _, err := os.Stat(savesPath); os.IsNotExist(err) {
		log.Fatalf("Error: Saves folder does not exist: %s", savesPath)
	}
	
	// Create backups folder
	backupsPath := filepath.Join(savesPath, backupFolderName)
	if err := os.MkdirAll(backupsPath, 0755); err != nil {
		log.Fatalf("Error creating backups folder: %v", err)
	}
	
	// Create watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("Error creating file watcher: %v", err)
	}
	defer watcher.Close()
	
	reminder := &SaveReminder{
		savesPath:   savesPath,
		backupsPath: backupsPath,
		watcher:     watcher,
	}
	
	// Add the saves folder to watcher
	if err := watcher.Add(savesPath); err != nil {
		log.Fatalf("Error adding folder to watcher: %v", err)
	}
	
	log.Printf("File watcher initialized. Waiting for save file changes...")
	log.Printf("Press Ctrl+C to exit")
	
	// Process events
	reminder.processEvents()
}

func (sr *SaveReminder) processEvents() {
	for {
		select {
		case event, ok := <-sr.watcher.Events:
			if !ok {
				return
			}
			
			// Check if this is the quicksave file
			if sr.isQuicksaveFile(event.Name) {
				sr.handleQuicksaveChange(event)
			}
			
		case err, ok := <-sr.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)
		}
	}
}

func (sr *SaveReminder) isQuicksaveFile(filePath string) bool {
	fileName := filepath.Base(filePath)
	// Check if filename starts with "000000 - quicksave"
	return strings.HasPrefix(fileName, quicksaveName)
}

func (sr *SaveReminder) handleQuicksaveChange(event fsnotify.Event) {
	// Only process write/create events (not remove)
	if event.Op&fsnotify.Write == 0 && event.Op&fsnotify.Create == 0 {
		return
	}
	
	// Cancel existing debounce timer if any
	if sr.debounceTimer != nil {
		sr.debounceTimer.Stop()
	}
	
	// Start debounce timer
	sr.debounceTimer = time.AfterFunc(debounceDelay, func() {
		sr.processQuicksave(event.Name)
	})
	
	log.Printf("Detected change to quicksave file, waiting %v before processing...", debounceDelay)
}

func (sr *SaveReminder) processQuicksave(filePath string) {
	log.Printf("Processing quicksave: %s", filePath)
	
	// Check if file exists (might have been deleted)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Printf("File no longer exists, skipping backup")
		return
	}
	
	// Create backup
	if err := sr.createBackup(filePath); err != nil {
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

func (sr *SaveReminder) createBackup(filePath string) error {
	// Get file info to check if it's actually a file (not a directory)
	info, err := os.Stat(filePath)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a file")
	}
	
	// Create timestamp folder
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	timestampFolder := filepath.Join(sr.backupsPath, timestamp)
	if err := os.MkdirAll(timestampFolder, 0755); err != nil {
		return fmt.Errorf("error creating timestamp folder: %v", err)
	}
	
	// Copy file to backup location
	fileName := filepath.Base(filePath)
	destPath := filepath.Join(timestampFolder, fileName)
	
	// Read source file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading source file: %v", err)
	}
	
	// Write to destination
	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return fmt.Errorf("error writing backup file: %v", err)
	}
	
	log.Printf("Backup created: %s", destPath)
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
	// Start the initial 5-minute timer
	sr.alarmTimer = time.AfterFunc(alarmInterval, func() {
		sr.triggerAlarm()
		sr.startRepeatAlarm()
	})
	
	log.Printf("Alarm timer started. Will alert in %v if no new save is made.", alarmInterval)
}

func (sr *SaveReminder) startRepeatAlarm() {
	sr.alarmActive = true
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
	if alarmSoundFile != "" {
		// Play custom audio file
		if _, err := os.Stat(alarmSoundFile); err == nil {
			sr.playAudioFile(alarmSoundFile)
			return
		}
		log.Printf("Warning: Audio file not found: %s, using system beep instead", alarmSoundFile)
	}
	
	// Default: Use system beep
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
		// Windows: Use PowerShell SoundPlayer for WAV files (most reliable)
		ext := strings.ToLower(filepath.Ext(filePath))
		if ext == ".wav" {
			// Use .NET SoundPlayer via PowerShell (works well for WAV files)
			absPath, err := filepath.Abs(filePath)
			if err != nil {
				absPath = filePath
			}
			// Escape backslashes for PowerShell
			absPath = strings.ReplaceAll(absPath, `\`, `\\`)
			cmd := exec.Command("powershell", "-Command", fmt.Sprintf(`[System.Media.SoundPlayer]::new("%s").PlaySync()`, absPath))
			if err := cmd.Run(); err != nil {
				log.Printf("Error playing audio file: %v", err)
			}
		} else {
			// For other formats (MP3, etc.), try using default program
			cmd := exec.Command("cmd", "/C", "start", "/MIN", filePath)
			if err := cmd.Run(); err != nil {
				log.Printf("Error playing audio file: %v", err)
			}
		}
	} else {
		// Unix-like: Use aplay, paplay, or similar
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

