package breakengine

import (
	"log"
	"time"

	"blink/internal/platform"
)

// ---- 时长计算 ----

// focusDur 返回当前专注周期的时长。
func (e *Engine) focusDur() time.Duration {
	return time.Duration(e.settings.FocusDurationMin) * time.Minute
}

// shortDur 返回短休息的时长。
func (e *Engine) shortDur() time.Duration {
	return time.Duration(e.settings.ShortBreakDurationSec) * time.Second
}

// longDur 返回长休息的时长。
func (e *Engine) longDur() time.Duration {
	return time.Duration(e.settings.LongBreakDurationMin) * time.Minute
}

// preDur 返回休息前提醒的时长。
func (e *Engine) preDur() time.Duration {
	return time.Duration(e.settings.PreBreakWarningSec) * time.Second
}

// shouldLong 报告是否应该进入长休息。
func (e *Engine) shouldLong() bool {
	return e.settings.EnableLongBreaks && e.breaksDone >= e.settings.LongBreakInterval
}

// ---- 阶段切换（调用者持有 e.mu） ----

// setPhase 设置当前阶段、总时长和结束时间，并清除暂停状态。
func (e *Engine) setPhase(p Phase, dur time.Duration) {
	e.phase = p
	e.total = dur
	e.phaseEnd = time.Now().Add(dur)
	e.paused = false
	e.pausedRemaining = 0
}

// startFocus 开始一个新的专注周期，隐藏通知和遮罩窗口。
func (e *Engine) startFocus() {
	e.setPhase(PhaseFocusing, e.focusDur())
	e.hideNotice()
	e.hideOverlays()
	e.emit()
}

// startPreBreak 开始休息前提醒阶段，显示通知窗口。
func (e *Engine) startPreBreak() {
	dur := e.preDur()
	e.setPhase(PhasePreBreak, dur)
	e.showNotice()
	if e.settings.SoundEnabled {
		platform.PlaySound(soundPreBreak)
	}
	e.emit()
}

// startBreak 开始一次休息（短休息或长休息），显示全屏遮罩。
func (e *Engine) startBreak() {
	long := e.shouldLong()
	dur := e.shortDur()
	phase := PhaseShortBreak
	if long {
		dur = e.longDur()
		phase = PhaseLongBreak
	}
	e.setPhase(phase, dur)
	log.Printf("blink/breakengine: 休息开始：阶段=%s 时长=%s（长休息=%v, 已完成短休息=%d）", phase, dur, long, e.breaksDone)
	e.hideNotice()
	e.showOverlays()
	e.emit()
}

// endBreak 结束当前休息，更新计数器并重新开始专注周期。
func (e *Engine) endBreak() {
	ended := e.phase
	switch ended {
	case PhaseLongBreak:
		e.breaksDone = 0
	case PhaseShortBreak:
		e.breaksDone++
	}
	log.Printf("blink/breakengine: 休息结束：阶段=%s 已完成短休息=%d，重新开始专注", ended, e.breaksDone)
	if e.settings.SoundEnabled {
		platform.PlaySound(soundBreakEnd)
	}
	e.startFocus()
}

// ---- tick 推进逻辑（调用者持有 e.mu） ----

// tick 每秒推进状态机一次。调用者持有 e.mu。
func (e *Engine) tick(now time.Time) {
	if e.paused {
		// 暂停时保持 lastTick 新鲜，这样长时间暂停不会在下次活动 tick 时
		// 被误判为系统休眠（否则会重置专注周期并丢失用户的 Resume）。
		// 不触发 emit：状态已冻结（Pause/Resume 在实际切换时才 emit）。
		e.lastTick = now
		return
	}

	// 检测系统休眠：两次 tick 之间的大间隔意味着机器被挂起了。
	// 重置而不是触发一个在休眠期间已过期的休息。
	if e.detectSleepGap(now) {
		return
	}

	// 空闲处理只影响专注周期：用户离开时专注倒计时保持在满值，
	// 回来后获得一个新的完整周期。idleLoop 负责从 focus -> PhaseIdle
	// 的切换，并在切换时 emit，所以这里每秒不需要做什么。
	if e.phase == PhaseIdle {
		return
	}

	if e.phaseEnd.IsZero() {
		return
	}

	e.advancePhase(now)
}

// detectSleepGap 检测系统休眠（两次 tick 间隔过大），并在检测到时
// 重置状态。返回 true 表示已处理（调用方应跳过后续逻辑）。
func (e *Engine) detectSleepGap(now time.Time) bool {
	gap := now.Sub(e.lastTick)
	e.lastTick = now
	if gap <= sleepGap {
		return false
	}
	log.Printf("blink/breakengine: 检测到休眠间隔：gap=%s > sleepGap=%s，重置阶段=%s", gap, sleepGap, e.phase)
	if e.phase.IsBreak() {
		e.endBreak()
	} else {
		e.startFocus()
	}
	return true
}

// advancePhase 根据当前阶段和剩余时间推进状态机。调用者持有 e.mu。
func (e *Engine) advancePhase(now time.Time) {
	remaining := e.phaseEnd.Sub(now)
	switch e.phase {
	case PhaseFocusing:
		// 当剩余时间 <= 预提醒时长时切换到预提醒阶段；
		// 剩余时间 <= 0 时直接开始休息。
		if e.preDur() > 0 && remaining <= e.preDur() {
			e.startPreBreak()
		} else if remaining <= 0 {
			e.startBreak()
		} else {
			e.emit()
		}
	case PhasePreBreak:
		if remaining <= 0 {
			e.startBreak()
		} else {
			e.emit()
		}
	case PhaseShortBreak, PhaseLongBreak:
		if remaining <= 0 {
			e.endBreak()
		} else {
			e.emit()
		}
	default:
		e.emit()
	}
}
