//go:build windows

package platform

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/sys/windows/registry"
)

// regRunKey 是 HKCU 下的自启动注册表键，所有用户级登录自启应用都在这里。
var regRunKey = `Software\Microsoft\Windows\CurrentVersion\Run`

// runEntryNameOnce 缓存登录项在注册表里的值名（来自可执行文件名，进程内不变），
// 避免每次调用 os.Executable()。使用"Blink"而非随机名，便于用户查看与手动清理。
var runEntryNameOnce = sync.OnceValue(func() string {
	exe, err := os.Executable()
	if err != nil {
		return "Blink"
	}
	return strings.TrimSuffix(filepath.Base(exe), filepath.Ext(exe))
})

// SetLaunchAtLogin 在 Windows 上把当前应用加入或移出 HKCU 的 Run 键。
// HKCU 无需管理员权限，且只影响当前用户，适合这类菜单栏小工具。
func SetLaunchAtLogin(enabled bool) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	if !enabled {
		// 移出自启动：删除本应用的条目。
		k, err := registry.OpenKey(registry.CURRENT_USER, regRunKey, registry.QUERY_VALUE|registry.SET_VALUE)
		if err != nil {
			if err == registry.ErrNotExist {
				return nil
			}
			return err
		}
		defer k.Close()
		// 删除不存在的键会返回错误，忽略即可。
		_ = k.DeleteValue(runEntryNameOnce())
		return nil
	}

	k, _, err := registry.CreateKey(registry.CURRENT_USER, regRunKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.SetStringValue(runEntryNameOnce(), `"`+exe+`"`)
}
