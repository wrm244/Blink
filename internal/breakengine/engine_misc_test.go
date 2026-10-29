package breakengine

import (
	"testing"
	"time"
)

// ---- 空闲保持测试 ----

// TestIdleHoldsFocusTimer 验证在 PhaseIdle 期间专注倒计时保持满值不变，
// 不会向休息推进。
func TestIdleHoldsFocusTimer(t *testing.T) {
	e := newTestEngine()
	now := startEngine(e)
	e.phase = PhaseIdle
	e.idle = true

	// 小于 sleepGap 的间隔（如真实 1s ticker 产生的间隔）不得推进
	// 保持的计时器或改变阶段。
	e.tick(now.Add(3 * time.Second))

	st := e.state()
	if st.Phase != PhaseIdle {
		t.Fatalf("空闲期间 tick 改变了阶段：%s", st.Phase)
	}
	if !fullRemainingOK(st.RemainingSec, e) {
		t.Errorf("空闲计时器移动了：remaining=%d，应为完整 %d", st.RemainingSec, int(e.focusDur()/time.Second))
	}
}

// TestIdleFromPreBreakKeepsFullFocusDuration 是回归测试：预提醒阶段只有
// 10 秒，进入空闲后不能把预提醒的 total 当成下一次专注的完整时长。
func TestIdleFromPreBreakKeepsFullFocusDuration(t *testing.T) {
	e := newTestEngine()
	startEngine(e)
	e.startPreBreak()
	if e.total != e.preDur() {
		t.Fatalf("设置：预提醒 total=%v，应为 %v", e.total, e.preDur())
	}

	e.enterIdle(time.Minute)

	st := e.state()
	if st.Phase != PhaseIdle {
		t.Fatalf("进入空闲后阶段=%s，应为 idle", st.Phase)
	}
	if st.TotalSec != int(e.focusDur()/time.Second) || !fullRemainingOK(st.RemainingSec, e) {
		t.Errorf("空闲状态 total=%d remaining=%d，应为完整专注周期 %d 秒", st.TotalSec, st.RemainingSec, int(e.focusDur()/time.Second))
	}
}

// ---- 窗口期望状态测试 ----

// TestPublishWantsKeepsLatestOnFullChannel 验证通道满时最坏的情况——
// 丢失撤掉遮罩的请求——不会再发生。
//
// 旧实现把窗口操作当命令队列，通道满就丢弃。丢一条 show 会在下一次
// 阶段切换时自愈，但丢一条 hide 会让全屏遮罩（ScreenSaver 层级，覆盖
// Dock 和菜单栏）永久挡在用户屏幕上，只能强杀进程。改成"期望状态"
// 之后，新期望总是覆盖尚未消费的旧期望，最新的一条必定保留。
func TestPublishWantsKeepsLatestOnFullChannel(t *testing.T) {
	e := newTestEngine()
	e.winCh = make(chan winWants, 1)
	e.showOverlays() // 填满通道

	done := make(chan struct{})
	go func() {
		e.hideOverlays() // 不得在满通道上阻塞
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("publishWants 在满通道上阻塞；非阻塞投递失效")
	}

	if len(e.winCh) != 1 {
		t.Fatalf("期望被重复入队：len=%d，应为 1", len(e.winCh))
	}
	if got := <-e.winCh; got.overlays {
		t.Error("最新的 hideOverlays 被丢弃，保留的仍是 showOverlays（遮罩会永久残留）")
	}
}

// TestWantsCoalesceAcrossKinds 验证不同种类的期望会合并成一条完整快照，
// 而不像命令队列那样互相覆盖导致信息丢失。
func TestWantsCoalesceAcrossKinds(t *testing.T) {
	e := newTestEngine()
	e.winCh = make(chan winWants, 1)

	e.showNotice()
	e.showOverlays()
	e.hideOverlays()

	got := <-e.winCh
	if got.overlays {
		t.Error("overlays 期望应为 false")
	}
	if !got.notice {
		t.Error("notice 期望丢失：合并成完整快照时不应互相覆盖")
	}
}

// TestWindowLoopAppliesLatestWantsOnStop 固定关闭顺序不变式：
// Stop 在关闭 stopCh 之前投递"什么都不显示"的期望，windowLoop 在退出前
// 必须应用它，不得让遮罩残留在屏幕上。
func TestWindowLoopAppliesLatestWantsOnStop(t *testing.T) {
	e := newTestEngine()
	e.started = true
	e.stopCh = make(chan struct{})
	e.winCh = make(chan winWants, 1)

	e.Stop() // 内部：hideNotice + hideOverlays + close(stopCh)

	select {
	case <-e.stopCh:
	default:
		t.Fatal("Stop 未关闭 stopCh")
	}
	if e.started {
		t.Error("Stop 未清除 started")
	}

	select {
	case w := <-e.winCh:
		if w.overlays || w.notice {
			t.Errorf("Stop 投递的期望=%+v，应为全部关闭", w)
		}
	default:
		t.Fatal("Stop 未在关闭 stopCh 前投递隐藏期望（遮罩会残留）")
	}
}

// TestWindowLoopExitsOnPendingWants 验证 windowLoop 收到停止信号后
// 会消费完队列里最后一条期望并立刻返回（不会因为通道里还有东西而卡住）。
func TestWindowLoopExitsOnPendingWants(t *testing.T) {
	e := newTestEngine()
	stopCh := make(chan struct{})
	winCh := make(chan winWants, 1)
	winCh <- winWants{} // 带一条未消费的期望

	done := make(chan struct{})
	go func() {
		e.windowLoop(stopCh, winCh, nil)
		close(done)
	}()
	close(stopCh)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("windowLoop 未在 stopCh 关闭后退出")
	}
}

// ---- 从空闲回到专注的标记测试 ----

// TestStartFocusClearsIdleFlag 验证新专注周期不会带着 idle=true 出生。
// 从 PhaseIdle 直接进入 startFocus 的路径（PostponeBreak、检测到休眠
// 间隔）若不清理 idle，快照会短暂出现 phase=focusing 且 idle=true 的
// 自相矛盾状态，托盘据此显示"已暂停"，前端也不敢推进进度条。
func TestStartFocusClearsIdleFlag(t *testing.T) {
	e := newTestEngine()
	startEngine(e)

	e.enterIdle(time.Minute)
	if !e.state().Idle {
		t.Fatal("进入空闲后 idle 应为 true")
	}

	e.PostponeBreak() // 从空闲直接开新专注周期

	st := e.state()
	if st.Phase != PhaseFocusing {
		t.Fatalf("阶段=%s，应为 focusing", st.Phase)
	}
	if st.Idle {
		t.Error("聚焦中仍标记 idle=true，托盘会显示「已暂停」且进度条停滞")
	}
}

// ---- 预提醒时长截断测试 ----

// TestStartPreBreakCapsDurationToRemaining 是回归测试：专注接近尾声时用户
// 把提醒时长调大（如剩余 10 秒时调到 60 秒），提醒阶段必须按剩余时间走，
// 否则 setPhase 会把提醒阶段固定为完整时长，休息被额外推迟。
func TestStartPreBreakCapsDurationToRemaining(t *testing.T) {
	e := newTestEngine()
	startEngine(e)
	// 模拟专注已接近尾声：剩余 10 秒，而提醒时长是 60 秒。
	e.phaseEnd = time.Now().Add(10 * time.Second)
	e.settings.PreBreakWarningSec = 60

	e.startPreBreak()

	if e.phase != PhasePreBreak {
		t.Fatalf("startPreBreak 后阶段=%s，应为 prebreak", e.phase)
	}
	// 提醒阶段时长应被截断为剩余时间（~10s），而非完整 60s。
	got := e.phaseEnd.Sub(time.Now())
	if !within(got, 10*time.Second, 2*time.Second) {
		t.Errorf("提醒阶段时长=%v，应为 ~10s（截断），而不是完整的 60s", got)
	}
}

// TestStartPreBreakKeepsFullDurationWhenPlentyRemaining 验证提醒时长
// 短于剩余专注时间时不做截断（正常路径不受影响）。
func TestStartPreBreakKeepsFullDurationWhenPlentyRemaining(t *testing.T) {
	e := newTestEngine()
	startEngine(e)
	// 剩余 5 分钟，提醒时长 10 秒：不截断。
	e.phaseEnd = time.Now().Add(5 * time.Minute)
	e.settings.PreBreakWarningSec = 10

	e.startPreBreak()

	got := e.phaseEnd.Sub(time.Now())
	if !within(got, 10*time.Second, 2*time.Second) {
		t.Errorf("提醒阶段时长=%v，应为完整的 ~10s", got)
	}
}

// TestTickAfterStopPublishesNoWants 固定 Stop 与在途 tick 的竞态契约。
//
// tickOnce 会在锁外做一次 ~2ms 的音频探测，这段时间里 Stop() 可能已经
// 执行完（它只等锁、不等循环收尾）。若此时不检查 started 就继续推进状态
// 机，落在阶段边界的这一次 tick 会 startBreak() 并投递 {overlays:true}；
// 而停止时 windowLoop 的 select 里 stopCh 已关闭、winCh 又有值，选中后者
// 就会弹出一个 ScreenSaver 级别的全屏遮罩，此后没有任何 goroutine 会撤掉
// 它——用户只能强杀进程。这里直接驱动 tickOnce，断言停止后不再投递期望。
func TestTickAfterStopPublishesNoWants(t *testing.T) {
	e := newTestEngine()
	e.audioProbe = func() (meeting, media bool) { return false, false }
	e.winCh = make(chan winWants, 1)

	now := time.Now()
	e.mu.Lock()
	e.phase = PhaseFocusing
	e.phaseEnd = now.Add(-time.Millisecond) // 本 tick 就该切休息
	e.idle = false
	e.paused = false
	e.started = false // Stop 已经执行完
	e.mu.Unlock()

	e.tickOnce(now)

	select {
	case w := <-e.winCh:
		t.Fatalf("停止后的 tick 仍投递了窗口期望 %+v（遮罩会永久挡屏）", w)
	default:
	}
	if e.phase != PhaseFocusing {
		t.Errorf("停止后的 tick 推进了阶段到 %s", e.phase)
	}
}
