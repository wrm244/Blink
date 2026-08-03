package breakengine

import (
	"testing"
	"time"

	"blink/internal/config"
)

// newTestEngine 返回一个禁用音效、使用默认时长的引擎，
// 确保测试是确定性的且不触碰平台音效/UI 代码。
func newTestEngine() *Engine {
	s := config.Default()
	s.SoundEnabled = false
	return New(s)
}

// startEngine 以 Start() 的方式初始化引擎（全新专注阶段 + 合理的
// lastTick 基线），但不启动 goroutine 也不触碰 Wails 应用。
func startEngine(e *Engine) time.Time {
	now := time.Now()
	e.startFocus()
	e.lastTick = now
	return now
}

// fullRemainingOK 报告 remaining 是否等于完整专注时长，
// 允许亚秒级的墙钟漂移（阶段设置与读取之间的时间差）。
func fullRemainingOK(sec int, e *Engine) bool {
	full := int(e.focusDur() / time.Second)
	return sec >= full-1 && sec <= full
}

// within 报告 a 与 b 的差值是否在 tol 范围内。
func within(a, b, tol time.Duration) bool {
	d := a - b
	return d >= -tol && d <= tol
}

// ---- 休眠间隔重置测试 ----

func TestTickNormalGapDoesNotResetFocus(t *testing.T) {
	e := newTestEngine()
	startEngine(e)
	// 显式回拨 phaseEnd 表示已过 3 秒，而不是指望 startFocus 与 state()
	// 两次调用之间的墙钟漂移。Windows 的单调时钟粒度约 0.5ms，两次紧邻的
	// time.Now() 常常返回完全相同的值，导致 remaining 恰好等于完整时长；
	// macOS 上亚微秒的漂移会让它掉到 full-1，测试才"碰巧"通过。
	e.phaseEnd = e.phaseEnd.Add(-3 * time.Second)

	phaseEndBefore := e.phaseEnd
	e.tick(time.Now().Add(time.Second)) // 1s 间隔 <= sleepGap

	if e.phase != PhaseFocusing {
		t.Fatalf("普通 tick 改变了阶段：%s", e.phase)
	}
	if !e.phaseEnd.Equal(phaseEndBefore) {
		t.Errorf("普通 tick 重置了 phaseEnd：%v -> %v", phaseEndBefore, e.phaseEnd)
	}
	st := e.state()
	if st.RemainingSec >= int(e.focusDur()/time.Second) {
		t.Errorf("tick 后倒计时未推进：remaining=%d", st.RemainingSec)
	}
}

func TestTickSleepGapResetsFocusPeriod(t *testing.T) {
	e := newTestEngine()
	startEngine(e)

	// 间隔大于 sleepGap 表示机器休眠了：过期的专注周期必须重置为
	// 全新的完整周期，而不是触发一个过期的休息。tick 时间来自真实时钟
	// （引擎基于 time.Now() 重置，而非调用者的时间戳）。
	slept := time.Now().Add(sleepGap + 30*time.Second)
	e.tick(slept)

	if e.phase != PhaseFocusing {
		t.Fatalf("休眠间隔后阶段=%s，应为 focusing", e.phase)
	}
	if got := e.phaseEnd.Sub(time.Now()); !within(got, e.focusDur(), 2*time.Second) {
		t.Errorf("休眠间隔未重置为完整专注周期：phaseEnd 在 %v 后，应为 ~%v", got, e.focusDur())
	}
	if st := e.state(); !fullRemainingOK(st.RemainingSec, e) {
		t.Errorf("休眠重置后 remaining=%d，应为 ~%d", st.RemainingSec, int(e.focusDur()/time.Second))
	}
}

// TestTickSleepGapDuringBreakEndsBreak 是一个恢复测试：休眠后过期的
// 休息必须被结束（且短休息计数器递增），而不是让用户卡在过期的休息遮罩上。
func TestTickSleepGapDuringBreakEndsBreak(t *testing.T) {
	e := newTestEngine()
	now := startEngine(e)
	e.startBreak()

	if e.phase != PhaseShortBreak {
		t.Fatalf("设置：应为短休息，得到 %s", e.phase)
	}

	e.tick(now.Add(sleepGap + time.Minute))

	if e.phase != PhaseFocusing {
		t.Fatalf("休息期间休眠间隔后阶段=%s，应为 focusing", e.phase)
	}
	if e.breaksDone != 1 {
		t.Errorf("结束短休息后 breaksDone=%d，应为 1", e.breaksDone)
	}
}

// ---- 暂停冻结 / 恢复测试 ----

func TestPauseFreezesCountdownAcrossTicks(t *testing.T) {
	e := newTestEngine()
	startEngine(e)
	// 回拨 phaseEnd 表示已过 4 秒（同 TestTickNormalGapDoesNotResetFocus：
	// 不能依赖墙钟在两次调用间自行推进，Windows 时钟粒度太粗）。
	e.phaseEnd = e.phaseEnd.Add(-4 * time.Second)
	// 4s 间隔（<= sleepGap）是普通 tick。
	e.tick(time.Now().Add(4 * time.Second))
	frozen := e.state().RemainingSec
	if frozen >= int(e.focusDur()/time.Second) {
		t.Fatalf("设置：倒计时未推进：remaining=%d", frozen)
	}

	e.Pause()
	if !e.paused {
		t.Fatal("Pause 未设置 paused")
	}
	if e.pausedRemaining <= 0 {
		t.Fatalf("pausedRemaining=%v，应 > 0", e.pausedRemaining)
	}

	// 冻结的倒计时必须在 tick 后保持不变，包括大于 sleepGap 的间隔
	//（长时间暂停不应被误判为系统休眠并重置用户进度）。
	e.tick(time.Now().Add(sleepGap + 10*time.Minute))

	st := e.state()
	if !st.Paused {
		t.Fatal("tick 后状态丢失了 paused 标志")
	}
	if st.Phase != PhaseFocusing {
		t.Fatalf("暂停的引擎改变了阶段：%s", st.Phase)
	}
	if st.RemainingSec != frozen {
		t.Errorf("暂停的倒计时移动了：%d -> %d", frozen, st.RemainingSec)
	}
}

// TestPauseNegativeCases 覆盖空操作守卫：已暂停时再暂停和空闲时暂停
// 不得破坏冻结的状态。
func TestPauseNegativeCases(t *testing.T) {
	e := newTestEngine()
	startEngine(e)
	e.Pause()
	pausedRemaining := e.pausedRemaining

	// 双重暂停是空操作：不得从（仍在运行的）墙钟 phaseEnd 重新冻结，
	// 否则会静默延长倒计时。
	e.Pause()
	if !e.paused || e.pausedRemaining != pausedRemaining {
		t.Errorf("双重暂停改变了状态：paused=%v remaining=%v", e.paused, e.pausedRemaining)
	}

	// 空闲时暂停是空操作（已被不活动有效暂停）。
	e2 := newTestEngine()
	startEngine(e2)
	e2.phase = PhaseIdle
	e2.Pause()
	if e2.paused {
		t.Error("空闲时暂停设置了 paused")
	}
	if e2.phase != PhaseIdle {
		t.Errorf("空闲时暂停改变了阶段：%s", e2.phase)
	}
}

func TestResumeRestoresCountdown(t *testing.T) {
	e := newTestEngine()
	startEngine(e)
	// 模拟 3 秒已过的专注时间。tick() 只评估阶段转换；倒计时本身是
	// 绝对 phaseEnd 对比真实时钟，所以把 phaseEnd 回拨来表示已过时间。
	e.phaseEnd = e.phaseEnd.Add(-3 * time.Second)
	e.Pause()
	frozen := e.pausedRemaining
	if s := int(frozen / time.Second); s < 1196 || s > 1198 {
		t.Fatalf("设置：冻结 remaining=%ds，应为 ~1197s（20分钟减3秒）", s)
	}

	e.Resume()
	if e.paused {
		t.Fatal("Resume 未清除 paused")
	}
	if e.pausedRemaining != 0 {
		t.Errorf("pausedRemaining 未清除：%v", e.pausedRemaining)
	}
	// phaseEnd 必须从冻结的 remaining 重建，而非墙钟。
	// 这正是倒计时从冻结值继续的原因：后续每个 tick 都基于此 phaseEnd
	// 测量，所以恰好剩余 pausedRemaining。（单元测试无法等待真实墙钟秒数
	// 来观察 tick 递减。）
	if got := e.phaseEnd.Sub(time.Now()); !within(got, frozen, 2*time.Second) {
		t.Errorf("Resume 未恢复 phaseEnd：%v 后，应为 ~%v", got, frozen)
	}
}

// TestResumeFromIdleRestartsFocus 是一个恢复测试：手动恢复空闲引擎
// 会开始新的专注周期。
func TestResumeFromIdleRestartsFocus(t *testing.T) {
	e := newTestEngine()
	startEngine(e)
	e.phase = PhaseIdle
	e.idle = true

	e.Resume()

	if e.phase != PhaseFocusing {
		t.Fatalf("从空闲恢复后阶段=%s，应为 focusing", e.phase)
	}
	if e.idle {
		t.Error("从空闲恢复未清除 idle")
	}
	if st := e.state(); !fullRemainingOK(st.RemainingSec, e) {
		t.Errorf("空闲恢复后 remaining=%d，应为完整 %d", st.RemainingSec, int(e.focusDur()/time.Second))
	}
}
