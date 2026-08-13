//go:build !darwin

package platform

import (
	"os"
	"strings"
)

// SystemLanguage 返回系统首选语言，映射到应用支持的两种语言之一：
// "zh-CN" 或 "en"。非 darwin 平台没有 CGo 的 NSLocale 可用，
// 退而求其次解析 LANG/LC_ALL 环境变量（Linux/Windows 上大体可靠）。
// 解析不出中/英时返回空串，调用方自行决定回退。
func SystemLanguage() string {
	for _, env := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		v := strings.ToLower(os.Getenv(env))
		if strings.HasPrefix(v, "zh") {
			return "zh-CN"
		}
		if strings.HasPrefix(v, "en") {
			return "en"
		}
	}
	return ""
}
