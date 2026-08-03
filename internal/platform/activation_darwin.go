//go:build darwin

package platform

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AppKit

#import <AppKit/AppKit.h>
#import <dispatch/dispatch.h>

// pm_set_dock_visible 在 Regular（Dock 图标 + Cmd-Tab 条目，普通前台应用）
// 和 Accessory（无 Dock 图标和 Cmd-Tab 的菜单栏代理）之间切换激活策略。
// setActivationPolicy 必须在主线程执行，因此调用被异步分发到主线程——
// 调用者是 Wails 绑定/goroutine 线程，而非主线程。
static void pm_set_dock_visible(int visible) {
    dispatch_async(dispatch_get_main_queue(), ^{
        [NSApp setActivationPolicy:visible ? NSApplicationActivationPolicyRegular
                                           : NSApplicationActivationPolicyAccessory];
    });
}

// pm_activate 将应用带到前台，使其关键窗口获得焦点。
// 在 Accessory -> Regular 切换后需要调用，因为切换本身不会自动激活。
static void pm_activate(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        [NSApp activateIgnoringOtherApps:YES];
    });
}
*/
import "C"

// SetDockVisible 在普通前台应用（Dock 图标 + Cmd-Tab）和菜单栏代理之间切换。
// 仅在设置窗口打开时显示 Dock 图标。切换被分发到主线程并立即返回。
func SetDockVisible(visible bool) {
	if visible {
		C.pm_set_dock_visible(1)
	} else {
		C.pm_set_dock_visible(0)
	}
}

// Activate 将应用移到前台。如果已在前台则为空操作。
func Activate() {
	C.pm_activate()
}
