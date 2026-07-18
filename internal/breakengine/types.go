package breakengine

// Phase is the high-level state the engine is currently in. It is emitted to
// the frontend as a string so the UI can switch views without parsing numbers.
type Phase string

const (
	// PhaseFocusing: a work period is counting down toward the next break.
	PhaseFocusing Phase = "focusing"
	// PhasePreBreak: the heads-up window just before a break starts.
	PhasePreBreak Phase = "prebreak"
	// PhaseShortBreak: a regular short break overlay is showing.
	PhaseShortBreak Phase = "shortbreak"
	// PhaseLongBreak: an occasional longer break overlay is showing.
	PhaseLongBreak Phase = "longbreak"
	// PhaseIdle: the user has been inactive past the idle threshold; the focus
	// timer is held and will reset when activity resumes.
	//
	// Note: there is no PhasePaused constant. Pause() freezes the countdown by
	// setting the live boolean e.paused without changing the phase, so the
	// phase value during a user-pause is whatever it was when Pause was called
	// (typically PhaseFocusing). The frontend reads the `paused` boolean in
	// the emitted State, not the phase string, to detect pause.
	PhaseIdle Phase = "idle"
)

// IsBreak reports whether the phase is an active break overlay (short or long).
func (p Phase) IsBreak() bool {
	return p == PhaseShortBreak || p == PhaseLongBreak
}

// State is the snapshot emitted to the frontend on every tick and phase change.
// Durations are expressed in whole seconds for easy formatting in the UI.
type State struct {
	Phase           Phase `json:"phase"`
	RemainingSec    int   `json:"remainingSec"`
	TotalSec        int   `json:"totalSec"`
	BreaksUntilLong int   `json:"breaksUntilLong"`
	ShortBreakCount int   `json:"shortBreakCount"`
	Idle            bool  `json:"idle"`
	Paused          bool  `json:"paused"`
}
