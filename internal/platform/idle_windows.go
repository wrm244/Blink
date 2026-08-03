//go:build windows

package platform

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modUser32           = windows.NewLazySystemDLL("user32.dll")
	modKernel32         = windows.NewLazySystemDLL("kernel32.dll")
	procGetLastInputInf = modUser32.NewProc("GetLastInputInfo")
	procGetTickCount64  = modKernel32.NewProc("GetTickCount64")
)

// lastInputInfo 对应 Win32 的 LASTINPUTINFO 结构体。
// DwTime 是最后一次输入事件发生时的系统运行毫秒数（与 GetTickCount 同源，
// 32 位，约 49.7 天回绕一次）。
type lastInputInfo struct {
	CbSize uint32
	DwTime uint32
}

// IdleSeconds 返回用户最后一次操作键盘/鼠标以来的秒数。
//
// 使用 GetLastInputInfo：它报告的是系统级最后输入时刻，而非仅本进程收到的
// 输入，语义与 macOS 侧的 CGEventSourceSecondsSinceLastEventType 一致。
//
// 回绕处理：GetLastInputInfo 返回的是 32 位 tick 值，而 GetTickCount64 是
// 64 位。两者直接相减会在 32 位计数器回绕（约 49.7 天）后得到巨大的错误值，
// 因此先把 64 位的当前 tick 截断为 uint32 再做无符号减法——uint32 的模运算
// 天然处理回绕，得到的差值始终正确。
func IdleSeconds() float64 {
	info := lastInputInfo{}
	info.CbSize = uint32(unsafe.Sizeof(info))

	ret, _, _ := procGetLastInputInf.Call(uintptr(unsafe.Pointer(&info)))
	if ret == 0 {
		// 调用失败（例如会话被锁定时的部分场景）：报告为活跃，
		// 宁可少暂停一次，也不要凭错误数据把计时器冻结住。
		return 0
	}

	tick64, _, _ := procGetTickCount64.Call()
	elapsedMS := uint32(tick64) - info.DwTime
	return float64(elapsedMS) / 1000.0
}
