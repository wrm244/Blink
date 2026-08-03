//go:build darwin

package platform

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AppKit

#import <AppKit/AppKit.h>
#import <dispatch/dispatch.h>

// pm_set_dock_visible toggles the macOS activation policy between Regular
// (Dock icon + Cmd-Tab entry, a normal foreground app) and Accessory (a
// menu-bar-only agent with neither). setActivationPolicy must run on the main
// thread, so the call is dispatched there asynchronously — callers are Wails
// binding/goroutine threads, not the main thread.
static void pm_set_dock_visible(int visible) {
    dispatch_async(dispatch_get_main_queue(), ^{
        [NSApp setActivationPolicy:visible ? NSApplicationActivationPolicyRegular
                                           : NSApplicationActivationPolicyAccessory];
    });
}

// pm_activate brings the app to the foreground so its key window is focused.
// Needed after an Accessory→Regular transition, which does not activate on its
// own.
static void pm_activate(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        [NSApp activateIgnoringOtherApps:YES];
    });
}
*/
import "C"

// SetDockVisible switches the app between a regular foreground app (Dock icon,
// Cmd-Tab) and a menu-bar accessory agent. Used to surface a Dock icon only
// while the settings window is open. The switch is marshalled to the main
// thread and returns immediately.
func SetDockVisible(visible bool) {
	if visible {
		C.pm_set_dock_visible(1)
	} else {
		C.pm_set_dock_visible(0)
	}
}

// Activate moves the app to the foreground. It is a no-op if already active.
func Activate() {
	C.pm_activate()
}
