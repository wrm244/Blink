# PocketMind

A macOS menu-bar app that reminds you to rest your eyes, modelled on [LookAway](https://lookaway.app) and the 20-20-20 rule: every 20 minutes, look at something ~20 feet away for 20 seconds.

Built with [Wails v3](https://v3.wails.io) (Go + Vue 3 + TypeScript).

## Features

- **Menu-bar tray** with a live countdown of the time to your next break.
- **Focus → pre-break warning → break** cycle, with frequent short breaks and occasional longer breaks.
- **Break overlay** that gently blurs every screen (multi-monitor aware) and shows a soft countdown; a chime plays when the break ends.
- **Pre-break notice** so you can wrap up your task before the screen dims.
- **Idle-aware**: the focus timer pauses and resets when you've been away, so breaks only trigger while you're actually working. Survives system sleep.
- **Global keyboard shortcuts** for start / skip / postpone / preferences.
- **Fully configurable** timings, long-break cadence, pre-break warning, idle threshold, sounds and shortcuts — saved to `~/Library/Application Support/PocketMind/settings.json`.

## Getting started

```bash
wails3 dev      # hot-reload development
wails3 build    # production binary → bin/pocketmind
wails3 task package   # → bin/pocketmind.app (codesign ad-hoc)
```

Open the menu-bar item (or press `Cmd+Shift+,`) for Preferences, or `Cmd+Shift+B` to take a break now.

## Architecture

| Package | Responsibility |
| --- | --- |
| `internal/breakengine` | The timer state machine (phases, idle, sleep-reset, break windows) |
| `internal/config` | Settings struct + JSON persistence |
| `internal/platform` | CGo: system idle time (`CGEventSource`) and `NSSound` chime on macOS; no-op stubs elsewhere |
| `main.go` / `tray.go` / `shortcuts.go` / `prefs.go` / `breakservice.go` | App wiring: tray+menu, global shortcuts, preferences window, the Wails service the frontend calls |
| `frontend/src` | Vue 3 SPA — a hash router selects Settings / BreakOverlay / PreBreakNotice per window |

> Note: conference/call detection and media-playback detection (LookAway's auto-pause features) are out of scope for this initial cut.
