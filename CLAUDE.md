# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Blink is a macOS menu-bar app that enforces the 20-20-20 eye-strain rule: every 20 minutes of screen time, look 20 feet away for 20 seconds. It runs as a menu-bar accessory (no Dock icon) with a state machine cycling through focus → pre-break warning → short/long break → repeat. Built on **Wails v3** (Go backend + Vue 3 frontend in a WebView).

## Build / Run

```bash
pnpm install                          # frontend dependencies (run in repo root)
wails3 dev -config ./build/config.yml # dev mode with hot-reload (Go + Vite)
wails3 build                          # production binary → bin/blink
wails3 task package                   # .app bundle (ad-hoc signed)
wails3 task darwin:dmg                # distributable DMG
```

- Node package manager is **pnpm** (lockfile: `pnpm-lock.yaml`).
- The project-level `Taskfile.yml` delegates to per-platform Taskfiles under `build/`.
- Vite dev server port: `9245` (set via `WAILS_VITE_PORT` env).
- Go unit tests cover the core engine (`internal/breakengine`) and stats store (`internal/stats`): run `go test ./internal/...` (the `test` Taskfile task adds `-race`). Frontend uses `vue-tsc` for type-checking only. CI must run the Go suite with `-race` — concurrent-engine bugs are only caught there.

## Architecture

### Go backend (root and `internal/`)

The entry point is `main.go`, which initialises four things at startup:
1. Loads `config.Settings` from JSON (defaults if missing).
2. Creates the `breakengine.Engine` — the timer state machine.
3. Registers a Wails `BreakService` that exposes the engine's API to the frontend.
4. Builds the macOS tray menu and global shortcuts.

**`BreakService`** (`breakservice.go`) is a thin Wails Service facade. Every method delegates to `engine.*`. The frontend calls these via auto-generated TypeScript bindings (`frontend/bindings/`). Key flow: `SaveSettings` persists to disk, applies to the engine, and rebinds shortcuts — all on the main thread, so the shortcut rebind is dispatched to a goroutine to avoid deadlock.

**`breakengine.Engine`** (`internal/breakengine/`) is the core state machine. Phases:
```
PhaseFocusing → PhasePreBreak (heads-up) → PhaseShortBreak / PhaseLongBreak → PhaseFocusing...
```
- `loop()` ticks every second, advances the phase when timers expire, and emits `blink:tick` events (a `State` struct) to the frontend.
- `idleLoop()` polls `platform.IdleSeconds()` (CGo → CGEventSource) every 5s. When idle exceeds the threshold, the phase becomes `PhaseIdle` (focus timer held at full); activity resets to `PhaseFocusing`.
- A separate 1s poll (`externalSuspendLoop`) runs `checkExternalSuspend()`: when a meeting (mic input active) or media playback (audio output active) is detected, the countdown is frozen with `autoPaused` (edge-triggered, so a manual Resume is not immediately re-paused); the timer resumes automatically once the condition clears. Auto-pause skips breaks/idle phases, hides the notice/overlay while active, and re-shows the notice if the pre-break phase resumes. `Start()` resets the detection flags so a restart during an active meeting/media re-triggers the pause.
- System sleep is detected by a >5s gap between ticks; the focus period resets rather than firing a stale break.
- **Window ops are funneled through a dedicated goroutine** (`windowLoop`) via a buffered channel (`cmdCh`). The engine mutex is never held while calling Wails window APIs (which marshal to the main thread internally — holding the mutex across that would deadlock against binding calls).

**`config.Settings`** (`internal/config/settings.go`) is the single settings struct, persisted as JSON to `~/Library/Application Support/Blink/settings.json`. `Load()` fills missing fields with defaults so old files survive new-field additions. `Save()` creates the directory if needed.

**`internal/platform/`** uses CGo and build constraints (`//go:build darwin`) for these macOS-specific operations:
- `IdleSeconds()` — HID idle time via `CGEventSourceSecondsSinceLastEventType`.
- `AudioActivity()` — one pass over the audio process list (own PID excluded) returning whether any process has an active input (mic) or output stream, via `kAudioHardwarePropertyProcessObjectList` + `kAudioProcessPropertyIsRunningInput/Output`. Used for meeting/media auto-pause.
- `PlaySound()` — plays named system sounds via `NSSound`.
- `SetDockVisible()` / `Activate()` — toggles `NSApp.activationPolicy` between Regular (Dock shown, settings open) and Accessory (menu-bar-only).

Non-darwin stubs (`*_other.go`) degrade gracefully: `IdleSeconds` returns 0, `PlaySound` is a no-op, `AudioActivity` returns false, false.

### Main-thread deadlock discipline (important)

Wails window APIs (`Show`, `Hide`, `Focus`, `NewWithOptions`) internally call `InvokeSync`, which dispatches to the main thread and blocks. If called **from** the main thread (e.g., a tray click handler or a Wails binding call), this deadlocks. The codebase consistently handles this:
- Tray click handlers, menu callbacks, and window event listeners dispatch work to a **goroutine** before touching windows.
- `showPreferences()` creates the window on first open (via `NewWithOptions`, which shows itself) but only calls `Show()`/`Focus()` on the **existing-window** path (safe because that path is only reached from goroutine callers).
- The break engine's `windowLoop` goroutine is the sole owner of overlay/notice window lifecycle — the state machine only sends commands via the channel.

### Menu bar / tray (`tray.go`)

- `buildTray()` creates the NSStatusItem menu with action items. Left-click falls through to the native menu tracking (`systrayPreClickCallback` returns 1 when no click handler is registered).
- `trayStatusLoop()` reads engine state every second and updates the tray label (countdown) and tooltip, skipping redraws when text hasn't changed.
- Tray menu strings are in Go (not the webview i18n). Locale is tracked in `menuLang` (`atomic.Value`) and set by `setMenuLanguage()` when settings change.

### Shortcuts (`shortcuts.go`)

`registerAll()` unbinds all existing shortcuts and rebinds from settings. Protected by `rebindMu` to serialise rapid re-registrations. Safe to call from any goroutine — before `app.Run()` bindings are deferred; after, they marshal to the main thread.

### Frontend (`frontend/src/`)

A Vue 3 SPA that uses **hash-based view switching** — the Go side opens windows at different URLs (`/#settings`, `/#break`, `/#notice`), and `App.vue` reads `window.location.hash` once on mount to select the component.

| Hash | Component | Purpose |
|------|-----------|---------|
| (default) | `Settings.vue` | Full settings panel (time presets, sliders, shortcuts, about) |
| `#break` | `BreakOverlay.vue` | Full-screen dark overlay with countdown during breaks |
| `#notice` | `PreBreakNotice.vue` | Glassmorphism heads-up card before a break starts |

**State flow:** The Go engine emits `blink:tick` events → Wails runtime delivers them to the frontend → components read `state.phase` to decide visibility and `state.remainingSec` / `state.totalSec` for the countdown. User actions (skip, postpone, start break) call back to Go through Wails bindings generated from `BreakService`.

**Styling:** Tailwind CSS v4 with a custom "Slate" palette (monochrome, no gradients). The theme system (`theme.ts`) toggles a `dark` class on `<html>` and follows OS preference changes in real time when set to "system". Theme and locale are cached in `localStorage` (keys `pm:theme`, `pm:locale`) for early application before the full settings load.

**i18n:** `vue-i18n` with `zh-CN` and `en` locales. System locale detection runs once at startup; saved preference overrides it. The tray menu has separate Go-side locale strings.

**Component library:** A set of glassmorphism-styled primitives: `GlassPanel`, `GButton`, `GSlider`, `GToggle`, `GSegmented`, `GTimeField`.

## Key patterns

- **Settings save flow:** Frontend calls `BreakService.SaveSettings(settings)` → Go persists JSON → applies to engine → goroutine rebinds shortcuts and tray menu labels.
- **Onboarding gate:** `settings.Onboarded` is `false` by default. `ServiceStartup` opens the settings window without starting the engine. `CompleteOnboarding()` sets `Onboarded = true`, saves, and calls `engine.Start()`.
- **Multi-monitor breaks:** `ensureOverlays()` creates one full-screen `WebviewWindow` per display (at `MacWindowLevelScreenSaver` to cover Dock and menu bar). Windows are reused when the display configuration hasn't changed; recreated on display connect/disconnect. When a break ends the overlays are **closed (destroyed), not hidden** — keeping the WKWebView alive would leave its WebKit renderer process resident in memory. The next break recreates them.
- **Pause vs Idle:** Pause is user-initiated, freezes the countdown in-place with `e.paused = true` (phase unchanged). Idle is auto-detected, sets phase to `PhaseIdle` and resets to a fresh focus period on resume.
- **`withDefaults()` pattern:** `config.Settings.withDefaults()` fills zero-value fields from `Default()` so adding new fields to the struct doesn't break existing saved files.
