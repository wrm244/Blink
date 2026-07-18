package main

import "github.com/wailsapp/wails/v3/pkg/application"

const prefsWindowName = "pm-prefs"

// preferencesOptions describes the settings window. It uses a solid opaque
// background (not macOS translucency): a translucent window on Tahoe shows a
// bright glass frame at the edges and an overscroll "white" when content is
// scrolled past its end. The web page paints its own themed gradient on top of
// the opaque surface.
func preferencesOptions() application.WebviewWindowOptions {
	return application.WebviewWindowOptions{
		Name:             prefsWindowName,
		Title:            "PocketMind",
		Width:            920,
		Height:           660,
		// Min size matches the default size so the window opens fixed and
		// cannot be shrunk below the designed layout.
		MinWidth:         920,
		MinHeight:        660,
		URL:              "/",
		InitialPosition:  application.WindowCentered,
		BackgroundType:   application.BackgroundTypeSolid,
		BackgroundColour: application.NewRGB(21, 23, 28),
		Mac: application.MacWindow{
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
