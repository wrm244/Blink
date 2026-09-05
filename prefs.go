package main

import (
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"blink/internal/platform"
)

const prefsWindowName = "pm-prefs"

// preferencesOptions 描述设置窗口。使用不透明实色背景（非 macOS 半透明）：
// 在 Tahoe 上半透明窗口会在边缘显示明亮的玻璃边框，滚动到底时出现白色过滚动。
// 网页在 opaque 表面上绘制自己的主题渐变。
// Hidden: true - 窗口创建时隐藏，等前端数据加载完成后再显示，避免白屏闪烁。
func preferencesOptions() application.WebviewWindowOptions {
	return application.WebviewWindowOptions{
		Name:             prefsWindowName,
		Title:            "Blink",
		Width:            920,
		Height:           680,
		MinWidth:         920,
		MinHeight:        680,
		URL:              "/",
		InitialPosition:  application.WindowCentered,
		Hidden:           true,
		BackgroundType:   application.BackgroundTypeSolid,
		BackgroundColour: application.NewRGB(21, 23, 28),
		Mac: application.MacWindow{
			TitleBar:                application.MacTitleBarHiddenInset,
			InvisibleTitleBarHeight: 50,
		},
		CloseButtonState: application.ButtonEnabled,
	}
}

// showPreferences 显示设置窗口，首次调用时创建。
//
// 线程安全：可能从主线程（首次运行的 ServiceStartup 钩子）或
// goroutine（托盘菜单 / 全局快捷键）调用。全新创建路径不调用
// Show()/Focus()：NewWithOptions 已自行显示窗口（Hidden 未设置），
// 而 Show() 执行可重入的 InvokeSync 会死锁主线程（InvokeSync 通过
// dispatch_async 投递到主队列然后阻塞在 WaitGroup 上；如果调用者
// 就是主线程，投递的 block 永远无法执行）。已有窗口分支（Show/Focus）
// 仅从 goroutine 调用者到达，因此是安全的。
func showPreferences() {
	if app == nil {
		return
	}
	if w, ok := app.Window.GetByName(prefsWindowName); ok && w != nil {
		w.Show()
		w.Focus()
		// Windows：Focus() 只调 SetForegroundWindow，后台进程（托盘常驻）
		// 会因前台锁定机制静默失败——窗口可见但藏在别的窗口后面。用
		// ForceForeground（AttachThreadInput 绕过前台锁）补一刀；macOS/
		// 其它平台为空操作。LockOSThread 保证 Attach/Detach 落在同一 OS
		// 线程上（goroutine 随时可能被调度到别的线程，否则 detach 会失败）。
		if hwnd := w.NativeWindow(); hwnd != nil {
			runtime.LockOSThread()
			platform.ForceForeground(uintptr(hwnd))
			runtime.UnlockOSThread()
		}
		platform.SetDockVisible(true)
		platform.Activate()
		return
	}
	// 全新创建：NewWithOptions 自行显示窗口（Hidden 为 false），
	// 无论内联执行（app 已运行）还是延迟执行（app 仍在启动）都一样。
	// 不要在此调用 Show()/Focus()。
	w := app.Window.NewWithOptions(preferencesOptions())
	// 设置窗口打开时应用是普通前台应用（Dock 图标 + Cmd-Tab）。
	// 窗口关闭时回到菜单栏代理。默认关闭处理器销毁窗口，
	// 因此每次重新打开都是全新创建并重新注册此监听器--无泄漏。
	w.OnWindowEvent(events.Common.WindowClosing, func(*application.WindowEvent) {
		platform.SetDockVisible(false)
	})
	// Windows：接管关闭按钮，改为询问是否保留在后台运行（详见
	// prefs_close_windows.go）。其它平台是空操作。
	installCloseHandler(w)
	platform.SetDockVisible(true)
	platform.Activate()
}
