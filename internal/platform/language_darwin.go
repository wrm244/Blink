//go:build darwin

package platform

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Foundation

#import <Foundation/Foundation.h>
#import <stdlib.h>

// pm_preferred_language 返回系统首选语言标签（如 "zh-Hans-CN"、"en-US"）。
// 返回的是 strdup 的副本，调用方负责 free。
static char *pm_preferred_language(void) {
    NSArray<NSString *> *langs = [NSLocale preferredLanguages];
    if (langs == nil || langs.count == 0) {
        return NULL;
    }
    const char *utf8 = [langs[0] UTF8String];
    if (utf8 == NULL) {
        return NULL;
    }
    return strdup(utf8);
}
*/
import "C"

import (
	"strings"
	"unsafe"
)

// SystemLanguage 返回系统首选语言，映射到应用支持的两种语言之一：
// "zh-CN" 或 "en"。系统语言不是中/英时返回空串，调用方自行决定回退。
func SystemLanguage() string {
	p := C.pm_preferred_language()
	if p == nil {
		return ""
	}
	defer C.free(unsafe.Pointer(p))
	lang := strings.ToLower(C.GoString(p))
	if strings.HasPrefix(lang, "zh") {
		return "zh-CN"
	}
	if strings.HasPrefix(lang, "en") {
		return "en"
	}
	return ""
}
