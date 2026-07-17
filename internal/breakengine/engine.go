// Package breakengine contains the timer state machine at the heart of
// PocketMind: it cycles through focus, pre-break warning and (short or long)
// break phases, pauses on user idle, survives system sleep, and drives the
// on-screen break overlays.
package breakengine

import (
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"pocketmind/internal/config"
	"pocketmind/internal/platform"
)

const (
	eventTick = "pm:tick"

	// Built-in macOS system sounds: a soft chime when a break ends and a
	// gentle tick when the pre-break warning appears.
	soundBreakEnd = "Glass"
	soundPreBreak = "Tink"

	tickInterval       = 1 * time.Second
	idleCheckInterval  = 5 * time.Second
	// sleepGap: if two ticks are farther apart than this the machine likely
	// slept; rather than fire a stale break we reset the focus period.
	sleepGap = 5 * time.Second
)

// Engine is the break-reminder state machine. It is safe for concurrent use:
// every exported method takes the mutex, and the background ticker is the only
// other writer.
type Engine struct {
	mu       sync.Mutex
	settings config.Settings

	phase    Phase
	phaseEnd time.Time // absolute time at which the current phase ends
	total    time.Duration

	// breaksDone counts short breaks completed since the last long break.
	breaksDone int

	idle   bool
	paused bool
	// pausedRemaining lets Pause/Resume freeze the countdown without the clock
	// running underneath them.
	pausedRemaining time.Duration

	// lastTick is used to detect system sleep (a suddenly large gap).
	lastTick time.Time

	app     *application.App
	overlays []application.Window
	notice   application.Window

	stopCh  chan struct{}
	cmdCh   chan windowCmd
	started bool
}

// New creates an engine bound to the given settings.
func New(s config.Settings) *Engine {
	return &Engine{settings: s}
}

// Start launches the background ticker and idle checker. It is idempotent so
// it is safe to call from a service's ServiceStartup hook.
func (e *Engine) Start() {
	e.mu.Lock()
	if e.started {
		e.mu.Unlock()
		return
	}
	e.started = true
	e.stopCh = make(chan struct{})
	e.cmdCh = make(chan windowCmd, 8)
	e.app = application.Get()
	now := time.Now()
	e.lastTick = now
	e.startFocus(now)
	e.mu.Unlock()

	go e.loop()
	go e.idleLoop()
	go e.windowLoop()
}

// IsStarted reports whether the engine's background ticker is running.
func (e *Engine) IsStarted() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.started
}

// Stop halts the background goroutines. The engine can be restarted with Start.
func (e *Engine) Stop() {
	e.mu.Lock()
	if !e.started {
		e.mu.Unlock()
		return
	}
	e.started = false
	close(e.stopCh)
	e.hideNotice()
	e.hideOverlays()
	e.mu.Unlock()
}

// app_ returns the cached Wails app, set once at Start time.
func (e *Engine) app_() *application.App {
	return e.app
}

// ---- state ----

func (e *Engine) state() State {
	st := State{Phase: e.phase, Idle: e.idle, Paused: e.paused, ShortBreakCount: e.breaksDone}
	if e.settings.EnableLongBreaks {
		st.BreaksUntilLong = e.settings.LongBreakInterval - e.breaksDone
		if st.BreaksUntilLong < 0 {
			st.BreaksUntilLong = 0
		}
	}
	total := e.total
	remaining := total
	if !e.phaseEnd.IsZero() {
		remaining = e.phaseEnd.Sub(time.Now())
	}
	if e.paused {
		remaining = e.pausedRemaining
	}
	if remaining < 0 {
		remaining = 0
	}
	st.TotalSec = int(total / time.Second)
	st.RemainingSec = int(remaining / time.Second)
	return st
}

func (e *Engine) emit() {
	a := e.app_()
	if a == nil {
		return
	}
	a.Event.Emit(eventTick, e.state())
}

// ---- durations ----

func (e *Engine) focusDur() time.Duration { return time.Duration(e.settings.FocusDurationMin) * time.Minute }
func (e *Engine) shortDur() time.Duration  { return time.Duration(e.settings.ShortBreakDurationSec) * time.Second }
func (e *Engine) longDur() time.Duration  { return time.Duration(e.settings.LongBreakDurationMin) * time.Minute }
func (e *Engine) preDur() time.Duration   { return time.Duration(e.settings.PreBreakWarningSec) * time.Second }

func (e *Engine) shouldLong() bool {
	return e.settings.EnableLongBreaks && e.breaksDone >= e.settings.LongBreakInterval
}

// ---- transitions (caller holds e.mu) ----

func (e *Engine) setPhase(p Phase, dur time.Duration) {
	e.phase = p
	e.total = dur
	e.phaseEnd = time.Now().Add(dur)
	e.paused = false
	e.pausedRemaining = 0
}

func (e *Engine) startFocus(now time.Time) {
	e.setPhase(PhaseFocusing, e.focusDur())
	e.hideNotice()
	e.hideOverlays()
	e.emit()
}

func (e *Engine) startPreBreak() {
	dur := e.preDur()
	e.setPhase(PhasePreBreak, dur)
	e.showNotice()
	if e.settings.SoundEnabled {
		platform.PlaySound(soundPreBreak)
	}
	e.emit()
}

func (e *Engine) startBreak() {
	long := e.shouldLong()
	dur := e.shortDur()
	phase := PhaseShortBreak
	if long {
		dur = e.longDur()
		phase = PhaseLongBreak
	}
	e.setPhase(phase, dur)
	e.hideNotice()
	e.showOverlays()
	e.emit()
}

func (e *Engine) endBreak() {
	if e.phase == PhaseLongBreak {
		e.breaksDone = 0
	} else if e.phase == PhaseShortBreak {
		e.breaksDone++
	}
	if e.settings.SoundEnabled {
		platform.PlaySound(soundBreakEnd)
	}
	e.startFocus(time.Now())
}

// tick advances the state machine by one second. Caller holds e.mu.
func (e *Engine) tick(now time.Time) {
	if e.paused {
		e.emit()
		return
	}

	// Detect system sleep: a large gap between ticks means the machine was
	// suspended. Reset rather than firing a break that became overdue while
	// asleep.
	gap := now.Sub(e.lastTick)
	e.lastTick = now
	if gap > sleepGap {
		if e.phase.IsBreak() {
			e.endBreak()
		} else {
			e.startFocus(now)
		}
		return
	}

	// Idle handling only affects the focus period: while the user is away the
	// focus countdown is held at full so they get a fresh period on return.
	if e.phase == PhaseFocusing && e.idle {
		e.phaseEnd = now.Add(e.focusDur())
		e.emit()
		return
	}

	if e.phaseEnd.IsZero() {
		return
	}

	remaining := e.phaseEnd.Sub(now)
	switch e.phase {
	case PhaseFocusing:
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

// loop is the per-second driver.
func (e *Engine) loop() {
	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()
	for {
		select {
		case <-e.stopCh:
			return
		case now := <-ticker.C:
			e.mu.Lock()
			e.tick(now)
			e.mu.Unlock()
		}
	}
}

// idleLoop periodically samples system idle time and, when the user has been
// inactive past the threshold, marks the engine idle (which holds/resets the
// focus timer). Activity clears the flag and starts a fresh focus period.
func (e *Engine) idleLoop() {
	ticker := time.NewTicker(idleCheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-e.stopCh:
			return
		case <-ticker.C:
			idle := time.Duration(platform.IdleSeconds()) * time.Second
			e.mu.Lock()
			threshold := time.Duration(e.settings.IdleThresholdMin) * time.Minute
			wasIdle := e.idle
			if idle >= threshold {
				e.idle = true
				// Only idle-pause the focus workflow; never interrupt a break,
				// and leave an explicit user pause alone.
				if e.phase != PhaseIdle && !e.phase.IsBreak() && e.phase != PhasePaused {
					e.phase = PhaseIdle
					e.hideNotice()
					e.hideOverlays()
					e.emit()
				}
			} else {
				e.idle = false
				if wasIdle && e.phase == PhaseIdle {
					e.startFocus(time.Now())
				}
			}
			e.mu.Unlock()
		}
	}
}

// ---- public controls (frontend-callable via the service) ----

// GetState returns a snapshot of the current engine state.
func (e *Engine) GetState() State {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.state()
}

// GetSettings returns the current settings.
func (e *Engine) GetSettings() config.Settings {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.settings
}

// ApplySettings updates the running engine's settings. A change to the focus
// duration takes effect immediately by restarting the current focus period;
// an in-progress break is left to finish naturally.
func (e *Engine) ApplySettings(s config.Settings) {
	e.mu.Lock()
	e.settings = s
	if e.phase == PhaseFocusing || e.phase == PhaseIdle || e.phase == PhasePreBreak {
		e.startFocus(time.Now())
	}
	e.emit()
	e.mu.Unlock()
}

// StartBreakNow forces a break to begin immediately, skipping the pre-break
// warning. If a break is already in progress this is a no-op.
func (e *Engine) StartBreakNow() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.phase.IsBreak() {
		return
	}
	e.startBreak()
}

// SkipBreak ends the current break (or pre-break warning) and returns to a
// fresh focus period. No-op when not on a break.
func (e *Engine) SkipBreak() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.phase.IsBreak() || e.phase == PhasePreBreak {
		e.endBreak()
	}
}

// PostponeBreak cancels the current or upcoming break and restarts the focus
// period at full duration, effectively delaying the next break.
func (e *Engine) PostponeBreak() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.phase.IsBreak() || e.phase == PhasePreBreak {
		e.endBreak()
	} else {
		e.startFocus(time.Now())
	}
}

// Pause freezes the countdown. The phase is recorded so Resume can restore it.
func (e *Engine) Pause() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.paused {
		return
	}
	if e.phase == PhaseIdle {
		// Already effectively paused by inactivity.
		return
	}
	remaining := e.phaseEnd.Sub(time.Now())
	if remaining < 0 {
		remaining = 0
	}
	e.pausedRemaining = remaining
	e.paused = true
	e.emit()
}

// Resume continues a paused countdown from where it froze.
func (e *Engine) Resume() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.paused {
		return
	}
	e.paused = false
	e.phaseEnd = time.Now().Add(e.pausedRemaining)
	e.pausedRemaining = 0
	e.emit()
}

// Reset clears the long-break counter and starts a fresh focus period.
func (e *Engine) Reset() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.breaksDone = 0
	e.startFocus(time.Now())
}
