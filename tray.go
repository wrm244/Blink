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
	menuItemTakeBreak    *application.MenuItem
	menuItemSkip         *application.MenuItem
	menuItemPostpone     *application.MenuItem
	menuItemPauseResume  *application.MenuItem
	menuItemStart        *application.MenuItem
	menuItemReset        *application.MenuItem
	menuItemPreferences  *application.MenuItem
	menuItemQuit         *application.MenuItem
	menuLang             = "zh-CN"
)

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
		"quit":        "退出 PocketMind",
	},
	"en": {
		"takeBreak":   "Take a break now",
		"skip":        "Skip break",
		"postpone":    "Postpone break",
		"pauseResume": "Pause / Resume",
		"startFocus":  "Start focusing",
		"reset":       "Reset cycle",
		"preferences": "Preferences…",
		"quit":        "Quit PocketMind",
	},
}

func tr(key string) string {
	if m, ok := trayStrings[menuLang]; ok {
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
	statusItem = menu.Add("PocketMind")
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
	tray.SetLabel("PocketMind")
	tray.SetTooltip("PocketMind")
	tray.SetMenu(menu)
}

// setMenuLanguage rebuilds the tray menu item labels for the given locale.
func setMenuLanguage(lang string) error {
	menuLang = lang
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
		return fmtDuration(rem), "PocketMind · " + trStatus(st, rem)
	case breakengine.PhaseShortBreak:
		return fmtDuration(rem), "PocketMind · " + trStatus(st, rem)
	case breakengine.PhaseLongBreak:
		return fmtDuration(rem), "PocketMind · " + trStatus(st, rem)
	case breakengine.PhasePaused:
		return "⏸", "PocketMind · " + tr("pauseResume")
	case breakengine.PhaseIdle:
		return "⏸", "PocketMind · " + tr("pauseResume")
	default:
		// Engine not running (e.g. before onboarding).
		return "PocketMind", "PocketMind"
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
		"focusing":  "专注中，距下次休息 %s",
		"prebreak":  "%s 后开始休息",
		"shortbreak": "短休息，剩余 %s",
		"longbreak": "长休息，剩余 %s",
	},
	"en": {
		"focusing":  "focusing, next break in %s",
		"prebreak":  "break in %s",
		"shortbreak": "short break, %s remaining",
		"longbreak": "long break, %s remaining",
	},
}

func sfmt(phase string, rem time.Duration) string {
	if m, ok := statusStrings[menuLang]; ok {
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
