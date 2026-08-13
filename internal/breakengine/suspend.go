package breakengine

import (
	"log"
	"time"

	"blink/internal/platform"
)

// ---- 会议/媒体自动暂停 ----
//
// 当检测到会议（麦克风被使用）或媒体播放（音频输出活跃）时，
// 自动冻结专注倒计时；条件全部消失后自动从冻结处恢复。语义与
// 用户手动 Pause 相同（冻结而非重置），但由检测自动触发和解除。

// externalCheckInterval 是外部检测（会议/媒体）的轮询周期。
// 比空闲检测（5 秒）更频繁，让播放结束后的自动恢复几乎无感
// （≤1 秒）；每次仅遍历一次音频进程列表，开销可忽略。
const externalCheckInterval = 1 * time.Second

// externalSuspendLoop 周期性轮询会议/媒体检测并驱动自动暂停。
func (e *Engine) externalSuspendLoop() {
	ticker := time.NewTicker(externalCheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-e.stopCh:
			return
		case <-ticker.C:
			e.mu.Lock()
			e.checkExternalSuspend()
			e.mu.Unlock()
		}
	}
}

// resetExternalState 清除外部检测状态。Start 时调用：引擎停止期间
// 检测标志停留在旧值，直接重启会吞掉"条件仍活跃"的边沿，导致
// 会议/媒体进行中重启引擎却不自动暂停。
func (e *Engine) resetExternalState() {
	e.meeting = false
	e.media = false
	e.autoPaused = false
}

// autoSuspend 冻结倒计时并隐藏提醒窗口。调用者持有 e.mu。
// 仅应在非休息、非空闲、未暂停时调用（由 applyExternalSuspend 保证）。
func (e *Engine) autoSuspend(cause string) {
	remaining := time.Until(e.phaseEnd)
	if remaining < 0 {
		remaining = 0
	}
	e.pausedRemaining = remaining
	e.paused = true
	e.autoPaused = true
	e.pausedAt = time.Now()
	log.Printf("blink/breakengine: 自动暂停：原因=%s 阶段=%s 剩余=%s", cause, e.phase, remaining)
	// 会议/媒体播放期间不应弹出休息提醒打扰用户：隐藏提醒窗口。
	// （专注阶段本来就没有窗口，调用是空操作；预提醒阶段则撤下提醒。）
	e.hideNotice()
	e.hideOverlays()
	e.emit()
}

// autoResume 解除自动暂停，从冻结位置继续倒计时。调用者持有 e.mu。
func (e *Engine) autoResume() {
	e.paused = false
	e.autoPaused = false
	if !e.pausedAt.IsZero() {
		e.pausedAccum += time.Since(e.pausedAt)
	}
	e.phaseEnd = time.Now().Add(e.pausedRemaining)
	e.pausedRemaining = 0
	// 若暂停发生在休息前提醒阶段，恢复时把提醒窗口重新显示出来：
	// 否则用户会在毫无预告的情况下直接迎来休息遮罩。
	if e.phase == PhasePreBreak {
		e.showNotice()
	}
	log.Printf("blink/breakengine: 自动恢复：阶段=%s 剩余=%s", e.phase, e.phaseEnd.Sub(time.Now()))
	e.emit()
}

// applyExternalSuspend 以本轮检测到的会议/媒体状态驱动自动暂停。
// 调用者持有 e.mu。meeting/media 是平台的原始检测结果，
// 是否生效由对应设置开关决定。
//
// 采用"边沿触发"：仅在条件从未活跃变为活跃的那一刻暂停，之后条件
// 持续活跃时不重复暂停。这样用户若在检测期间手动 Resume（表示"我
// 知道，但我现在想继续计时"），不会被每 1 秒一轮的检测立即重新暂停
// ——直到条件先消失、再出现，才会再次自动暂停。
//
// 休息和空闲阶段不做自动暂停，且不更新检测标志：这样阶段结束后若
// 条件仍活跃，会以"边沿"形式再次生效（例如用户离开期间会议开始，
// 回来后计时立即被暂停，而不是悄悄跑起来）。
func (e *Engine) applyExternalSuspend(meeting, media bool) {
	meeting = meeting && e.settings.PauseOnMeeting
	media = media && e.settings.PauseOnMedia
	if e.phase.IsBreak() || e.phase == PhaseIdle {
		return
	}
	started := (meeting && !e.meeting) || (media && !e.media)
	cleared := !meeting && !media
	e.meeting, e.media = meeting, media

	switch {
	case started && !e.paused:
		cause := "media"
		if meeting {
			cause = "meeting"
		}
		e.autoSuspend(cause)
	case cleared && e.autoPaused:
		e.autoResume()
	}
}

// checkExternalSuspend 轮询平台检测（会议/媒体），驱动自动暂停。
// 调用者持有 e.mu。由 externalSuspendLoop 周期性调用。
func (e *Engine) checkExternalSuspend() {
	meeting, media := platform.AudioActivity()
	e.applyExternalSuspend(meeting, media)
}
