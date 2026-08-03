//go:build !windows

package main

import (
	"github.com/wailsapp/wails/v3/pkg/application"
)

// installCloseHandler 在 macOS / Linux 上不做任何事。
//
// macOS 已有等价语义且更贴近平台习惯：应用是菜单栏 Accessory，
// ApplicationShouldTerminateAfterLastWindowClosed 为 false，关掉设置窗口
// 天然就是"回到后台"，退出走菜单栏的"退出 Blink"。再弹一个确认框反而
// 违反 macOS 惯例。Windows 上没有这层默认语义，关窗通常等于退出应用，
// 用户需要被明确告知程序仍在驻留，所以只在那边询问。
func installCloseHandler(w *application.WebviewWindow) {}
