//go:build darwin

package platform

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework ApplicationServices

#include <ApplicationServices/ApplicationServices.h>

// pm_idle_seconds returns the number of seconds since any HID input event was
// last seen system-wide. Uses the HID system state so it reflects real
// hardware activity rather than just this app's event source.
static double pm_idle_seconds(void) {
    return CGEventSourceSecondsSinceLastEventType(kCGEventSourceStateHIDSystemState, kCGAnyInputEventType);
}
*/
import "C"

func IdleSeconds() float64 {
	return float64(C.pm_idle_seconds())
}
