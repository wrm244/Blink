//go:build darwin

package platform

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework CoreServices
#cgo LDFLAGS: -framework ServiceManagement

#import <CoreServices/CoreServices.h>
#import <ServiceManagement/ServiceManagement.h>
#import <stdlib.h>

// pm_set_login_item_legacy 使用 LSSharedFileList 把 path 指向的应用加入或
// 移出当前用户的登录项。这是 macOS 12 及更早版本的回退路径（SMAppService
// 需要 macOS 13+）。该 API 在 macOS 10.11 起被标记为 deprecated，但在旧系统
// 上仍是唯一可用的公开登录项接口，且当前部署目标为 macOS 12。这里的 pragma
// 抑制废弃警告（与 activation_darwin.go 处理 activateIgnoringOtherApps: 同理）。
#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wdeprecated-declarations"
static void pm_set_login_item_legacy(const char *path, int enable) {
    CFStringRef cfPath = CFStringCreateWithCString(NULL, path, kCFStringEncodingUTF8);
    if (!cfPath) return;
    CFURLRef url = CFURLCreateWithFileSystemPath(NULL, cfPath, kCFURLPOSIXPathStyle, true);
    CFRelease(cfPath);
    if (!url) return;

    LSSharedFileListRef list = LSSharedFileListCreate(NULL, kLSSharedFileListSessionLoginItems, NULL);
    if (!list) { CFRelease(url); return; }

    if (enable) {
        LSSharedFileListItemRef item = LSSharedFileListInsertItemURL(list, kLSSharedFileListItemLast, NULL, NULL, url, NULL, NULL);
        if (item) CFRelease(item);
    } else {
        CFArrayRef items = LSSharedFileListCopySnapshot(list, NULL);
        if (items) {
            CFIndex count = CFArrayGetCount(items);
            for (CFIndex i = 0; i < count; i++) {
                LSSharedFileListItemRef item = (LSSharedFileListItemRef)CFArrayGetValueAtIndex(items, i);
                CFURLRef itemURL = LSSharedFileListItemCopyResolvedURL(item, 0, NULL);
                if (itemURL) {
                    if (CFEqual(itemURL, url)) {
                        LSSharedFileListItemRemove(list, item);
                    }
                    CFRelease(itemURL);
                }
            }
            CFRelease(items);
        }
    }
    CFRelease(list);
    CFRelease(url);
}
#pragma clang diagnostic pop

// pm_register_login_item_sm 使用 SMAppService 注册当前应用为登录项
//（需要 macOS 13+，且应用以 .app 包形式运行）。返回 0 表示成功。
static int pm_register_login_item_sm(void) {
    if (@available(macOS 13.0, *)) {
        NSError *err = nil;
        BOOL ok = [[SMAppService mainAppService] registerAndReturnError:&err];
        if (!ok && err != nil) {
            NSLog(@"Blink: 注册开机自启失败: %@", err);
        }
        return ok ? 0 : 1;
    }
    return -1; // 系统低于 macOS 13，调用方应走 legacy 路径
}

// pm_unregister_login_item_sm 使用 SMAppService 注销当前应用的登录项。
static int pm_unregister_login_item_sm(void) {
    if (@available(macOS 13.0, *)) {
        NSError *err = nil;
        BOOL ok = [[SMAppService mainAppService] unregisterAndReturnError:&err];
        if (!ok && err != nil) {
            NSLog(@"Blink: 注销开机自启失败: %@", err);
        }
        return ok ? 0 : 1;
    }
    return -1; // 系统低于 macOS 13
}
*/
import "C"

import (
	"os"
	"path/filepath"
	"strings"
	"unsafe"
)

// SetLaunchAtLogin 在 macOS 上把当前应用加入或移出用户的登录项。
//
// macOS 13+ 优先使用 SMAppService（官方推荐、系统托管的登录项机制），
// 它要求应用以 .app 包形式运行（开发模式的裸二进制无法注册，会静默返回
// 但不报错，避免干扰 wails3 dev）。macOS 12 及更早回退到 LSSharedFileList。
func SetLaunchAtLogin(enabled bool) error {
	if enabled {
		if C.pm_register_login_item_sm() == 0 {
			return nil
		}
		// SMAppService 不可用（系统过旧或裸二进制）时回退到 legacy 登录项。
		path, err := appBundlePath()
		if err != nil {
			return err
		}
		cpath := C.CString(path)
		C.pm_set_login_item_legacy(cpath, 1)
		C.free(unsafe.Pointer(cpath))
		return nil
	}

	if C.pm_unregister_login_item_sm() == 0 {
		return nil
	}
	path, err := appBundlePath()
	if err != nil {
		return err
	}
	cpath := C.CString(path)
	C.pm_set_login_item_legacy(cpath, 0)
	C.free(unsafe.Pointer(cpath))
	return nil
}

// appBundlePath 从可执行文件路径向上推导出 .app 包路径。
// 开发模式（wails3 dev 跑裸二进制）找不到 .app 时退回可执行文件本身，
// 这种情况下登录项仍能指向它，但更推荐通过打包后的 .app 运行。
func appBundlePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	dir := exe
	for {
		if strings.HasSuffix(dir, ".app") {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return exe, nil
}
