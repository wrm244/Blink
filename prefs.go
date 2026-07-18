package main

import "github.com/wailsapp/wails/v3/pkg/application"

const prefsWindowName = "pm-prefs"

// preferencesOptions describes the settings window: a translucent,
// frameless-inset window centered on the primary display. The window carries
// the macOS vibrancy (which follows the system appearance); the web page paints
// a semi-opaque themed surface on top so text stays readable while keeping a
// hint of native vibrancy.
func preferencesOptions() application.WebviewWindowOptions {
	return application.WebviewWindowOptions{
		Name:             prefsWindowName,
		Title:            "PocketMind",
		Width:            520,
		Height:           680,
		MinWidth:         460,
		MinHeight:        560,
		URL:              "/",
		InitialPosition:  application.WindowCentered,
		BackgroundType:   application.BackgroundTypeTranslucent,
		BackgroundColour: application.NewRGBA(0, 0, 0, 0),
		Mac: application.MacWindow{
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
			InvisibleTitleBarHeight: 50,
		},
		CloseButtonState: application.ButtonEnabled,
	}
}

// showPreferences reveals the settings window, creating it on first use.
//
// Thread-safety: this may be called from the main thread (the ServiceStartup
// hook on first run) or from a goroutine (tray menu / global shortcut). The
// fresh-creation path deliberately does NOT call Show()/Focus(): NewWithOptions
// already shows the window (Hidden is unset), and Show() does a re-entrant
// InvokeSync that deadlocks the main thread (InvokeSync posts to the main
// queue via dispatch_async then blocks on a WaitGroup; if the caller IS the
// main thread, the posted block can never run). The existing-window branch
// (Show/Focus) is only reached from goroutine callers, so it is safe.
func showPreferences() {
	if app == nil {
		return
	}
	if w, ok := app.Window.GetByName(prefsWindowName); ok && w != nil {
		w.Show()
		w.Focus()
		return
	}
	// Fresh creation: NewWithOptions shows the window itself (Hidden is false),
	// whether it runs inline (app already running) or deferred (app still
	// starting). Do not call Show()/Focus() here.
	app.Window.NewWithOptions(preferencesOptions())
}
