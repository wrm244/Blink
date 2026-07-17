package main

import (
	"context"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"

	"pocketmind/internal/breakengine"
	"pocketmind/internal/config"
)

// BreakService is the Go type whose methods the frontend calls through Wails
// bindings. It is a thin facade over the break engine: the engine owns the
// timing state machine, this service just forwards user actions and settings.
type BreakService struct {
	engine *breakengine.Engine
}

func NewBreakService(engine *breakengine.Engine) *BreakService {
	return &BreakService{engine: engine}
}

// ServiceName is used by Wails for logging.
func (s *BreakService) ServiceName() string { return "BreakService" }

// ServiceStartup is called by Wails once the application has started. The
// engine only begins counting down if the user has already completed onboarding
// (and AutoStart is on); otherwise the app waits for CompleteOnboarding().
func (s *BreakService) ServiceStartup(_ context.Context, _ application.ServiceOptions) error {
	settings := s.engine.GetSettings()
	if settings.Onboarded && settings.AutoStart {
		s.engine.Start()
	} else if !settings.Onboarded {
		// First run: surface the setup window so the user can configure before
		// any countdown begins.
		showPreferences()
	}
	return nil
}

// GetState returns a snapshot of the engine for the UI to render.
func (s *BreakService) GetState() breakengine.State { return s.engine.GetState() }

// GetSettings returns the current settings.
func (s *BreakService) GetSettings() config.Settings { return s.engine.GetSettings() }

// SaveSettings persists the settings, applies them to the running engine and
// re-binds the global shortcuts. The shortcut rebind runs on its own goroutine
// because binding marshals to the main thread and this method itself runs on
// the main thread (it is a Wails binding call).
func (s *BreakService) SaveSettings(settings config.Settings) error {
	if err := config.Save(settings); err != nil {
		return err
	}
	s.engine.ApplySettings(settings)
	go func() {
		registerAll(settings)
		if settings.Language != "" {
			if err := setMenuLanguage(settings.Language); err != nil {
				log.Printf("pocketmind: set menu language: %v", err)
			}
		}
	}()
	return nil
}

// CompleteOnboarding marks onboarding done and starts the engine for the first
// focus cycle. Called once the user finishes the first-run setup screen.
func (s *BreakService) CompleteOnboarding() error {
	settings := s.engine.GetSettings()
	settings.Onboarded = true
	if err := config.Save(settings); err != nil {
		return err
	}
	s.engine.ApplySettings(settings)
	go registerAll(settings)
	if settings.Language != "" {
		go setMenuLanguage(settings.Language)
	}
	s.engine.Start()
	return nil
}

// StartEngine starts the countdown (used after a manual stop or pause).
func (s *BreakService) StartEngine() { s.engine.Start() }

// StopEngine halts the countdown entirely.
func (s *BreakService) StopEngine() { s.engine.Stop() }

// StartBreakNow forces a break immediately, skipping the pre-break warning.
func (s *BreakService) StartBreakNow() { s.engine.StartBreakNow() }

// SkipBreak ends the current break or pre-break warning.
func (s *BreakService) SkipBreak() { s.engine.SkipBreak() }

// PostponeBreak cancels the current/upcoming break and restarts the focus period.
func (s *BreakService) PostponeBreak() { s.engine.PostponeBreak() }

// Pause freezes the countdown.
func (s *BreakService) Pause() { s.engine.Pause() }

// Resume continues a paused countdown.
func (s *BreakService) Resume() { s.engine.Resume() }

// Reset clears the long-break counter and restarts the focus period.
func (s *BreakService) Reset() { s.engine.Reset() }
