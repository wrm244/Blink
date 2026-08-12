package stats

import (
	"os"
	"path/filepath"
)

// DefaultPath 返回统计文件的默认存储路径，与 settings.json 同目录：
// ~/Library/Application Support/Blink/stats.json（macOS）。
//
// 路径不可解析时返回空串，调用方据此退化为纯内存模式。
func DefaultPath() string {
	base, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(base, "Blink", "stats.json")
}
