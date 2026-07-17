// Package config holds the user-configurable settings for PocketMind and
// persists them as JSON in the per-user application-support directory.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Settings is the full set of user-configurable behaviour.
//
// Durations are split into the unit the UI naturally edits in (minutes for the
// long ones, seconds for the short break) and reconciled to time.Duration by
// the break engine.
type Settings struct {
	// FocusDurationMin is how long a focus period lasts before a break is due.
	FocusDurationMin int `json:"focusDurationMin"`
	// ShortBreakDurationSec is the length of a regular (short) break.
	ShortBreakDurationSec int `json:"shortBreakDurationSec"`
	// LongBreakDurationMin is the length of an occasional longer break.
	LongBreakDurationMin int `json:"longBreakDurationMin"`
	// LongBreakInterval is how many short breaks happen before a long break.
	LongBreakInterval int `json:"longBreakInterval"`
	// PreBreakWarningSec is the heads-up shown before a break starts.
	PreBreakWarningSec int `json:"preBreakWarningSec"`
	// IdleThresholdMin: after this many minutes of inactivity the focus timer
	// is treated as paused and resets when activity resumes.
	IdleThresholdMin int `json:"idleThresholdMin"`
	// EnableLongBreaks toggles the occasional long-break cycle.
	EnableLongBreaks bool `json:"enableLongBreaks"`
	// SoundEnabled toggles the break-end chime and pre-break tick.
	SoundEnabled bool `json:"soundEnabled"`

	// Global keyboard shortcuts (Wails accelerator syntax, e.g. "Cmd+Shift+B").
	ShortcutStartBreak    string `json:"shortcutStartBreak"`
	ShortcutSkipBreak     string `json:"shortcutSkipBreak"`
	ShortcutPostponeBreak string `json:"shortcutPostponeBreak"`
	ShortcutPreferences   string `json:"shortcutPreferences"`
}

// Default returns the built-in defaults, modelled on the 20-20-20 rule: a
// focus period of 20 minutes, a 20-second short break, a 5-minute long break
// every 4 short breaks, and a 10-second heads-up before each break.
func Default() Settings {
	return Settings{
		FocusDurationMin:      20,
		ShortBreakDurationSec: 20,
		LongBreakDurationMin:  5,
		LongBreakInterval:      4,
		PreBreakWarningSec:    10,
		IdleThresholdMin:      5,
		EnableLongBreaks:      true,
		SoundEnabled:          true,
		ShortcutStartBreak:    "Cmd+Shift+B",
		ShortcutSkipBreak:     "Cmd+Shift+S",
		ShortcutPostponeBreak: "Cmd+Shift+P",
		ShortcutPreferences:   "Cmd+Shift+,",
	}
}

// withDefaults returns s with any zero/invalid values replaced by the defaults.
// This keeps older saved files valid as new fields are added.
func (s Settings) withDefaults() Settings {
	d := Default()
	if s.FocusDurationMin < 1 {
		s.FocusDurationMin = d.FocusDurationMin
	}
	if s.ShortBreakDurationSec < 1 {
		s.ShortBreakDurationSec = d.ShortBreakDurationSec
	}
	if s.LongBreakDurationMin < 1 {
		s.LongBreakDurationMin = d.LongBreakDurationMin
	}
	if s.LongBreakInterval < 1 {
		s.LongBreakInterval = d.LongBreakInterval
	}
	if s.PreBreakWarningSec < 0 {
		s.PreBreakWarningSec = d.PreBreakWarningSec
	}
	if s.IdleThresholdMin < 1 {
		s.IdleThresholdMin = d.IdleThresholdMin
	}
	if s.ShortcutStartBreak == "" {
		s.ShortcutStartBreak = d.ShortcutStartBreak
	}
	if s.ShortcutSkipBreak == "" {
		s.ShortcutSkipBreak = d.ShortcutSkipBreak
	}
	if s.ShortcutPostponeBreak == "" {
		s.ShortcutPostponeBreak = d.ShortcutPostponeBreak
	}
	if s.ShortcutPreferences == "" {
		s.ShortcutPreferences = d.ShortcutPreferences
	}
	return s
}

// configPath returns the path to the settings file. It lives under the OS's
// per-user application-support directory (~/Library/Application Support on
// macOS), which the user is not expected to edit by hand.
func configPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "PocketMind", "settings.json"), nil
}

// Load reads the settings from disk, falling back to defaults (and creating
// the file) when none exist yet or the file is unreadable.
func Load() (Settings, error) {
	s := Default()
	path, err := configPath()
	if err != nil {
		return s, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Nothing saved yet: persist the defaults so the file exists.
			_ = Save(s)
			return s, nil
		}
		return s, err
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return Default(), err
	}
	return s.withDefaults(), nil
}

// Save writes the settings to disk, creating the directory if needed.
func Save(s Settings) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
