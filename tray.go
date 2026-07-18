package main

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"blink/internal/breakengine"
)

var (
	tray                *application.SystemTray
	statusItem          *application.MenuItem
	menuItemTakeBreak   *application.MenuItem
	menuItemSkip        *application.MenuItem
	menuItemPostpone    *application.MenuItem
	menuItemPauseResume *application.MenuItem
	menuItemStart       *application.MenuItem
	menuItemReset       *application.MenuItem
	menuItemPreferences *application.MenuItem
	menuItemQuit        *application.MenuItem
	// menuLang holds the active tray-menu locale. It is read by trayStatusLoop
	// (and the menu click handlers, which run on the main thread) and written
	// by setMenuLanguage, which SaveSettings dispatches on a goroutine. We use
	// atomic.Value rather than a bare string so the race detector stays quiet
	// and the read in trayStatusLoop never sees a half-written value.
	menuLang atomic.Value
)

func init() {
	menuLang.Store("zh-CN")
}

// menuLangString loads the current menu locale as a plain string. Used by the
// tray helpers below so they can stay simple.
func menuLangString() string {
	return menuLang.Load().(string)
}

// Tray menu strings per locale. Kept in Go because the menu is built natively
// on the macOS side, not in the webview.
var trayStrings = map[string]map[string]string{
	"zh-CN": {
		"takeBreak":   "立即开始休息",
		"skip":        "跳过休息",
		"postpone":    "推迟休息",
		"pauseResume": "暂停 / 继续",
		"startFocus":  "开始专注",
		"reset":       "重置周期",
		"preferences": "设置…",
		"quit":        "退出 Blink",
	},
	"en": {
		"takeBreak":   "Take a break now",
		"skip":        "Skip break",
		"postpone":    "Postpone break",
		"pauseResume": "Pause / Resume",
		"startFocus":  "Start focusing",
		"reset":       "Reset cycle",
		"preferences": "Preferences…",
		"quit":        "Quit Blink",
	},
}

func tr(key string) string {
	if m, ok := trayStrings[menuLangString()]; ok {
		if v, ok := m[key]; ok {
			return v
		}
	}
	return trayStrings["en"][key]
}

// buildTray creates the menu-bar status item and its dropdown menu.
func buildTray() {
	menu := application.NewMenu()

	// A disabled first item that doubles as a live status read-out.
	statusItem = menu.Add("Blink")
	statusItem.SetEnabled(false)

	menu.AddSeparator()
	menuItemTakeBreak = menu.Add(tr("takeBreak"))
	menuItemTakeBreak.OnClick(func(*application.Context) { engine.StartBreakNow() })
	menuItemSkip = menu.Add(tr("skip"))
	menuItemSkip.OnClick(func(*application.Context) { engine.SkipBreak() })
	menuItemPostpone = menu.Add(tr("postpone"))
	menuItemPostpone.OnClick(func(*application.Context) { engine.PostponeBreak() })
	menu.AddSeparator()
	menuItemPauseResume = menu.Add(tr("pauseResume"))
	menuItemPauseResume.OnClick(func(*application.Context) { togglePause() })
	menuItemStart = menu.Add(tr("startFocus"))
	menuItemStart.OnClick(func(*application.Context) { engine.Start() })
	menuItemReset = menu.Add(tr("reset"))
	menuItemReset.OnClick(func(*application.Context) { engine.Reset() })
	menu.AddSeparator()
	menuItemPreferences = menu.Add(tr("preferences"))
	menuItemPreferences.OnClick(func(*application.Context) { showPreferences() })
	menu.AddSeparator()
	menuItemQuit = menu.Add(tr("quit"))
	menuItemQuit.OnClick(func(*application.Context) { app.Quit() })

	tray = app.SystemTray.New()
	tray.SetLabel("Blink")
	tray.SetTooltip("Blink")
	tray.SetMenu(menu)
}

// setMenuLanguage rebuilds the tray menu item labels for the given locale.
func setMenuLanguage(lang string) error {
	menuLang.Store(lang)
	if menuItemTakeBreak != nil {
		menuItemTakeBreak.SetLabel(tr("takeBreak"))
		menuItemSkip.SetLabel(tr("skip"))
		menuItemPostpone.SetLabel(tr("postpone"))
		menuItemPauseResume.SetLabel(tr("pauseResume"))
		menuItemStart.SetLabel(tr("startFocus"))
		menuItemReset.SetLabel(tr("reset"))
		menuItemPreferences.SetLabel(tr("preferences"))
		menuItemQuit.SetLabel(tr("quit"))
	}
	return nil
}

func togglePause() {
	if engine.GetState().Paused {
		engine.Resume()
	} else {
		engine.Pause()
	}
}

// trayStatusLoop updates the status item label, tray label and tooltip when the
// rendered text changes. The engine emits a tick event every second while
// counting down, but the formatted label only changes when the remaining time
// crosses a whole second - which is every second during focus/break, but never
// during pause/idle. By caching the last strings we skip the main-thread
// SetLabel/SetTooltip calls (each of which triggers a native redraw) when the
// text is identical, eliminating pointless UI work during the paused and idle
// phases. The 2s cadence is enough for a status read-out: the per-second tick
// event from the engine still drives the webview UI at full resolution.
func trayStatusLoop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	var lastLabel, lastTooltip string
	for range ticker.C {
		st := engine.GetState()
		label, tooltip := formatStatus(st)
		if label == lastLabel && tooltip == lastTooltip {
			continue
		}
		lastLabel, lastTooltip = label, tooltip
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
	case breakengine.PhaseFocusing, breakengine.PhasePreBreak,
		breakengine.PhaseShortBreak, breakengine.PhaseLongBreak:
		return fmtDuration(rem), "Blink · " + trStatus(st, rem)
	case breakengine.PhaseIdle:
		return "⏸", "Blink · " + tr("pauseResume")
	default:
		// Engine not running (e.g. before onboarding).
		return "Blink", "Blink"
	}
}

// trStatus localises the per-phase tooltip text.
func trStatus(st breakengine.State, rem time.Duration) string {
	switch st.Phase {
	case breakengine.PhaseFocusing:
		return sfmt("focusing", rem)
	case breakengine.PhasePreBreak:
		return sfmt("prebreak", rem)
	case breakengine.PhaseShortBreak:
		return sfmt("shortbreak", rem)
	case breakengine.PhaseLongBreak:
		return sfmt("longbreak", rem)
	default:
		return ""
	}
}

// sfmt returns a localised "phase · remaining" string. Because the tray runs
// outside the webview i18n, the strings live here.
var statusStrings = map[string]map[string]string{
	"zh-CN": {
		"focusing":   "专注中，距下次休息 %s",
		"prebreak":   "%s 后开始休息",
		"shortbreak": "短休息，剩余 %s",
		"longbreak":  "长休息，剩余 %s",
	},
	"en": {
		"focusing":   "focusing, next break in %s",
		"prebreak":   "break in %s",
		"shortbreak": "short break, %s remaining",
		"longbreak":  "long break, %s remaining",
	},
}

func sfmt(phase string, rem time.Duration) string {
	if m, ok := statusStrings[menuLangString()]; ok {
		if tpl, ok := m[phase]; ok {
			return fmt.Sprintf(tpl, fmtDuration(rem))
		}
	}
	if tpl, ok := statusStrings["en"][phase]; ok {
		return fmt.Sprintf(tpl, fmtDuration(rem))
	}
	return fmtDuration(rem)
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
