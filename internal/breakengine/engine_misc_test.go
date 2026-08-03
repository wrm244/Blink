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

// ---- 命令通道丢弃语义测试 ----

// TestSendCmdNonBlockingDrop 验证满的 cmdCh 永远不会阻塞状态机：
// 入队被跳过，更早的（覆盖性）命令保留。
func TestSendCmdNonBlockingDrop(t *testing.T) {
	e := newTestEngine()
	e.cmdCh = make(chan windowCmd, 1)
	e.cmdCh <- cmdShowOverlays // 填满通道

	done := make(chan struct{})
	go func() {
		e.sendCmd(cmdHideNotice) // 不得在满通道上阻塞
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("sendCmd 在满通道上阻塞；丢弃语义失效")
	}

	if len(e.cmdCh) != 1 {
		t.Fatalf("被丢弃的命令被入队了：len=%d，应为 1", len(e.cmdCh))
	}
	if got := <-e.cmdCh; got != cmdShowOverlays {
		t.Errorf("更早的命令被替换了：得到 %v，应为 cmdShowOverlays", got)
	}
}

// TestSendCmdLaterSupersedesEarlier 是正向验证：当通道有空间时，
// 最新命令被入队并在更早的命令之后处理（hide 跟在 show 后面胜出）。
func TestSendCmdLaterSupersedesEarlier(t *testing.T) {
	e := newTestEngine()
	e.cmdCh = make(chan windowCmd, 2)
	e.sendCmd(cmdShowOverlays)
	e.sendCmd(cmdHideOverlays)

	var got []windowCmd
	for len(e.cmdCh) > 0 {
		got = append(got, <-e.cmdCh)
	}
	if len(got) != 2 || got[0] != cmdShowOverlays || got[1] != cmdHideOverlays {
		t.Errorf("命令顺序错误：%v，应为 [show hide]", got)
	}
}

// ---- Stop 关闭顺序测试 ----

// TestStopEnqueuesHidesBeforeClosingStopCh 固定关闭顺序不变式：
// Stop 在关闭 stopCh 之前通过入队命令来隐藏遮罩/通知，这样 windowLoop
// 会排空它们（而非立即返回），永远不会让遮罩残留在屏幕上。
func TestStopEnqueuesHidesBeforeClosingStopCh(t *testing.T) {
	e := newTestEngine()
	e.started = true
	e.stopCh = make(chan struct{})
	e.cmdCh = make(chan windowCmd, 8)

	e.Stop()

	select {
	case <-e.stopCh:
	default:
		t.Fatal("Stop 未关闭 stopCh")
	}
	if e.started {
		t.Error("Stop 未清除 started")
	}

	var cmds []windowCmd
	for len(e.cmdCh) > 0 {
		cmds = append(cmds, <-e.cmdCh)
	}
	hasNotice, hasOverlays := false, false
	for i, c := range cmds {
		if c == cmdHideNotice {
			hasNotice = true
			if i != 0 {
				t.Errorf("cmdHideNotice 在索引 %d，应为 0（在 overlays 之前入队）", i)
			}
		}
		if c == cmdHideOverlays {
			hasOverlays = true
		}
	}
	if !hasNotice {
		t.Error("Stop 未在关闭 stopCh 前入队 cmdHideNotice")
	}
	if !hasOverlays {
		t.Error("Stop 未在关闭 stopCh 前入队 cmdHideOverlays")
	}
}
