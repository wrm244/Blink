//go:build windows

package main

import (
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"blink/internal/config"
)

// installCloseHandler 接管设置窗口的关闭行为。
//
// 背景：Wails 给每个窗口预置了一个 Common.WindowClosing 监听器，负责销毁窗口。
// 要改变关闭语义就必须赶在它之前否决事件——而监听器（OnWindowEvent）是在
// goroutine 里跑的，且只有钩子（RegisterHook）会被同步执行并检查取消标志
// （见 Wails 的 HandleWindowEvent）。所以这里必须用 RegisterHook，
// 用 OnWindowEvent 是拦不住的。
//
// 线程模型：钩子由 WM_CLOSE 同步触发，运行在主线程上。因此钩子内部
// 绝不能碰任何 InvokeSync 的 API（Show/Hide 都是），否则会自锁。
// 做法是先无条件取消事件，再把后续处理甩给 goroutine。
func installCloseHandler(w *application.WebviewWindow) {
	w.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		// 一律先拦下：窗口是留是关，交给下面的 goroutine 决定。
		e.Cancel()
		go handleCloseRequest()
	})
}

// handleCloseRequest 在 goroutine 上决定关闭窗口后的去向。
func handleCloseRequest() {
	switch engine.GetSettings().CloseAction {
	case config.CloseActionBackground:
		hidePrefsWindow()
	case config.CloseActionQuit:
		quitApp()
	default:
		askFrontendToConfirm()
	}
}

// askFrontendToConfirm 请前端弹出关闭确认框。
//
// 为什么不用系统对话框：Windows 上 Wails 的 MessageDialog 最终走 MessageBox，
// QuestionDialogType 固定渲染 MB_YESNO，按钮文字由系统语言决定、应用无法自定义；
// 更麻烦的是 Wails 靠把返回值映射成固定英文字符串（"Yes"/"No"）去匹配
// Button.Label 才能触发回调，按钮标签一旦本地化，回调就永远不会执行。
// 而且 MessageBox 根本没有"记住我的选择"这种复选框。
// 改由前端渲染后，文案、勾选框、视觉风格全部可控，也天然复用现有 i18n。
//
// 用户的选择通过 BreakService.ResolveClose 回到 Go 端。
func askFrontendToConfirm() {
	if app == nil {
		return
	}
	// 去重 + 挂兜底定时器；已有询问在等待回应时直接放弃。
	if !beginCloseAsk() {
		return
	}
	app.Event.Emit(eventCloseRequest)
}
