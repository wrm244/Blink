package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"blink/internal/breakengine"
)

// initTrayStatus 订阅引擎的 blink:tick 事件，在状态文本变化时更新
// 状态项标签、托盘标签和工具提示。
//
// 用事件驱动取代每秒轮询 GetState()：引擎每次 tick / 阶段切换都会 emit，
// 因此倒计时变化天然逐秒驱动这里；而暂停、空闲等状态不变时不更新，
// 避免了轮询带来的不必要锁竞争与无变化时的重复 UI 调用。
//
// 必须在 goroutine 上执行：末尾的 updateTrayState 会调用 tray.SetLabel /
// SetTooltip，它们内部 InvokeSync 回主线程，从主线程调用会自锁。
//
// 注册完监听后本 goroutine 即可退出，不需要用 select{} 之类的手段把自己
// 挂住——app.Event.On 登记的监听器由 app 持有，回调运行在 app 自己的
// 分发 goroutine 上，与本 goroutine 是否存活无关。以前那个永久阻塞的
// select{} 白白占住一个 goroutine 的栈，还让"这个 goroutine 还在跑"
// 看起来像是订阅有效的必要条件。
func initTrayStatus() {
	if app == nil {
		return
	}
	// 事件监听回调跑在独立 goroutine（见 Wails 的 dispatchEventToListeners），
	// 可安全地执行主线程 UI 更新以外的逻辑；SystemTray 的 SetLabel 内部会
	// 调度到主线程，此处非主线程调用正是安全的。
	// off 用于取消监听；托盘订阅与进程同生命周期，无需调用。
	_ = app.Event.On("blink:tick", func(ev *application.CustomEvent) {
		st, ok := ev.Data.(breakengine.State)
		if !ok {
			return
		}
		updateTrayState(st)
	})
	// 启动时先以当前状态渲染一次，避免托盘一直显示默认 "Blink"。
	updateTrayState(engine.GetState())
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
	// 暂停优先：倒计时冻结，继续显示剩余时间会误导（看起来还在走）。
	// 空闲（PhaseIdle）本质是自动暂停，同样显示暂停态。
	if st.Paused || st.Phase == breakengine.PhaseIdle {
		rem := time.Duration(st.RemainingSec) * time.Second
		label := "⏸"
		tip := tr("pauseResume")
		switch {
		case st.Phase == breakengine.PhaseIdle:
			tip = tr("statusIdle")
		case st.AutoPaused:
			if st.Meeting {
				tip = tr("statusAutoMeeting")
			} else if st.MediaPlaying {
				tip = tr("statusAutoMedia")
			} else {
				tip = tr("pauseResume")
			}
		}
		return label, "Blink · " + tip + " · " + fmtDuration(rem)
	}
	rem := time.Duration(st.RemainingSec) * time.Second
	switch st.Phase {
	case breakengine.PhaseFocusing, breakengine.PhasePreBreak,
		breakengine.PhaseShortBreak, breakengine.PhaseLongBreak:
		return fmtDuration(rem), "Blink · " + trStatus(st, rem)
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
