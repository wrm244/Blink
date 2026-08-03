//go:build windows

package platform

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modWinmm       = windows.NewLazySystemDLL("winmm.dll")
	procPlaySoundW = modWinmm.NewProc("PlaySoundW")
)

// PlaySoundW 的标志位（见 mmsystem.h）。
const (
	sndAsync     = 0x0001 // 立即返回，音效在后台播放
	sndNoDefault = 0x0002 // 找不到音效时保持安静，不要播放默认"叮"
	sndAlias     = 0x00010000
	sndNoStop    = 0x0010 // 不打断正在播放的音效
)

// soundAliases 把引擎使用的跨平台音效名映射到 Windows 系统音效别名。
//
// 引擎里的名字（"Glass"、"Tink"）源自 macOS 内置音效库，Windows 没有同名音效。
// 这里映射到注册表中的系统事件别名，用户在"设置 → 声音"里改过主题后
// 会自动跟随，比自带 wav 文件更贴合系统。
//
//	Glass（休息结束）→ SystemAsterisk：明确的"完成"提示音
//	Tink （休息前提醒）→ SystemNotification：轻量的通知音
//
// 未收录的名称回退到 SystemAsterisk，避免静默失败。
var soundAliases = map[string]string{
	"Glass": "SystemAsterisk",
	"Tink":  "SystemNotification",
}

// PlaySound 播放与给定名称对应的 Windows 系统音效。
// 调用立即返回；音效在后台播放，且不会打断已在播放的音效。
func PlaySound(name string) {
	alias, ok := soundAliases[name]
	if !ok {
		alias = "SystemAsterisk"
	}

	namePtr, err := windows.UTF16PtrFromString(alias)
	if err != nil {
		return
	}

	// hmod 传 0：使用系统别名时不需要模块句柄。
	procPlaySoundW.Call(
		uintptr(unsafe.Pointer(namePtr)),
		0,
		uintptr(sndAsync|sndAlias|sndNoDefault|sndNoStop),
	)
}
