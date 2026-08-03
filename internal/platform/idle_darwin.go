//go:build darwin

package platform

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework ApplicationServices

#include <ApplicationServices/ApplicationServices.h>

// pm_idle_seconds 返回系统范围内最后一次 HID 输入事件至今的秒数。
// 使用 HID 系统状态，因此反映的是真实硬件活动而非仅本应用的事件源。
static double pm_idle_seconds(void) {
    return CGEventSourceSecondsSinceLastEventType(kCGEventSourceStateHIDSystemState, kCGAnyInputEventType);
}
*/
import "C"

// IdleSeconds 返回用户最后一次操作键盘/鼠标以来的秒数。
func IdleSeconds() float64 {
	return float64(C.pm_idle_seconds())
}
