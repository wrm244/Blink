//go:build windows

package main

import (
	_ "embed"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// trayIcon 是托盘通知区图标。macOS 用文字标签显示倒计时，Windows 通知区
// 只能显示图标，因此这里必须显式提供一个，否则 Wails 会回退到自带的
// Wails 默认图标。
//
//go:embed build/windows/icon.ico
var trayIcon []byte

// applyTrayPlatform 补齐托盘在 Windows 上与 macOS 不同的部分。
//
// 两点平台差异：
//
//  1. SystemTray.SetLabel 在 Windows 上是空实现（见 Wails 的
//     systemtray_windows.go）。通知区不支持文字，倒计时只能通过工具提示
//     和菜单首项呈现——trayStatusLoop 已经在更新这两处，这里无需额外处理。
//
//  2. Wails 在 Windows 上不会为托盘自动安装任何点击处理器：左键/右键/双击
//     分别由 WM_LBUTTONUP / WM_RBUTTONUP / WM_LBUTTONDBLCLK 分发到对应的
//     handler，菜单也不会自动弹出，必须由 handler 显式调用 OpenMenu()。
//     按 Windows 托盘惯例配置如下交互：
//       · 右键 → 弹出菜单（OnRightClick）
//       · 双击左键 → 打开设置页（OnDoubleClick）
//       · 左键单击 → 不打开菜单（既不与 macOS 的左键跟踪冲突，也不会让
//         用户误触弹菜单）
func applyTrayPlatform(t *application.SystemTray, menu *application.Menu) {
	t.SetIcon(trayIcon)
	// 右键弹出菜单，符合 Windows 托盘惯例。
	t.OnRightClick(func() { t.OpenMenu() })
	// 双击直接打开设置，符合 Windows 托盘应用的惯例。
	// 走 goroutine 是因为点击回调运行在主线程，而窗口的 Show/Focus
	// 内部通过 InvokeSync 再次调度到主线程，直接调用会自锁。
	t.OnDoubleClick(func() { go showPreferences() })
}
