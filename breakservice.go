package main

import (
	"context"

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

// ServiceStartup is called by Wails once the application has started; this is
// the earliest point at which application.Get() is valid, so the engine is
// launched here rather than in main.
func (s *BreakService) ServiceStartup(_ context.Context, _ application.ServiceOptions) error {
	s.engine.Start()
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
	go registerAll(settings)
	return nil
}

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
