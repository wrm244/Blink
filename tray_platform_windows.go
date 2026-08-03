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
//  2. Windows 只在右键时弹出菜单：Wails 的 applySmartDefaults 仅在托盘
//     绑定了窗口时才安装左键处理器，而 Blink 没有绑定窗口，于是左键点击
//     什么都不会发生。macOS 上左键会落到原生菜单跟踪，行为不一致且不符合
//     Windows 用户预期，所以显式把左键也接到菜单上。
func applyTrayPlatform(t *application.SystemTray, menu *application.Menu) {
	t.SetIcon(trayIcon)
	t.OnClick(func() { t.OpenMenu() })
	// 双击直接打开设置，符合 Windows 托盘应用的惯例。
	// 走 goroutine 是因为点击回调运行在主线程，而窗口的 Show/Focus
	// 内部通过 InvokeSync 再次调度到主线程，直接调用会自锁。
	t.OnDoubleClick(func() { go showPreferences() })
}
