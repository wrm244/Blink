package breakengine

import (
	"log"
	"time"

	"blink/internal/config"
	"blink/internal/stats"
)

// ---- 公共控制方法（前端通过 Service 调用） ----

// GetState 返回当前引擎状态的快照。
func (e *Engine) GetState() State {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.state()
}

// GetSettings 返回当前设置。
func (e *Engine) GetSettings() config.Settings {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.settings
}

// ApplySettings 更新运行中引擎的设置。专注时长变更会立即重置当前专注周期；
// 进行中的休息不受影响，会自然结束。用户手动暂停会被保留：新的时长在
// Resume 之后的下一个专注周期生效。
//
// 仅 FocusDurationMin 变更会重置周期：它是决定当前阶段结束时间的唯一设置，
// 其它设置（主题、语言、音效、快捷键、休息时长）不应清除用户进行中的专注。
// PreBreakWarningSec 也无需重置——tick 循环每秒实时读取它来决定何时提醒。
// 注意：PhasePreBreak 不包含在重置条件中，因为预提醒阶段的结束时间由
// preDur() 决定而非 focusDur()，重置会取消即将到来的休息。
func (e *Engine) ApplySettings(s config.Settings) {
	e.mu.Lock()
	focusChanged := s.FocusDurationMin != e.settings.FocusDurationMin
	e.settings = s
	if focusChanged && !e.paused && (e.phase == PhaseFocusing || e.phase == PhaseIdle) {
		// startFocus 自身会 emit，此处无需额外 emit。
		e.startFocus()
	} else {
		// 休息进行中、用户已暂停或无计时变更：仅通知前端设置可能已变
		// （例如 BreaksUntilLong 可能已改变）。
		e.emit()
	}
	e.mu.Unlock()
}

// StartBreakNow 强制立即开始休息，跳过休息前提醒。
// 如果休息已在进行中则不做任何操作。
func (e *Engine) StartBreakNow() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.phase.IsBreak() {
		return
	}
	e.startBreak()
}

// SkipBreak 结束当前休息（或休息前提醒）并返回新的专注周期。
// 不在休息中时不做任何操作。
func (e *Engine) SkipBreak() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.phase.IsBreak() || e.phase == PhasePreBreak {
		e.endBreak()
	}
}

// PostponeBreak 取消当前或即将到来的休息，并以完整时长重新开始专注周期，
// 实际上是延迟下一次休息。
func (e *Engine) PostponeBreak() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.phase.IsBreak() || e.phase == PhasePreBreak {
		e.endBreak()
	} else {
		e.startFocus()
	}
}

// Pause 冻结倒计时。阶段会被记录以便 Resume 恢复。
func (e *Engine) Pause() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.paused {
		return
	}
	if e.phase == PhaseIdle {
		// 已被空闲状态有效暂停。
		return
	}
	remaining := time.Until(e.phaseEnd)
	if remaining < 0 {
		remaining = 0
	}
	e.pausedRemaining = remaining
	e.paused = true
	e.pausedAt = time.Now()
	log.Printf("blink/breakengine: 暂停：阶段=%s 剩余=%s", e.phase, remaining)
	e.emit()
}

// Resume 从冻结位置继续暂停的倒计时。当引擎空闲（因不活动而暂停）时，
// 也会重新开始专注周期，这样用户可以手动"继续"而无需等待移动鼠标。
func (e *Engine) Resume() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.phase == PhaseIdle {
		e.idle = false
		e.startFocus()
		return
	}
	if !e.paused {
		return
	}
	e.paused = false
	// 累计本次暂停时长，用于从统计的实际专注时长中扣除暂停部分。
	if !e.pausedAt.IsZero() {
		e.pausedAccum += time.Since(e.pausedAt)
	}
	e.phaseEnd = time.Now().Add(e.pausedRemaining)
	e.pausedRemaining = 0
	log.Printf("blink/breakengine: 继续：阶段=%s 剩余=%s", e.phase, e.phaseEnd.Sub(time.Now()))
	e.emit()
}

// Reset 清除长休息计数器并开始新的专注周期。
func (e *Engine) Reset() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.breaksDone = 0
	e.startFocus()
}

// ---- 统计查询 ----

// GetMonthlyStats 返回指定年月每天的统计数据。
// year 用完整年份（如 2026），month 用 time.Month 枚举。
// 未启用统计时返回空 map。
func (e *Engine) GetMonthlyStats(year int, month int) map[int]stats.DayStats {
	e.mu.Lock()
	store := e.statsStore
	e.mu.Unlock()
	if store == nil {
		return map[int]stats.DayStats{}
	}
	return store.GetMonth(year, time.Month(month))
}

// GetDayStats 返回指定日期的统计数据。
// 未启用统计时返回零值。
func (e *Engine) GetDayStats(year int, month int, day int) stats.DayStats {
	e.mu.Lock()
	store := e.statsStore
	e.mu.Unlock()
	if store == nil {
		return stats.DayStats{}
	}
	return store.GetDay(year, time.Month(month), day)
}
