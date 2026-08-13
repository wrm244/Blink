// Package config 管理 Blink 的用户可配置设置，
// 并以 JSON 格式持久化到用户级应用支持目录。
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// 关闭窗口时的可选行为，对应 Settings.CloseAction。
const (
	// CloseActionAsk 每次关闭窗口都弹确认框。
	CloseActionAsk = "ask"
	// CloseActionBackground 关闭窗口后保留在托盘继续计时。
	CloseActionBackground = "background"
	// CloseActionQuit 关闭窗口即退出应用。
	CloseActionQuit = "quit"
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
	// PauseOnMeeting 检测到会议/通话（任何应用正在使用麦克风）时自动暂停计时，
	// 结束后自动恢复。
	PauseOnMeeting bool `json:"pauseOnMeeting"`
	// PauseOnMedia 检测到媒体播放（任何应用正在输出音频，如看视频/听音乐）时
	// 自动暂停计时，结束后自动恢复。
	PauseOnMedia bool `json:"pauseOnMedia"`
	// AutoStart 在应用启动后（完成引导后）自动开始专注计时。
	// 为 false 时用户从托盘手动启动。
	AutoStart bool `json:"autoStart"`
	// LaunchAtLogin 让应用在用户登录系统后自动启动。
	// 由平台层写入系统的登录项/启动项（macOS 登录项、Windows 启动注册表）。
	LaunchAtLogin bool `json:"launchAtLogin"`
	// Onboarded 记录用户是否已完成首次设置。
	// 在其为 true 之前引擎不运行倒计时，应用打开到设置页面而非立即倒计时。
	Onboarded bool `json:"onboarded"`
	// Language 是 UI 语言，"zh-CN" 或 "en"（空 = 跟随系统）。
	Language string `json:"language"`
	// Theme 是 UI 配色方案："system"（跟随系统）、"light" 或 "dark"。
	Theme string `json:"theme"`

	// CloseAction 决定关闭设置窗口时的行为，仅 Windows 使用：
	//   "ask"        询问（默认）——弹出确认框让用户选择
	//   "background" 直接最小化到托盘继续计时
	//   "quit"       直接退出应用
	//
	// macOS 上关闭窗口天然保留在菜单栏，不读取此字段。
	CloseAction string `json:"closeAction"`

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
		PauseOnMeeting:        true,
		PauseOnMedia:          true,
		AutoStart:             true,
		Onboarded:             false,
		Language:              "",
		Theme:                 "system",
		CloseAction:           CloseActionAsk,
		ShortcutStartBreak:    mod + "+Shift+B",
		ShortcutSkipBreak:     mod + "+Shift+S",
		ShortcutPostponeBreak: mod + "+Shift+P",
		ShortcutPreferences:   mod + "+Shift+,",
	}
}

// mod 是本平台主修饰键在 Wails 加速键语法中的名称。
//
// Wails 解析时把 "Cmd" 归一化为 CmdOrCtrl，因此 "Cmd+Shift+B" 在 Windows
// 上确实能注册成 Ctrl+Shift+B——但设置界面会原样显示 "Cmd"，Windows 用户
// 看到的是一个键盘上不存在的键。默认值直接按平台给出正确的字面量。
var mod = func() string {
	if runtime.GOOS == "darwin" {
		return "Cmd"
	}
	return "Ctrl"
}()

// normalizeAccelerator 把 macOS 风格的修饰键名改写为本平台的名称。
//
// 场景：配置文件在平台间迁移，或早期版本在 Windows 上写入了 "Cmd+Shift+B"。
// 这类值功能上可用（Cmd 会被解析成 CmdOrCtrl），但显示出来会误导用户，
// 所以在加载时统一改写。仅替换修饰键 token，不触碰主键。
func normalizeAccelerator(acc string) string {
	if acc == "" || runtime.GOOS == "darwin" {
		return acc
	}
	parts := strings.Split(acc, "+")
	// 最后一段是主键（可能就是 "+" 本身导致的空串），只处理它之前的修饰键。
	for i := 0; i < len(parts)-1; i++ {
		switch strings.ToLower(strings.TrimSpace(parts[i])) {
		case "cmd", "command", "cmdorctrl":
			parts[i] = "Ctrl"
		case "option":
			parts[i] = "Alt"
		}
	}
	return strings.Join(parts, "+")
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
	s.ShortcutStartBreak = normalizeAccelerator(s.ShortcutStartBreak)
	s.ShortcutSkipBreak = normalizeAccelerator(s.ShortcutSkipBreak)
	s.ShortcutPostponeBreak = normalizeAccelerator(s.ShortcutPostponeBreak)
	s.ShortcutPreferences = normalizeAccelerator(s.ShortcutPreferences)
	switch s.CloseAction {
	case CloseActionAsk, CloseActionBackground, CloseActionQuit:
		// 有效值，保留。
	default:
		// 空值（旧配置文件）或无法识别的值都退回默认的"询问"。
		s.CloseAction = d.CloseAction
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
