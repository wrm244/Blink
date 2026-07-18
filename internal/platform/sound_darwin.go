//go:build darwin

package platform

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AppKit -framework Foundation

#include <stdlib.h>
#import <AppKit/AppKit.h>
#import <Foundation/Foundation.h>

// pm_play_sound plays one of the built-in macOS system sounds by name (e.g.
// "Glass", "Tink"). Returns 1 if a sound with that name was found, 0 otherwise.
// It plays asynchronously so it never blocks the caller.
static int pm_play_sound(const char *name) {
    @autoreleasepool {
        NSString *n = [NSString stringWithUTF8String:name];
        NSSound *s = [NSSound soundNamed:n];
        if (s == nil) {
            return 0;
        }
        [s play];
        return 1;
    }
}
*/
import "C"

import "unsafe"

// PlaySound plays the named macOS system sound. If the name is unknown to the
// system nothing happens. The call returns immediately; the sound plays in
// the background.
func PlaySound(name string) {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	C.pm_play_sound(cname)
}
