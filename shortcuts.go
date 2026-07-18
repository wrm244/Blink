package main

import (
	"log"
	"sync"

	"blink/internal/config"
)

// rebindMu serialises shortcut re-registration so a flurry of settings saves
// cannot interleave UnregisterAll/Register.
var rebindMu sync.Mutex

// registerAll binds the four global shortcuts from the settings, replacing any
// previously bound ones. Safe to call from any goroutine: before the app runs
// the bindings are deferred, after it runs they marshal to the main thread
// (which is why callers that already hold the main thread - the SaveSettings
// binding call - dispatch this on its own goroutine).
func registerAll(s config.Settings) {
	rebindMu.Lock()
	defer rebindMu.Unlock()
	if app == nil {
		return
	}
	_ = app.GlobalShortcut.UnregisterAll()

	bind := func(acc string, fn func()) {
		if acc == "" {
			return
		}
		if err := app.GlobalShortcut.Register(acc, fn); err != nil {
			log.Printf("blink: shortcut %q not registered: %v", acc, err)
		}
	}
	bind(s.ShortcutStartBreak, func() { engine.StartBreakNow() })
	bind(s.ShortcutSkipBreak, func() { engine.SkipBreak() })
	bind(s.ShortcutPostponeBreak, func() { engine.PostponeBreak() })
	bind(s.ShortcutPreferences, func() { showPreferences() })
}
