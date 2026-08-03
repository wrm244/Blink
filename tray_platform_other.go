//go:build !windows

package main

import (
	"github.com/wailsapp/wails/v3/pkg/application"
)

// applyTrayPlatform 在 macOS / Linux 上无需额外处理。
//
// macOS 用 SetLabel 在菜单栏直接显示倒计时文字，左键点击由原生菜单跟踪
// 接管（systrayPreClickCallback 在未注册点击处理器时返回 1），因此这里
// 刻意不注册 OnClick——注册了反而会抢走原生菜单行为。
func applyTrayPlatform(t *application.SystemTray, menu *application.Menu) {}
