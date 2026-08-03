// Package config 管理 Blink 的用户可配置设置，
// 并以 JSON 格式持久化到用户级应用支持目录。
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Settings 是全部用户可配置行为。
//
// 时长按 UI 自然编辑的单位拆分（长时段用分钟，短休息用秒），
// 由 break engine 统一转换为 time.Duration。
type Settings struct {
	// FocusDurationMin 是一次专注周期的时长，之后会提醒休息。
	FocusDurationMin int `json:"focusDurationMin"`
	// ShortBreakDurationSec 是常规短休息的时长。
	ShortBreakDurationSec int `json:"shortBreakDurationSec"`
	// LongBreakDurationMin 是偶尔的较长休息的时长。
	LongBreakDurationMin int `json:"longBreakDurationMin"`
	// LongBreakInterval 是触发长休息前需要完成的短休息次数。
	LongBreakInterval int `json:"longBreakInterval"`
	// PreBreakWarningSec 是休息开始前的提前提醒时长。
	PreBreakWarningSec int `json:"preBreakWarningSec"`
	// IdleThresholdMin：不活动超过此时长后，专注计时器被视为暂停，
	// 活动恢复时重置。
	IdleThresholdMin int `json:"idleThresholdMin"`
	// EnableLongBreaks 切换偶尔的长休息循环。
	EnableLongBreaks bool `json:"enableLongBreaks"`
	// SoundEnabled 切换休息结束提示音和休息前提醒音。
	SoundEnabled bool `json:"soundEnabled"`
	// AutoStart 在应用启动后（完成引导后）自动开始专注计时。
	// 为 false 时用户从托盘手动启动。
	AutoStart bool `json:"autoStart"`
	// Onboarded 记录用户是否已完成首次设置。
	// 在其为 true 之前引擎不运行倒计时，应用打开到设置页面而非立即倒计时。
	Onboarded bool `json:"onboarded"`
	// Language 是 UI 语言，"zh-CN" 或 "en"（空 = 跟随系统）。
	Language string `json:"language"`
	// Theme 是 UI 配色方案："system"（跟随系统）、"light" 或 "dark"。
	Theme string `json:"theme"`

	// 全局键盘快捷键（Wails 加速键语法，如 "Cmd+Shift+B"）。
	ShortcutStartBreak    string `json:"shortcutStartBreak"`
	ShortcutSkipBreak     string `json:"shortcutSkipBreak"`
	ShortcutPostponeBreak string `json:"shortcutPostponeBreak"`
	ShortcutPreferences   string `json:"shortcutPreferences"`
}

// Default 返回内置默认值，基于 20-20-20 规则：
// 20 分钟专注、20 秒短休息、每 4 次短休息后 5 分钟长休息、
// 每次休息前 10 秒提前提醒。
func Default() Settings {
	return Settings{
		FocusDurationMin:      20,
		ShortBreakDurationSec: 20,
		LongBreakDurationMin:  5,
		LongBreakInterval:      4,
		PreBreakWarningSec:    10,
		IdleThresholdMin:      5,
		EnableLongBreaks:      true,
		SoundEnabled:          true,
		AutoStart:             true,
		Onboarded:             false,
		Language:              "",
		Theme:                 "system",
		ShortcutStartBreak:    "Cmd+Shift+B",
		ShortcutSkipBreak:     "Cmd+Shift+S",
		ShortcutPostponeBreak: "Cmd+Shift+P",
		ShortcutPreferences:   "Cmd+Shift+,",
	}
}

// withDefaults 返回 s 中零值/无效值被替换为默认值后的设置。
// 这确保旧的存档文件在新增字段后仍然有效。
func (s Settings) withDefaults() Settings {
	d := Default()
	if s.FocusDurationMin < 1 {
		s.FocusDurationMin = d.FocusDurationMin
	}
	if s.ShortBreakDurationSec < 1 {
		s.ShortBreakDurationSec = d.ShortBreakDurationSec
	}
	if s.LongBreakDurationMin < 1 {
		s.LongBreakDurationMin = d.LongBreakDurationMin
	}
	if s.LongBreakInterval < 1 {
		s.LongBreakInterval = d.LongBreakInterval
	}
	if s.PreBreakWarningSec < 0 {
		s.PreBreakWarningSec = d.PreBreakWarningSec
	}
	if s.IdleThresholdMin < 1 {
		s.IdleThresholdMin = d.IdleThresholdMin
	}
	if s.ShortcutStartBreak == "" {
		s.ShortcutStartBreak = d.ShortcutStartBreak
	}
	if s.ShortcutSkipBreak == "" {
		s.ShortcutSkipBreak = d.ShortcutSkipBreak
	}
	if s.ShortcutPostponeBreak == "" {
		s.ShortcutPostponeBreak = d.ShortcutPostponeBreak
	}
	if s.ShortcutPreferences == "" {
		s.ShortcutPreferences = d.ShortcutPreferences
	}
	return s
}

// configPath 返回设置文件的路径。它位于 OS 的用户级应用支持目录下
//（macOS 上为 ~/Library/Application Support），用户通常不需要手动编辑。
func configPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "Blink", "settings.json"), nil
}

// Load 从磁盘读取设置，不存在或不可读时回退到默认值（并创建文件）。
func Load() (Settings, error) {
	s := Default()
	path, err := configPath()
	if err != nil {
		return s, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// 尚未保存过：持久化默认值以确保文件存在。
			_ = Save(s)
			return s, nil
		}
		return s, err
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return Default(), err
	}
	return s.withDefaults(), nil
}

// Save 将设置写入磁盘，必要时创建目录。
func Save(s Settings) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
