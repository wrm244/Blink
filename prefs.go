package main

import "github.com/wailsapp/wails/v3/pkg/application"

const prefsWindowName = "pm-prefs"

// preferencesOptions describes the settings window: a translucent, dark,
// frameless-inset window centered on the primary display. The window itself
// carries the macOS vibrancy; the web page paints an opaque dark surface on
// top so text stays readable over any wallpaper.
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
		BackgroundColour: application.NewRGBA(14, 16, 28, 255),
		Mac: application.MacWindow{
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
			InvisibleTitleBarHeight: 50,
		},
		CloseButtonState: application.ButtonEnabled,
	}
}

// showPreferences reveals the settings window, creating it on first use. The
// window is created on demand (rather than pre-created hidden at startup) so
// that Wails' macOS backend shows it immediately - a Hidden:true window never
// receives the key event that its own lazy-show handler waits for, which would
// leave it invisible.
func showPreferences() {
	if app == nil {
		return
	}
	w, ok := app.Window.GetByName(prefsWindowName)
	if !ok || w == nil {
		w = app.Window.NewWithOptions(preferencesOptions())
	}
	w.Show()
	w.Focus()
}
