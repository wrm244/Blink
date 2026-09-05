package breakengine

import (
	"runtime"
	"testing"
	"time"
)

// 本文件是 Stop/Start 生命周期与时序精度的回归测试。
// 每个测试都对应一个在代码审查中发现的缺陷，修复前必须为红。

// ---- 缺陷 1：Stop→Start 泄漏后台循环 ----
//
// loop / idleLoop / externalSuspendLoop / windowLoop 每轮循环都从
// e.stopCh、e.cmdCh 字段重新读取 channel。Stop() 关闭的只是"当时那一代"
// 的 channel；若 Start() 在旧循环回到 select 之前就替换了字段，
// 旧循环会挂到新一代的 channel 上继续存活——于是同一个引擎同时有两套
// 循环在驱动状态机：每秒 tick 两次、每秒音频检测两次、emit 翻倍。
// 因为用户"停止后再开始"通常就是最后一次 Start，这套僵尸循环会一直
// 活到进程退出。
func TestStopStartDoesNotLeakLoops(t *testing.T) {
	base := runtime.NumGoroutine()

	// 基线：单独 Start 一套引擎时后台循环的数量。
	ref := newTestEngine()
	ref.Start()
	time.Sleep(300 * time.Millisecond)
	running := runtime.NumGoroutine() - base
	ref.Stop()
	time.Sleep(300 * time.Millisecond)
	// 不断言具体数量（循环合并等重构会改变它），只要求基线可信：
	// loop + idleLoop + windowLoop。
	if running < 3 {
		t.Fatalf("基线异常：单个引擎至少有 3 个后台循环，实际 %d", running)
	}

	// 被测：Stop 之后立刻 Start，不给旧循环退出的机会。
	e := newTestEngine()
	e.Start()
	e.Stop()
	e.Start()
	// 等足两个 ticker 周期，确保旧循环已从"卡在旧 channel"迁移完毕。
	time.Sleep(2500 * time.Millisecond)

	after := runtime.NumGoroutine() - base
	e.Stop()

	if after > running {
		t.Errorf("Stop→Start 后残留 %d 个僵尸循环：存活 %d 个，应只有 %d 个",
			after-running, after, running)
	}
}

// ---- 缺陷 2：setPhase 采样了两次时钟 ----
//
// phaseEnd 与 phaseStart 分别取自两次 time.Now()，于是
// phaseEnd != phaseStart+dur。recordFocus/recordBreak 用
// time.Since(phaseStart) 计算实际时长，这个偏差会直接渗进统计数据，
// 也让"阶段结束时刻"与"阶段开始时刻 + 时长"对不上。
func TestSetPhaseUsesSingleClockSample(t *testing.T) {
	e := newTestEngine()
	const dur = 20 * time.Minute

	e.setPhase(PhaseFocusing, dur)

	if got := e.phaseEnd.Sub(e.phaseStart); got != dur {
		t.Errorf("phaseEnd-phaseStart=%v，应精确等于 %v（setPhase 采样了两次时钟）", got, dur)
	}
}

// ---- 缺陷 3：专注已到期时，预提醒反而把休息推迟了整个提醒时长 ----
//
// advancePhase 先判 remaining<=preDur（过期时同样成立）再判 remaining<=0，
// 于是第二个分支永远走不到；startPreBreak 的截断又要求 rem>0 才生效。
// 结果是专注倒计时已经归零时，引擎不是去休息，而是插入一个完整的
// 提醒阶段，把休息又往后推了 preDur 秒。
func TestExpiredFocusGoesStraightToBreak(t *testing.T) {
	e := newTestEngine()
	startEngine(e)
	// 专注已过期（剩余为负）。
	e.phaseEnd = time.Now().Add(-time.Second)

	e.tick(time.Now())

	if e.phase == PhasePreBreak {
		t.Fatalf("专注已到期却进入了预提醒阶段，休息被推迟 %d 秒", e.settings.PreBreakWarningSec)
	}
	if e.phase != PhaseShortBreak {
		t.Fatalf("专注到期后阶段=%s，应为 shortbreak", e.phase)
	}
}

// TestPreBreakStillFiresBeforeExpiry 是上一条的对照：正常路径下
// （剩余时间落在提醒窗口内且仍 > 0）预告阶段必须照常出现。
func TestPreBreakStillFiresBeforeExpiry(t *testing.T) {
	e := newTestEngine()
	startEngine(e)
	// 剩余 3 秒，提醒窗口 10 秒：应进入预告，且不推迟休息。
	e.phaseEnd = time.Now().Add(3 * time.Second)

	e.tick(time.Now())

	if e.phase != PhasePreBreak {
		t.Fatalf("正常预告路径被破坏：阶段=%s，应为 prebreak", e.phase)
	}
	if got := e.state().RemainingSec; got > 3 || got < 0 {
		t.Errorf("预告剩余=%d 秒，应 <= 3（截断到专注剩余时间，不推迟休息）", got)
	}
}

// ---- 缺陷 5：状态没变也每秒投递一次事件 ----
//
// 暂停（以及其它任何冻结态）下每秒的快照完全相同，但 emit 仍无条件
// 投递：Go 侧每秒做一次 JSON 序列化并广播给每个 webview，前端再把
// 一组一模一样的值赋给响应式对象。下游（托盘、遮罩、提醒窗口）都有
// 各自的本地时钟或变更检测，不靠这些重复事件续命。
func TestEmitSkippedWhenStateUnchanged(t *testing.T) {
	e := newTestEngine()
	startEngine(e)
	e.Pause() // 冻结：后续 tick 的快照完全相同
	afterPause := e.emitCount

	e.tick(time.Now())
	e.tick(time.Now().Add(time.Second))
	e.tick(time.Now().Add(2 * time.Second))

	if got := e.emitCount - afterPause; got != 0 {
		t.Errorf("状态未变仍投递了 %d 次事件，应为 0", got)
	}
}

// TestEmitStillFiresOnRealChange 是上一条的对照：倒计时真正推进时
// 事件必须照常投递，去重不能把正常的心跳也吃掉。
func TestEmitStillFiresOnRealChange(t *testing.T) {
	e := newTestEngine()
	startEngine(e)
	// 回拨 phaseEnd 让倒计时随墙钟推进而递减（不依赖两次 time.Now()
	// 之间的时钟粒度）。
	e.phaseEnd = e.phaseEnd.Add(-3 * time.Second)
	before := e.emitCount

	e.tick(time.Now())

	if got := e.emitCount - before; got == 0 {
		t.Error("倒计时推进时未投递事件，去重把正常心跳也吃掉了")
	}
}

// ---- 缺陷 6：CGo 音频探测是在持有引擎锁时执行的 ----
//
// 实测 platform.AudioActivity 约 2.3ms（见 bench_test.go 的
// CheckExternalSuspend Disabled / Enabled 对比）。把它放在临界区内，
// 意味着前端每一次绑定调用（GetState / SaveSettings / 统计查询）都可能
// 在锁上排队等掉这 2ms，主循环的 tick 也会被拖晚同样的时间——对一个
// 靠 1 秒 ticker 推进的倒计时来说，这是实打实的抖动来源。
func TestAudioProbeDoesNotHoldEngineLock(t *testing.T) {
	e := newTestEngine()
	startEngine(e)
	e.settings.PauseOnMeeting = true

	// 用一个人为放慢的探针把"探测中"这个时间窗放大到可观测。
	const probeDelay = 60 * time.Millisecond
	e.audioProbe = func() (bool, bool) {
		time.Sleep(probeDelay)
		return false, false
	}

	running := make(chan struct{})
	done := make(chan struct{})
	go func() {
		close(running)
		e.tickOnce(time.Now())
		close(done)
	}()
	<-running
	time.Sleep(10 * time.Millisecond) // 确保已经进入探测

	// 探测进行中获取引擎锁：若探测持锁，这里会一直等到探测结束。
	start := time.Now()
	e.mu.Lock()
	waited := time.Since(start)
	e.mu.Unlock()
	<-done

	if waited > probeDelay/2 {
		t.Errorf("探测期间获取引擎锁等待了 %v：CGo 探测正持有引擎锁执行，"+
			"会阻塞前端绑定调用与主循环 tick（探针耗时 %v）", waited, probeDelay)
	}
}

// ---- 缺陷 4：开关全关时仍每秒执行 CGo 音频探测 ----
//
// checkExternalSuspend 无条件调用 platform.AudioActivity()——那是一次
// malloc + 遍历系统全部音频进程对象的 CGo 调用——然后才在
// applyExternalSuspend 里用设置开关过滤掉结果。两个开关都关闭时
// 这是纯粹的每秒空转。
func TestNoAudioProbeWhenBothSwitchesOff(t *testing.T) {
	e := newTestEngine()
	e.settings.PauseOnMeeting = false
	e.settings.PauseOnMedia = false
	startEngine(e)

	calls := 0
	e.audioProbe = func() (bool, bool) { calls++; return false, false }

	e.checkExternalSuspend()

	if calls != 0 {
		t.Errorf("两个开关都关闭时仍探测音频 %d 次，应为 0", calls)
	}
}

// TestTickOnceSkipsProbeWhenSwitchesOff 覆盖生产路径本身。
// 上面几条走的是 checkExternalSuspend（锁内检测，测试友好），而线上
// 跑的是 tickOnce → probeExternal + applyExternalResult。两条路径的
// 跳过条件必须一致，否则测试覆盖的和线上执行的就不是同一套逻辑。
func TestTickOnceSkipsProbeWhenSwitchesOff(t *testing.T) {
	e := newTestEngine()
	e.settings.PauseOnMeeting = false
	e.settings.PauseOnMedia = false
	startEngine(e)

	calls := 0
	e.audioProbe = func() (bool, bool) { calls++; return true, true }

	e.tickOnce(time.Now())

	if calls != 0 {
		t.Errorf("生产路径在开关全关时仍探测了 %d 次，应为 0", calls)
	}
	// 跳过期间标志必须被清零，重新打开开关才能形成边沿。
	if e.meeting || e.media {
		t.Errorf("生产路径跳过探测后标志未清零：meeting=%v media=%v", e.meeting, e.media)
	}
}

// TestTickOnceProbesWhenSwitchOn 是上一条的对照：开关打开时生产路径
// 必须照常探测。
func TestTickOnceProbesWhenSwitchOn(t *testing.T) {
	e := newTestEngine()
	e.settings.PauseOnMeeting = true
	startEngine(e)

	calls := 0
	e.audioProbe = func() (bool, bool) { calls++; return false, false }

	e.tickOnce(time.Now())

	if calls != 1 {
		t.Errorf("生产路径在开关打开时探测 %d 次，应为 1", calls)
	}
}

// TestReEnableSwitchStillTriggersAfterSkip 锁住跳过探测的副作用：
// 跳过期间必须清零检测标志，否则重新打开开关后首轮检测形不成边沿，
// 条件明明还活跃却不会自动暂停。
func TestReEnableSwitchStillTriggersAfterSkip(t *testing.T) {
	e := newTestEngine()
	startEngine(e)
	// 会议进行中，自动暂停生效。
	e.applyExternalSuspend(true, false)
	if !e.paused {
		t.Fatal("设置：会议开始未暂停")
	}
	e.Resume() // 用户接管，回到非暂停态以便观察下一轮边沿

	// 用户关掉开关：探测被跳过。
	e.settings.PauseOnMeeting = false
	e.settings.PauseOnMedia = false
	e.checkExternalSuspend()

	// 重新打开开关，会议仍在进行——必须以"边沿"重新触发暂停。
	e.settings.PauseOnMeeting = true
	e.audioProbe = func() (bool, bool) { return true, false }
	e.checkExternalSuspend()

	if !e.paused || !e.autoPaused {
		t.Errorf("重新打开开关后条件仍活跃却未自动暂停：paused=%v autoPaused=%v", e.paused, e.autoPaused)
	}
}

// TestAudioProbeRunsWhenSwitchOn 是上一条的对照：至少一个开关打开时
// 探测必须照常执行，不能为了省一次调用把功能关掉。
func TestAudioProbeRunsWhenSwitchOn(t *testing.T) {
	for _, tc := range []struct{ meeting, media bool }{{true, false}, {false, true}} {
		e := newTestEngine()
		e.settings.PauseOnMeeting = tc.meeting
		e.settings.PauseOnMedia = tc.media
		startEngine(e)

		calls := 0
		e.audioProbe = func() (bool, bool) { calls++; return true, true }

		e.checkExternalSuspend()

		if calls != 1 {
			t.Errorf("开关(meeting=%v media=%v)打开时探测次数=%d，应为 1", tc.meeting, tc.media, calls)
		}
	}
}
