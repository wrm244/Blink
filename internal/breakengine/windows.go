package breakengine

import (
	"github.com/wailsapp/wails/v3/pkg/application"
)

// windowCmd is a message sent (without blocking) from the state-machine goroutine
// to the window-control goroutine. The state machine never touches AppKit
// directly: window Show/Hide/Close and NewWithOptions all marshal to the main
// thread internally, so performing them while holding the engine mutex would
// risk deadlocking against a binding call that runs on the main thread. By
// funnelling every window mutation through this single goroutine we keep the
// mutex free of main-thread waits.
type windowCmd int

const (
	cmdShowOverlays windowCmd = iota
	cmdHideOverlays
	cmdShowNotice
	cmdHideNotice
)

// sendCmd enqueues a window command without blocking. Called under the engine
// mutex; dropping a command is harmless because later commands supersede
// earlier ones (e.g. a hide that follows a show wins).
func (e *Engine) sendCmd(c windowCmd) {
	select {
	case e.cmdCh <- c:
	default:
	}
}

func (e *Engine) showOverlays() { e.sendCmd(cmdShowOverlays) }
func (e *Engine) hideOverlays() { e.sendCmd(cmdHideOverlays) }
func (e *Engine) showNotice()   { e.sendCmd(cmdShowNotice) }
func (e *Engine) hideNotice()   { e.sendCmd(cmdHideNotice) }

// windowLoop owns the overlay and notice windows. It is the only goroutine that
// reads or writes e.overlays and e.notice, so they need no mutex.
func (e *Engine) windowLoop() {
	for {
		select {
		case <-e.stopCh:
			return
		case c := <-e.cmdCh:
			e.execWindowCmd(c)
		}
	}
}

func (e *Engine) execWindowCmd(c windowCmd) {
	switch c {
	case cmdShowOverlays:
		e.ensureOverlays()
		for _, w := range e.overlays {
			w.Show()
		}
	case cmdHideOverlays:
		for _, w := range e.overlays {
			w.Hide()
		}
	case cmdShowNotice:
		e.ensureNotice()
		if e.notice != nil {
			e.notice.Show()
		}
	case cmdHideNotice:
		if e.notice != nil {
			e.notice.Hide()
		}
	}
}

func overlayName(screenID string) string { return "pm-overlay-" + screenID }

// overlaysMatch reports whether the cached overlay windows still correspond to
// the currently attached displays (same count and screen IDs).
func (e *Engine) overlaysMatch(screens []*application.Screen) bool {
	if len(e.overlays) != len(screens) {
		return false
	}
	have := make(map[string]bool, len(e.overlays))
	for _, w := range e.overlays {
		have[w.Name()] = true
	}
	for _, s := range screens {
		if !have[overlayName(s.ID)] {
			return false
		}
	}
	return true
}

func (e *Engine) closeOverlays() {
	for _, w := range e.overlays {
		w.Close()
	}
	e.overlays = nil
}

// ensureOverlays creates one full-screen translucent window per attached
// display (so multi-monitor setups get the blur on every screen), reusing the
// cached windows when the display set hasn't changed.
func (e *Engine) ensureOverlays() {
	a := e.app_()
	if a == nil {
		return
	}
	screens := a.Screen.GetAll()
	if e.overlaysMatch(screens) {
		return
	}
	e.closeOverlays()
	for _, sc := range screens {
		screen := sc
		e.overlays = append(e.overlays, a.Window.NewWithOptions(overlayOptions(screen)))
	}
}

func overlayOptions(s *application.Screen) application.WebviewWindowOptions {
	return application.WebviewWindowOptions{
		Name:           overlayName(s.ID),
		Title:          "",
		Frameless:      true,
		AlwaysOnTop:    true,
		DisableResize:  true,
		Hidden:         true,
		URL:            "/#break",
		Width:          s.Bounds.Width,
		Height:         s.Bounds.Height,
		X:              s.Bounds.X,
		Y:              s.Bounds.Y,
		InitialPosition: application.WindowXY,
		// Solid dark background: macOS Tahoe's translucent vibrancy renders a
		// bright glassy frame at the window edges, so the break overlay uses an
		// opaque solid surface instead - no frame, and a calmer rest screen.
		BackgroundType:   application.BackgroundTypeSolid,
		BackgroundColour:  application.NewRGB(10, 12, 18),
		Mac: application.MacWindow{
			WindowLevel:        application.MacWindowLevelStatus,
			CollectionBehavior: application.MacWindowCollectionBehaviorCanJoinAllSpaces | application.MacWindowCollectionBehaviorStationary,
			TitleBar:            application.MacTitleBar{Hide: true, AppearsTransparent: true, FullSizeContent: true},
			DisableShadow:       true,
		},
		CloseButtonState:      application.ButtonHidden,
		MinimiseButtonState:   application.ButtonHidden,
		MaximiseButtonState:   application.ButtonHidden,
		FullscreenButtonState: application.ButtonHidden,
	}
}

// ensureNotice lazily creates the single pre-break heads-up window, placed at
// the top centre of the primary display.
func (e *Engine) ensureNotice() {
	if e.notice != nil {
		return
	}
	a := e.app_()
	if a == nil {
		return
	}
	e.notice = a.Window.NewWithOptions(noticeOptions(a))
}

func noticeOptions(a *application.App) application.WebviewWindowOptions {
	const (
		noticeWidth  = 380
		noticeHeight = 108
	)
	x, y := 0, 80
	if primary := a.Screen.GetPrimary(); primary != nil {
		x = primary.WorkArea.X + (primary.WorkArea.Width-noticeWidth)/2
		y = primary.WorkArea.Y + 80
	}
	return application.WebviewWindowOptions{
		Name:            "pm-notice",
		Title:           "",
		Frameless:       true,
		AlwaysOnTop:     true,
		DisableResize:   true,
		Hidden:          true,
		URL:             "/#notice",
		Width:           noticeWidth,
		Height:          noticeHeight,
		X:               x,
		Y:               y,
		InitialPosition: application.WindowXY,
		BackgroundType:   application.BackgroundTypeTransparent,
		BackgroundColour:  application.NewRGBA(0, 0, 0, 0),
		Mac: application.MacWindow{
			Backdrop:    application.MacBackdropTranslucent,
			WindowLevel: application.MacWindowLevelFloating,
			TitleBar:    application.MacTitleBar{Hide: true, AppearsTransparent: true, FullSizeContent: true},
		},
		CloseButtonState:      application.ButtonHidden,
		MinimiseButtonState:   application.ButtonHidden,
		MaximiseButtonState:   application.ButtonHidden,
		FullscreenButtonState: application.ButtonHidden,
	}
}
