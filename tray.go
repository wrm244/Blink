package main

import (
	"fmt"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"pocketmind/internal/breakengine"
)

var (
	tray       *application.SystemTray
	statusItem *application.MenuItem
)

// buildTray creates the menu-bar status item and its dropdown menu.
func buildTray() {
	menu := application.NewMenu()

	// A disabled first item that doubles as a live status read-out.
	statusItem = menu.Add("PocketMind")
	statusItem.SetEnabled(false)

	menu.AddSeparator()
	menu.Add("Take a break now").OnClick(func(*application.Context) { engine.StartBreakNow() })
	menu.Add("Skip break").OnClick(func(*application.Context) { engine.SkipBreak() })
	menu.Add("Postpone break").OnClick(func(*application.Context) { engine.PostponeBreak() })
	menu.AddSeparator()
	menu.Add("Pause / Resume").OnClick(func(*application.Context) { togglePause() })
	menu.Add("Reset cycle").OnClick(func(*application.Context) { engine.Reset() })
	menu.AddSeparator()
	menu.Add("Preferences…").OnClick(func(*application.Context) { showPreferences() })
	menu.AddSeparator()
	menu.Add("Quit PocketMind").OnClick(func(*application.Context) { app.Quit() })

	tray = app.SystemTray.New()
	tray.SetLabel("PocketMind")
	tray.SetTooltip("PocketMind")
	tray.SetMenu(menu)
}

func togglePause() {
	if engine.GetState().Paused {
		engine.Resume()
	} else {
		engine.Pause()
	}
}

// trayStatusLoop updates the status item label, tray label and tooltip once a
// second. It reads the engine state via the thread-safe GetState (which takes
// and releases the engine mutex) and only then performs the main-thread window
// calls, so it never holds the mutex across a main-thread wait.
func trayStatusLoop() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for range ticker.C {
		st := engine.GetState()
		label, tooltip := formatStatus(st)
		if tray != nil {
			tray.SetLabel(label)
			tray.SetTooltip(tooltip)
		}
		if statusItem != nil {
			statusItem.SetLabel(tooltip)
		}
	}
}

func formatStatus(st breakengine.State) (label, tooltip string) {
	rem := time.Duration(st.RemainingSec) * time.Second
	switch st.Phase {
	case breakengine.PhaseFocusing, breakengine.PhasePreBreak:
		return fmtDuration(rem), "PocketMind — focusing, next break in " + fmtDuration(rem)
	case breakengine.PhaseShortBreak:
		return fmtDuration(rem), "PocketMind — short break, " + fmtDuration(rem) + " remaining"
	case breakengine.PhaseLongBreak:
		return fmtDuration(rem), "PocketMind — long break, " + fmtDuration(rem) + " remaining"
	case breakengine.PhasePaused:
		return "Paused", "PocketMind — paused"
	case breakengine.PhaseIdle:
		return "Idle", "PocketMind — paused (inactive)"
	default:
		return "PocketMind", "PocketMind"
	}
}

func fmtDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	d = d.Round(time.Second)
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%d:%02d", m, s)
}
