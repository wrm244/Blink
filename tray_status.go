package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"blink/internal/breakengine"
)

// trayStatusLoop 监听引擎的 blink:tick 事件，在状态文本变化时更新
// 状态项标签、托盘标签和工具提示。
//
// 用事件驱动取代每秒轮询 GetState()：引擎每次 tick / 阶段切换都会 emit，
// 因此倒计时变化天然逐秒驱动这里；而暂停、空闲等状态不变时不更新，
// 避免了轮询带来的不必要锁竞争与无变化时的重复 UI 调用。
func trayStatusLoop() {
	if app == nil {
		return
	}
	// 事件监听回调跑在独立 goroutine（见 Wails 的 dispatchEventToListeners），
	// 可安全地执行主线程 UI 更新以外的逻辑；SystemTray 的 SetLabel 内部会
	// 调度到主线程，此处非主线程调用正是安全的。
	off := app.Event.On("blink:tick", func(ev *application.CustomEvent) {
		st, ok := ev.Data.(breakengine.State)
		if !ok {
			return
		}
		updateTrayState(st)
	})
	// off 用于取消监听；本 goroutine 生命周期与进程一致，无需调用。
	_ = off
	// 启动时先以当前状态渲染一次，避免托盘一直显示默认 "Blink"。
	updateTrayState(engine.GetState())
	// 阻塞直到退出：事件回调在独立的 goroutine 中执行，不在这里同步等待，
	// 因此用一个空 channel 挂住本 goroutine 以保持监听存活。
	select {}
}

var (
	trayMu      sync.Mutex
	lastLabel   string
	lastTooltip string
)

// updateTrayState 在状态文本变化时更新托盘的三处显示。
// tooltip 与 statusItem（菜单首项）共享同一文本，由同一缓存判断驱动，
// 避免重复计算与无变化的无效重绘。
func updateTrayState(st breakengine.State) {
	label, tooltip := formatStatus(st)
	trayMu.Lock()
	if label == lastLabel && tooltip == lastTooltip {
		trayMu.Unlock()
		return
	}
	lastLabel, lastTooltip = label, tooltip
	trayMu.Unlock()

	if tray != nil {
		tray.SetLabel(label)
		tray.SetTooltip(tooltip)
	}
	if statusItem != nil {
		statusItem.SetLabel(tooltip)
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
