# Plan: Grace Window for In-Progress Saves

## Problem

A race condition exists between the alarm timer and the save-detection pipeline:

1. User quicksaves at **T=4:57** (5 seconds before the alarm would fire)
2. `handleQuicksaveChange` is called → a 3-second debounce timer starts
3. At **T=5:00**, the `alarmTimer` fires → `triggerAlarm()` plays the sound
4. At **T=5:00** (3 seconds after 4:57), the debounce fires → `processQuicksave()` resets the alarm timers

The alarm plays unnecessarily because the file-change event was already detected and was being processed, but the debounce window hadn't completed yet to reset the timers.

---

## Root Cause

The `triggerAlarm()` function has no awareness of whether a save is currently in-flight (i.e., detected but not yet processed through the debounce pipeline).

---

## Proposed Fix: `pendingSave` Flag

Add a single boolean field `pendingSave` to `SaveReminder` that tracks whether a file change has been detected but not yet fully processed.

### Changes to `main.go`

**1. Add field to `SaveReminder` struct** (~line 64)

```go
lastChangeDetectedAt time.Time // set when a file-change event is detected; zero when not pending
```

**2. Set timestamp in `handleQuicksaveChange`** (~line 483)

When a file-change event arrives and the debounce timer starts:

```go
sr.lastChangeDetectedAt = time.Now()
```

**3. Clear timestamp in `processQuicksave`** (~line 503)

At the top of `processQuicksave`, before backup or timer work:

```go
sr.lastChangeDetectedAt = time.Time{} // zero value = no longer pending
```

**4. Check timestamp in `triggerAlarm`** (~line 647)

Suppress the alarm only if a change was detected recently (within `debounceDelay + 10s`):

```go
func (sr *SaveReminder) triggerAlarm() {
    debounceDelay, _ := time.ParseDuration(sr.config.DebounceDelay)
    graceWindow := debounceDelay + 10*time.Second
    if !sr.lastChangeDetectedAt.IsZero() && time.Since(sr.lastChangeDetectedAt) < graceWindow {
        log.Printf("Alarm suppressed: save detected %v ago, still within grace window", time.Since(sr.lastChangeDetectedAt))
        return
    }
    log.Printf("*** ALARM: Time to save! It's been %v since last save. ***", time.Since(sr.lastSaveTime))
    sr.playAlarmSound()
}
```

---

## Why This Approach

- **Minimal change**: touches only 3–4 lines across existing functions, no new config options or timers needed.
- **Semantically correct**: the timestamp directly captures when we last saw activity, which is exactly what we want to check.
- **Bounded suppression**: the grace window is `debounceDelay + 10s`. Even if file changes arrive in a continuous loop (for unknown reasons), once the *last* detected change is older than the grace window, the alarm fires normally. A plain `bool` would suppress alarms forever in that scenario.
- **Self-expiring**: no need for a separate cleanup timer — the timestamp ages naturally and the check in `triggerAlarm` handles expiry.

---

## Considered Alternatives

**Time-based grace window** (e.g., new `grace_window` config field): Check `time.Since(lastChangeTime) < graceWindow` in `triggerAlarm`. Rejected because it requires a new config option and a hardcoded/configured duration that may not match the actual debounce in practice. The `pendingSave` flag is more precise.

**Plain `pendingSave bool`**: Simple but stale-prone. If file events arrive continuously (for any reason), the debounce timer keeps getting pushed back, `processQuicksave` never fires, and the flag stays `true` forever — permanently suppressing all alarms. The timestamp approach avoids this by bounding suppression to a fixed window.

**Check `debounceTimer != nil`**: The `debounceTimer` field is never set back to nil after it fires, so it's not a reliable indicator of whether a save is pending. Would require additional nil-setting logic.

---

## Files Changed

- `main.go`: Four small, targeted changes as described above.
