//go:build !darwin

package platform

// SetDockVisible 在非 macOS 平台上为空操作：其它平台始终为可见窗口
// 显示任务栏条目，没有激活策略可切换。
func SetDockVisible(visible bool) {}

// Activate 在非 macOS 平台上为空操作。
func Activate() {}
