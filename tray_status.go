package main

import (
	"fmt"
	"time"

	"blink/internal/breakengine"
)

// trayStatusLoop 在渲染文本变化时更新状态项标签、托盘标签和工具提示。
// 引擎在倒计时期间每秒发出 tick 事件，专注/休息阶段的格式化标签每秒
// 变化（暂停/空闲期间不变）。通过缓存上次的字符串，在文本相同时跳过
// 主线程的 SetLabel/SetTooltip 调用（每次都触发原生重绘），消除暂停
// 和空闲阶段的无效 UI 工作。
func trayStatusLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	var lastLabel, lastTooltip string
	for range ticker.C {
		st := engine.GetState()
		label, tooltip := formatStatus(st)
		if label == lastLabel && tooltip == lastTooltip {
			continue
		}
		lastLabel, lastTooltip = label, tooltip
		if tray != nil {
			tray.SetLabel(label)
			tray.SetTooltip(tooltip)
		}
		if statusItem != nil {
			statusItem.SetLabel(tooltip)
		}
	}
}

// formatStatus 根据引擎状态生成托盘标签和工具提示。
func formatStatus(st breakengine.State) (label, tooltip string) {
	rem := time.Duration(st.RemainingSec) * time.Second
	switch st.Phase {
	case breakengine.PhaseFocusing, breakengine.PhasePreBreak,
		breakengine.PhaseShortBreak, breakengine.PhaseLongBreak:
		return fmtDuration(rem), "Blink · " + trStatus(st, rem)
	case breakengine.PhaseIdle:
		return "⏸", "Blink · " + tr("pauseResume")
	default:
		// 引擎未运行（如引导前）。
		return "Blink", "Blink"
	}
}

// trStatus 本地化各阶段的工具提示文本。
func trStatus(st breakengine.State, rem time.Duration) string {
	switch st.Phase {
	case breakengine.PhaseFocusing:
		return sfmt("focusing", rem)
	case breakengine.PhasePreBreak:
		return sfmt("prebreak", rem)
	case breakengine.PhaseShortBreak:
		return sfmt("shortbreak", rem)
	case breakengine.PhaseLongBreak:
		return sfmt("longbreak", rem)
	default:
		return ""
	}
}

// 状态字符串，按语言分组。因为托盘在 webview i18n 之外运行，
// 字符串保存在这里。
var statusStrings = map[string]map[string]string{
	"zh-CN": {
		"focusing":   "专注中，距下次休息 %s",
		"prebreak":   "%s 后开始休息",
		"shortbreak": "短休息，剩余 %s",
		"longbreak":  "长休息，剩余 %s",
	},
	"en": {
		"focusing":   "focusing, next break in %s",
		"prebreak":   "break in %s",
		"shortbreak": "short break, %s remaining",
		"longbreak":  "long break, %s remaining",
	},
}

// sfmt 返回本地化的"阶段 · 剩余"字符串。
func sfmt(phase string, rem time.Duration) string {
	if m, ok := statusStrings[menuLangString()]; ok {
		if tpl, ok := m[phase]; ok {
			return fmt.Sprintf(tpl, fmtDuration(rem))
		}
	}
	if tpl, ok := statusStrings["en"][phase]; ok {
		return fmt.Sprintf(tpl, fmtDuration(rem))
	}
	return fmtDuration(rem)
}

// fmtDuration 将时长格式化为 M:SS 形式。
func fmtDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	d = d.Round(time.Second)
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%d:%02d", m, s)
}
