package breakengine

import (
	"testing"
	"time"

	"blink/internal/config"
)

// newTestEngine returns an engine with sound disabled and default timing so
// tests are deterministic and never touch platform sound/UI code.
func newTestEngine() *Engine {
	s := config.Default()
	s.SoundEnabled = false
	return New(s)
}

// startEngine initialises the engine the way Start() does (fresh focus phase
// with a sane lastTick baseline) without launching goroutines or touching the
// Wails app.
func startEngine(e *Engine) time.Time {
	now := time.Now()
	e.startFocus(now)
	e.lastTick = now
	return now
}

// fullRemainingOK reports whether remaining is the full focus duration,
// allowing for sub-second wall-clock drift between phase setup and the read.
func fullRemainingOK(sec int, e *Engine) bool {
	full := int(e.focusDur() / time.Second)
	return sec >= full-1 && sec <= full
}

// within reports whether a is within tol of b.
func within(a, b, tol time.Duration) bool {
	d := a - b
	return d >= -tol && d <= tol
}

// ---- sleep gap reset ----

func TestTickNormalGapDoesNotResetFocus(t *testing.T) {
	e := newTestEngine()
	startEngine(e)

	phaseEndBefore := e.phaseEnd
	e.tick(time.Now().Add(time.Second)) // 1s gap <= sleepGap

	if e.phase != PhaseFocusing {
		t.Fatalf("normal tick changed phase: %s", e.phase)
	}
	if !e.phaseEnd.Equal(phaseEndBefore) {
		t.Errorf("normal tick reset phaseEnd: %v -> %v", phaseEndBefore, e.phaseEnd)
	}
	st := e.state()
	if st.RemainingSec >= int(e.focusDur()/time.Second) {
		t.Errorf("countdown did not advance after 1s tick: remaining=%d", st.RemainingSec)
	}
}

func TestTickSleepGapResetsFocusPeriod(t *testing.T) {
	e := newTestEngine()
	startEngine(e)

	// A gap larger than sleepGap means the machine slept: the stale focus
	// period must be reset to a fresh full period instead of firing an
	// overdue break. The tick time is derived from the real clock (the engine
	// resets against time.Now(), not the caller's timestamp).
	slept := time.Now().Add(sleepGap + 30*time.Second)
	e.tick(slept)

	if e.phase != PhaseFocusing {
		t.Fatalf("sleep gap left phase=%s, want focusing", e.phase)
	}
	if got := e.phaseEnd.Sub(time.Now()); !within(got, e.focusDur(), 2*time.Second) {
		t.Errorf("sleep gap did not reset to full focus period: phaseEnd in %v, want ~%v", got, e.focusDur())
	}
	if st := e.state(); !fullRemainingOK(st.RemainingSec, e) {
		t.Errorf("remaining after sleep reset = %d, want ~%d", st.RemainingSec, int(e.focusDur()/time.Second))
	}
}

// TestTickSleepGapDuringBreakEndsBreak is a recovery case: an overdue break
// after sleep must be ended (and the short-break counter advanced) rather than
// leaving the user stuck on a stale break overlay.
func TestTickSleepGapDuringBreakEndsBreak(t *testing.T) {
	e := newTestEngine()
	now := startEngine(e)
	e.startBreak()

	if e.phase != PhaseShortBreak {
		t.Fatalf("setup: want short break, got %s", e.phase)
	}

	e.tick(now.Add(sleepGap + time.Minute))

	if e.phase != PhaseFocusing {
		t.Fatalf("sleep gap during break left phase=%s, want focusing", e.phase)
	}
	if e.breaksDone != 1 {
		t.Errorf("breaksDone = %d after ended short break, want 1", e.breaksDone)
	}
}

// ---- Pause freeze / Resume restore ----

func TestPauseFreezesCountdownAcrossTicks(t *testing.T) {
	e := newTestEngine()
	startEngine(e)
	// A 4s gap (<= sleepGap) is a normal tick; time derives from the real
	// clock so Pause's time.Until sees the same elapsed time.
	e.tick(time.Now().Add(4 * time.Second))
	frozen := e.state().RemainingSec
	if frozen >= int(e.focusDur()/time.Second) {
		t.Fatalf("setup: countdown did not advance: remaining=%d", frozen)
	}

	e.Pause()
	if !e.paused {
		t.Fatal("Pause did not set paused")
	}
	if e.pausedRemaining <= 0 {
		t.Fatalf("pausedRemaining = %v, want > 0", e.pausedRemaining)
	}

	// Frozen countdown must survive ticks, including a gap larger than
	// sleepGap (a long pause must not look like system sleep and reset the
	// user's progress).
	e.tick(time.Now().Add(sleepGap + 10*time.Minute))

	st := e.state()
	if !st.Paused {
		t.Fatal("state lost paused flag after ticks")
	}
	if st.Phase != PhaseFocusing {
		t.Fatalf("paused engine changed phase: %s", st.Phase)
	}
	if st.RemainingSec != frozen {
		t.Errorf("paused countdown moved: %d -> %d", frozen, st.RemainingSec)
	}
}

// TestPauseNegativeCases covers the no-op guards: pause while already paused
// and pause while idle must not corrupt the frozen state.
func TestPauseNegativeCases(t *testing.T) {
	e := newTestEngine()
	startEngine(e)
	e.Pause()
	pausedRemaining := e.pausedRemaining

	// Double pause is a no-op: must not re-freeze from the (still running)
	// wall-clock phaseEnd, which would silently extend the countdown.
	e.Pause()
	if !e.paused || e.pausedRemaining != pausedRemaining {
		t.Errorf("double pause mutated state: paused=%v remaining=%v", e.paused, e.pausedRemaining)
	}

	// Pause while idle is a no-op (already effectively paused by inactivity).
	e2 := newTestEngine()
	startEngine(e2)
	e2.phase = PhaseIdle
	e2.Pause()
	if e2.paused {
		t.Error("Pause while idle set paused")
	}
	if e2.phase != PhaseIdle {
		t.Errorf("Pause while idle changed phase: %s", e2.phase)
	}
}

func TestResumeRestoresCountdown(t *testing.T) {
	e := newTestEngine()
	startEngine(e)
	// Simulate 3s of elapsed focus time. tick() only evaluates phase
	// transitions; the countdown itself is the absolute phaseEnd vs the real
	// clock, so pull phaseEnd back to represent the elapsed time.
	e.phaseEnd = e.phaseEnd.Add(-3 * time.Second)
	e.Pause()
	frozen := e.pausedRemaining
	if s := int(frozen / time.Second); s < 1196 || s > 1198 {
		t.Fatalf("setup: frozen remaining = %ds, want ~1197s (20min minus 3s)", s)
	}

	e.Resume()
	if e.paused {
		t.Fatal("Resume did not clear paused")
	}
	if e.pausedRemaining != 0 {
		t.Errorf("pausedRemaining not cleared: %v", e.pausedRemaining)
	}
	// phaseEnd must be rebuilt from the frozen remaining, not the wall clock.
	// That is what makes the countdown continue from the frozen value: every
	// subsequent tick measures against this phaseEnd, so exactly
	// pausedRemaining remains. (A unit test cannot wait out real wall-clock
	// seconds to observe a tick decrement.)
	if got := e.phaseEnd.Sub(time.Now()); !within(got, frozen, 2*time.Second) {
		t.Errorf("Resume did not restore phaseEnd: in %v, want ~%v", got, frozen)
	}
}

// TestResumeFromIdleRestartsFocus is a recovery case: manually resuming an
// idle engine starts a fresh focus period.
func TestResumeFromIdleRestartsFocus(t *testing.T) {
	e := newTestEngine()
	startEngine(e)
	e.phase = PhaseIdle
	e.idle = true

	e.Resume()

	if e.phase != PhaseFocusing {
		t.Fatalf("Resume from idle left phase=%s, want focusing", e.phase)
	}
	if e.idle {
		t.Error("Resume from idle did not clear idle")
	}
	if st := e.state(); !fullRemainingOK(st.RemainingSec, e) {
		t.Errorf("remaining after idle resume = %d, want full %d", st.RemainingSec, int(e.focusDur()/time.Second))
	}
}

// ---- idle hold ----

// TestIdleHoldsFocusTimer verifies the invariant that while PhaseIdle the
// focus countdown is held at full duration across ticks and never advances
// toward a break.
func TestIdleHoldsFocusTimer(t *testing.T) {
	e := newTestEngine()
	now := startEngine(e)
	e.phase = PhaseIdle
	e.idle = true

	// A sub-sleepGap gap (as the real 1s ticker produces) must not advance the
	// held timer or change the phase.
	e.tick(now.Add(3 * time.Second))

	st := e.state()
	if st.Phase != PhaseIdle {
		t.Fatalf("tick during idle changed phase: %s", st.Phase)
	}
	if !fullRemainingOK(st.RemainingSec, e) {
		t.Errorf("idle timer moved: remaining=%d, want full %d", st.RemainingSec, int(e.focusDur()/time.Second))
	}
}

// ---- cmd channel drop semantics ----

// TestSendCmdNonBlockingDrop verifies that a full cmdCh never blocks the state
// machine: the enqueue is skipped and earlier (superseding) commands survive.
func TestSendCmdNonBlockingDrop(t *testing.T) {
	e := newTestEngine()
	e.cmdCh = make(chan windowCmd, 1)
	e.cmdCh <- cmdShowOverlays // fill the channel

	done := make(chan struct{})
	go func() {
		e.sendCmd(cmdHideNotice) // must not block on the full channel
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("sendCmd blocked on a full channel; drop semantics broken")
	}

	if len(e.cmdCh) != 1 {
		t.Fatalf("dropped command was enqueued: len=%d, want 1", len(e.cmdCh))
	}
	if got := <-e.cmdCh; got != cmdShowOverlays {
		t.Errorf("earlier command was displaced: got %v, want cmdShowOverlays", got)
	}
}

// TestSendCmdLaterSupersedesEarlier is the positive counterpart: while the
// channel has room the latest command is enqueued and processed after earlier
// ones (a hide following a show wins).
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
		t.Errorf("command order wrong: %v, want [show hide]", got)
	}
}

// ---- Stop close ordering ----

// TestStopEnqueuesHidesBeforeClosingStopCh pins the ordering invariant that
// Stop hides overlays/notice by enqueuing commands BEFORE closing stopCh, so
// windowLoop drains them (instead of returning immediately) and never leaves
// overlays on screen.
func TestStopEnqueuesHidesBeforeClosingStopCh(t *testing.T) {
	e := newTestEngine()
	e.started = true
	e.stopCh = make(chan struct{})
	e.cmdCh = make(chan windowCmd, 8)

	e.Stop()

	select {
	case <-e.stopCh:
	default:
		t.Fatal("Stop did not close stopCh")
	}
	if e.started {
		t.Error("Stop did not clear started")
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
				t.Errorf("cmdHideNotice at index %d, want first (enqueued before overlays)", i)
			}
		}
		if c == cmdHideOverlays {
			hasOverlays = true
		}
	}
	if !hasNotice {
		t.Error("Stop did not enqueue cmdHideNotice before closing stopCh")
	}
	if !hasOverlays {
		t.Error("Stop did not enqueue cmdHideOverlays before closing stopCh")
	}
}
