//go:build !darwin && !windows

package platform

// SetDockVisible 在其它平台上为空操作：可见窗口始终有任务栏条目，
// 没有激活策略可切换。
func SetDockVisible(visible bool) {}

// Activate 在其它平台上为空操作。
func Activate() {}

// ForceForeground 在其它平台上为空操作。
func ForceForeground(hwnd uintptr) {}
