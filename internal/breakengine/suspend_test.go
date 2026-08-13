package breakengine

import (
	"testing"
	"time"

	"blink/internal/config"
)

// ---- 会议/媒体自动暂停测试 ----

// TestAutoSuspendFreezesCountdown 验证会议检测触发时倒计时被冻结，
// 且状态标记正确（autoPaused/meeting）。
func TestAutoSuspendFreezesCountdown(t *testing.T) {
	e := newTestEngine()
	startEngine(e)
	// 已过 3 秒（同既有测试：不能依赖墙钟漂移）。
	e.phaseEnd = e.phaseEnd.Add(-3 * time.Second)
	e.tick(time.Now().Add(3 * time.Second))
	before := e.state().RemainingSec

	e.applyExternalSuspend(true, false) // 会议开始

	st := e.state()
	if !st.Paused || !st.AutoPaused {
		t.Fatalf("会议开始后 paused=%v autoPaused=%v，应均为 true", st.Paused, st.AutoPaused)
	}
	if !st.Meeting {
		t.Error("状态未报告 meeting")
	}
	if st.RemainingSec != before {
		t.Errorf("自动暂停改变了剩余时间：%d -> %d", before, st.RemainingSec)
	}
	// 暂停中 tick 不应推进倒计时（也不误判休眠）。
	e.tick(time.Now().Add(10 * time.Minute))
	if got := e.state().RemainingSec; got != before {
		t.Errorf("自动暂停中 tick 移动了倒计时：%d -> %d", before, got)
	}
}

// TestAutoResumeRestoresCountdown 验证条件消失后自动恢复，
// 倒计时从冻结位置继续。
func TestAutoResumeRestoresCountdown(t *testing.T) {
	e := newTestEngine()
	startEngine(e)
	e.phaseEnd = e.phaseEnd.Add(-3 * time.Second)
	e.applyExternalSuspend(true, false)
	frozen := e.pausedRemaining

	e.applyExternalSuspend(false, false) // 会议结束

	if e.paused || e.autoPaused {
		t.Fatalf("会议结束后 paused=%v autoPaused=%v，应均为 false", e.paused, e.autoPaused)
	}
	if e.meeting {
		t.Error("meeting 标志未清除")
	}
	if got := e.phaseEnd.Sub(time.Now()); !within(got, frozen, 2*time.Second) {
		t.Errorf("自动恢复未从冻结值重建 phaseEnd：%v 后，应为 ~%v", got, frozen)
	}
}

// TestAutoSuspendEdgeTriggered 验证边沿触发：条件持续活跃时不重复暂停，
// 先消失再出现才再次暂停。
func TestAutoSuspendEdgeTriggered(t *testing.T) {
	e := newTestEngine()
	startEngine(e)

	e.applyExternalSuspend(true, false) // 边沿 1：暂停
	if !e.paused {
		t.Fatal("首次检测到会议未暂停")
	}
	e.applyExternalSuspend(true, false) // 持续活跃：无变化
	if !e.paused {
		t.Fatal("条件持续活跃时不应改变暂停状态")
	}
	if e.autoPaused != true {
		t.Fatal("持续活跃不应重置 autoPaused")
	}

	// 条件消失再出现：再次触发。先恢复再暂停。
	e.applyExternalSuspend(false, false)
	if e.paused {
		t.Fatal("条件消失后未自动恢复")
	}
	e.applyExternalSuspend(true, false)
	if !e.paused || !e.autoPaused {
		t.Fatal("条件再次出现未触发自动暂停")
	}
}

// TestManualResumeBypassesAutoPause 验证用户手动 Resume 后，
// 条件仍活跃时不会被重新自动暂停（直到条件先消失再出现）。
func TestManualResumeBypassesAutoPause(t *testing.T) {
	e := newTestEngine()
	startEngine(e)

	e.applyExternalSuspend(true, false)
	if !e.paused {
		t.Fatal("设置：会议开始未暂停")
	}

	e.Resume() // 用户手动接管
	if e.paused || e.autoPaused {
		t.Fatalf("Resume 后 paused=%v autoPaused=%v", e.paused, e.autoPaused)
	}

	e.applyExternalSuspend(true, false) // 条件仍活跃：不应重新暂停
	if e.paused {
		t.Fatal("用户 Resume 后条件仍活跃时被重新自动暂停")
	}

	// 条件消失再出现：重新触发。
	e.applyExternalSuspend(false, false)
	e.applyExternalSuspend(true, false)
	if !e.paused || !e.autoPaused {
		t.Fatal("条件消失再出现后未重新自动暂停")
	}
}

// TestAutoSuspendSkipsBreaks 验证休息阶段不自动暂停。
func TestAutoSuspendSkipsBreaks(t *testing.T) {
	e := newTestEngine()
	startEngine(e)
	e.startBreak()

	e.applyExternalSuspend(true, false)

	if e.paused {
		t.Fatal("休息阶段被自动暂停")
	}
	// 休息结束时条件仍活跃：专注开始后应以下一轮边沿触发暂停。
	e.endBreak()
	e.applyExternalSuspend(true, false)
	if !e.paused || !e.autoPaused {
		t.Fatal("休息结束后条件活跃未触发自动暂停")
	}
}

// TestAutoSuspendDuringManualPause 验证用户已手动暂停时，
// 检测到会议不覆盖用户状态，也不改变自动暂停标记。
func TestAutoSuspendDuringManualPause(t *testing.T) {
	e := newTestEngine()
	startEngine(e)
	e.Pause()
	if e.autoPaused {
		t.Fatal("设置：手动暂停不应带 autoPaused")
	}

	e.applyExternalSuspend(true, false)

	if !e.paused || e.autoPaused {
		t.Fatalf("手动暂停被检测覆盖：paused=%v autoPaused=%v", e.paused, e.autoPaused)
	}
	// 会议结束：不是自动暂停，不应自动恢复。
	e.applyExternalSuspend(false, false)
	if !e.paused {
		t.Fatal("手动暂停被自动恢复")
	}
}

// TestAutoSuspendDuringIdle 验证空闲阶段不更新检测标志：
// 用户回来时若条件仍活跃，会以边沿形式触发暂停。
func TestAutoSuspendDuringIdle(t *testing.T) {
	e := newTestEngine()
	startEngine(e)
	e.phase = PhaseIdle

	e.applyExternalSuspend(true, false)
	if e.paused {
		t.Fatal("空闲阶段不应自动暂停")
	}
	if e.meeting {
		t.Fatal("空闲阶段不应更新检测标志（否则恢复后无法边沿触发）")
	}

	e.Resume() // 活动恢复 -> 新的专注周期
	e.applyExternalSuspend(true, false)
	if !e.paused || !e.autoPaused {
		t.Fatal("空闲恢复后条件活跃未触发自动暂停")
	}
}

// TestApplySettingsDisablesAutoPause 验证关闭对应开关时立即解除
// 自动暂停；重新打开后条件仍活跃会再次触发。
func TestApplySettingsDisablesAutoPause(t *testing.T) {
	e := newTestEngine()
	startEngine(e)
	e.applyExternalSuspend(true, false)
	if !e.paused {
		t.Fatal("设置：会议开始未暂停")
	}

	s := e.settings
	s.PauseOnMeeting = false
	e.ApplySettings(s)

	if e.paused {
		t.Fatal("关闭会议开关后未解除自动暂停")
	}
	if e.meeting {
		t.Fatal("关闭会议开关后检测标志未清零")
	}

	// 重新打开：会议仍活跃 -> 边沿触发再次暂停。
	s.PauseOnMeeting = true
	e.ApplySettings(s)
	e.applyExternalSuspend(true, false)
	if !e.paused || !e.autoPaused {
		t.Fatal("重新打开开关后条件活跃未触发自动暂停")
	}
}

// TestAutoPauseDurationExcludedFromStats 验证自动暂停时长与手动暂停
// 一样从专注统计中扣除。
func TestAutoPauseDurationExcludedFromStats(t *testing.T) {
	e := newTestEngine()
	startEngine(e)
	e.phaseStart = e.phaseStart.Add(-600 * time.Second)

	e.applyExternalSuspend(true, false)
	// 模拟暂停了 100 秒后会议结束（直接设定 pausedAccum 绕过墙钟）。
	e.pausedAccum = 100 * time.Second
	e.applyExternalSuspend(false, false)
	e.startBreak()

	d := e.statsStore.GetDay(time.Now().Year(), time.Now().Month(), time.Now().Day())
	if d.FocusSec < 498 || d.FocusSec > 502 {
		t.Errorf("专注统计=%d秒，应为 ~500（600-100 自动暂停）", d.FocusSec)
	}
}

// TestAutoSuspendHiddenByDisabledSetting 验证开关关闭时检测不生效。
func TestAutoSuspendHiddenByDisabledSetting(t *testing.T) {
	s := config.Default()
	s.SoundEnabled = false
	s.PauseOnMeeting = false
	e := New(s)
	e.startFocus()

	e.applyExternalSuspend(true, false)

	if e.paused {
		t.Fatal("开关关闭时检测仍触发了暂停")
	}
}

// TestAutoResumeInPreBreakRestoresNotice 验证预提醒阶段被自动暂停、
// 恢复后提醒窗口重新显示（避免用户在无预告下迎来休息遮罩）。
// 窗口命令经 cmdCh 入队；测试中 cmdCh 为 nil，sendCmd 走 default
// 分支静默丢弃，此处断言恢复后的阶段与倒计时仍正确。
func TestAutoResumeInPreBreakRestoresNotice(t *testing.T) {
	e := newTestEngine()
	startEngine(e)
	e.startPreBreak()
	if e.phase != PhasePreBreak {
		t.Fatalf("设置：应为 prebreak，得到 %s", e.phase)
	}

	e.applyExternalSuspend(true, false)
	if !e.paused {
		t.Fatal("预提醒阶段未被自动暂停")
	}
	e.applyExternalSuspend(false, false)
	if e.paused {
		t.Fatal("条件消失后未自动恢复")
	}
	if e.phase != PhasePreBreak {
		t.Fatalf("自动恢复改变了阶段：%s", e.phase)
	}
	// 倒计时应从冻结处继续（剩余时间与暂停前一致，允许亚秒漂移）。
	if got, want := e.state().RemainingSec, e.settings.PreBreakWarningSec; got > want || got < want-2 {
		t.Errorf("恢复后剩余=%d 秒，应为 ~%d", got, want)
	}
}

// TestResetExternalStateClearsFlags 验证 resetExternalState 清除全部
// 外部检测标志。
func TestResetExternalStateClearsFlags(t *testing.T) {
	e := newTestEngine()
	e.meeting = true
	e.media = true
	e.autoPaused = true

	e.resetExternalState()

	if e.meeting || e.media || e.autoPaused {
		t.Fatalf("resetExternalState 未清零：meeting=%v media=%v autoPaused=%v", e.meeting, e.media, e.autoPaused)
	}
}

// TestRestartReTriggersAutoPause 是一个回归测试：引擎停止期间条件保持
// 活跃时，重启后第一轮检测必须重新触发自动暂停。若重启不清除检测标志
// （e.meeting 停留在 true），边沿会被吞掉，会议/媒体进行中重启引擎
// 却悄悄开始计时。
func TestRestartReTriggersAutoPause(t *testing.T) {
	e := newTestEngine()
	startEngine(e)
	// 引擎停止前检测到会议（标志残留）。
	e.applyExternalSuspend(true, false)

	// 重启：先清除标志，再开始新的专注阶段（与 Start() 的调用顺序一致：
	// resetExternalState 之后紧跟 startFocus，后者清除 paused），
	// 然后跑第一轮检测（条件仍活跃）。
	e.resetExternalState()
	e.startFocus()
	e.applyExternalSuspend(true, false)

	if !e.paused || !e.autoPaused {
		t.Fatal("重启后条件仍活跃未触发自动暂停")
	}
}
