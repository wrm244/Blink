//go:build !darwin && !windows

package platform

// SetLaunchAtLogin 在其它平台上为空操作。
// Linux 等平台没有统一的登录自启机制，静默接受以避免破坏构建。
func SetLaunchAtLogin(enabled bool) error {
	return nil
}
