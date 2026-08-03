//go:build darwin

package platform

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AppKit -framework Foundation

#include <stdlib.h>
#import <AppKit/AppKit.h>
#import <Foundation/Foundation.h>

// pm_play_sound 按名称播放内置 macOS 系统音效（如 "Glass"、"Tink"）。
// 找到对应名称的音效返回 1，否则返回 0。异步播放，永不阻塞调用者。
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

// PlaySound 播放指定名称的 macOS 系统音效。
// 如果名称未知则不做任何操作。调用立即返回；音效在后台播放。
func PlaySound(name string) {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	C.pm_play_sound(cname)
}
