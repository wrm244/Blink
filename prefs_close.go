package main

import (
	"sync/atomic"
	"time"
)

// 关闭设置窗口的确认流程（跨平台部分）。
//
// 目前只有 Windows 会真正走这套流程（见 prefs_close_windows.go 里的
// installCloseHandler；其它平台是空操作）。但状态机和动作执行放在这个
// 无构建约束的文件里，因为 breakservice.go 是跨平台编译的，它上面的
// ResolveClose 绑定方法需要调用这些函数。

// eventCloseRequest 通知前端弹出关闭确认框。
// 前端在设置视图里监听，用户做出选择后回调 BreakService.ResolveClose。
const eventCloseRequest = "blink:close-request"

// 前端可回传的动作。background / quit 复用 config 包的常量值，
// 这样"记住我的选择"可以直接把动作字符串存进 Settings.CloseAction。
const closeActionCancel = "cancel"

// closeAskTimeout 是等待前端"接管"回执的上限。
//
// 注意它计的不是用户的思考时间，而只是前端接住事件、把弹窗显示出来
// 所需的时间——前端一收到事件就会立刻回执（见 BreakService.AckClose），
// 正常是毫秒级。之所以要这道兜底：关闭事件已经被我们取消掉了，万一
// 前端脚本异常或监听器尚未挂载，就没有任何人来接手，关闭按钮从此
// 点了没反应，那种"卡住"的观感比任何默认行为都糟。
//
// 一旦收到回执就停表，之后用户对着弹窗想多久都不会被打断。
const closeAskTimeout = 4 * time.Second

var (
	// closeAskPending 表示当前有一次待回应的关闭询问。
	// 兼作去重：用户连点关闭按钮时只发一次事件、只挂一个兜底定时器。
	closeAskPending atomic.Bool
	// closeAskTimer 持有兜底定时器，前端及时回应时需要把它停掉。
	closeAskTimer atomic.Pointer[time.Timer]
)

// beginCloseAsk 尝试开启一次关闭询问。
// 返回 false 表示已有一次询问在等待回应，调用方应直接放弃。
func beginCloseAsk() bool {
	if !closeAskPending.CompareAndSwap(false, true) {
		return false
	}
	t := time.AfterFunc(closeAskTimeout, func() {
		// 兜底：前端没在时限内回应，按最安全的方式收场——隐藏到托盘。
		// 隐藏不会中断计时，用户随时能从通知区域把窗口叫回来；
		// 相比之下自动退出会悄悄停掉休息提醒，风险大得多。
		if closeAskPending.CompareAndSwap(true, false) {
			closeAskTimer.Store(nil)
			hidePrefsWindow()
		}
	})
	closeAskTimer.Store(t)
	return true
}

// ackCloseAsk 由前端在收到询问事件、弹窗已显示后立即调用。
//
// 只停兜底定时器，不改 pending 状态：询问仍在进行中，等的是用户的决定。
// 分成"接管"和"决定"两步，是为了让超时只用来探测前端是否活着——
// 否则无从区分"前端没接住"和"用户在对着弹窗思考"，后者被兜底逻辑
// 抢先处理的话，用户会看到窗口在眼皮底下自己消失。
func ackCloseAsk() {
	if t := closeAskTimer.Swap(nil); t != nil {
		t.Stop()
	}
}

// finishCloseAsk 认领当前这次关闭询问，成功则停掉兜底定时器。
// 返回 false 表示这次回应来晚了（兜底已经处理过）或本就没有待回应的询问，
// 此时调用方不应再执行任何动作，否则会和兜底逻辑重复执行。
func finishCloseAsk() bool {
	if !closeAskPending.CompareAndSwap(true, false) {
		return false
	}
	if t := closeAskTimer.Swap(nil); t != nil {
		t.Stop()
	}
	return true
}

// hidePrefsWindow 隐藏设置窗口，让应用继续在通知区域运行。
//
// 隐藏而非销毁：设置窗口是单例（prefsWindowName），保活可以让下次打开
// 免去重新加载 webview 的开销，也保住前端已加载的状态。这与休息遮罩的
// 处理相反——遮罩是全屏 webview，常驻内存代价高，所以那边选择销毁重建。
//
// 线程：Hide 内部是 InvokeSync，只能从非主线程调用。
func hidePrefsWindow() {
	if app == nil {
		return
	}
	if w, ok := app.Window.GetByName(prefsWindowName); ok && w != nil {
		w.Hide()
	}
}

// quitApp 停止计时并退出应用。
func quitApp() {
	if engine != nil {
		engine.Stop()
	}
	if app != nil {
		app.Quit()
	}
}
