//go:build windows

package platform

// modUser32 / modKernel32 在 idle_windows.go 中声明，同为 windows 构建标签。
var (
	procGetForegroundWindow      = modUser32.NewProc("GetForegroundWindow")
	procSetForegroundWindow      = modUser32.NewProc("SetForegroundWindow")
	procBringWindowToTop         = modUser32.NewProc("BringWindowToTop")
	procSetFocus                 = modUser32.NewProc("SetFocus")
	procAttachThreadInput        = modUser32.NewProc("AttachThreadInput")
	procGetWindowThreadProcessId = modUser32.NewProc("GetWindowThreadProcessId")
	procShowWindow               = modUser32.NewProc("ShowWindow")
	procIsIconic                 = modUser32.NewProc("IsIconic")
	procGetCurrentThreadId       = modKernel32.NewProc("GetCurrentThreadId")
)

const swRestore = 9

// SetDockVisible 在 Windows 上为空操作。
//
// macOS 需要在 Regular / Accessory 激活策略之间切换来控制 Dock 图标，
// Windows 没有对应概念：任务栏按钮由窗口自身的可见性和 HiddenOnTaskbar
// 窗口选项决定，隐藏窗口时任务栏按钮会自动消失。
func SetDockVisible(visible bool) {}

// Activate 在 Windows 上为空操作。
//
// Windows 没有 macOS 那种"激活整个应用"的语义，前台状态是按窗口
// 授予的。真正需要抢占前台时请对具体窗口调用 ForceForeground。
func Activate() {}

// ForceForeground 把指定窗口强行提到前台并交出键盘焦点。
//
// 为什么不能只调用 SetForegroundWindow：Windows 有前台锁定机制，
// 只有当前前台进程（或刚响应过用户输入的进程）才被允许抢占前台。
// Blink 是常驻托盘的后台进程，休息遮罩弹出时前台是用户正在用的其它应用，
// 此时 SetForegroundWindow 会静默失败——表现为遮罩虽然置顶可见，但键盘
// 焦点仍留在背后的应用里（用户以为在遮罩上打字，实际输入进了原程序），
// 且"跳过"按钮的第一次点击只用于激活窗口而被吞掉。
//
// 通行解法是先用 AttachThreadInput 把本线程的输入队列挂到当前前台窗口
// 所属线程上。两个线程共享输入状态期间，系统视本线程为前台的一部分，
// SetForegroundWindow 才会生效。用完立刻分离，否则会拖慢乃至挂起对方线程。
//
// 必须在拥有该窗口的 UI 线程（主线程）上调用，因为 AttachThreadInput
// 操作的是调用线程自身的输入队列。
func ForceForeground(hwnd uintptr) {
	if hwnd == 0 {
		return
	}

	fg, _, _ := procGetForegroundWindow.Call()
	if fg == hwnd {
		return
	}

	// 最小化状态下 SetForegroundWindow 不会还原窗口，先恢复。
	if iconic, _, _ := procIsIconic.Call(hwnd); iconic != 0 {
		procShowWindow.Call(hwnd, swRestore)
	}

	curThread, _, _ := procGetCurrentThreadId.Call()
	var fgThread uintptr
	if fg != 0 {
		fgThread, _, _ = procGetWindowThreadProcessId.Call(fg, 0)
	}

	attached := false
	if fgThread != 0 && fgThread != curThread {
		if ok, _, _ := procAttachThreadInput.Call(curThread, fgThread, 1); ok != 0 {
			attached = true
		}
	}

	procSetForegroundWindow.Call(hwnd)
	procBringWindowToTop.Call(hwnd)
	procSetFocus.Call(hwnd)

	if attached {
		procAttachThreadInput.Call(curThread, fgThread, 0)
	}
}
