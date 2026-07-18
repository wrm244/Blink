# PocketMind

A macOS menu-bar app that reminds you to rest your eyes, modelled on [LookAway](https://lookaway.app) and the 20-20-20 rule: every 20 minutes, look at something ~20 feet away for 20 seconds.

Built with [Wails v3](https://v3.wails.io) (Go + Vue 3 + TypeScript), [Tailwind CSS v4](https://tailwindcss.com) + [Reka UI](https://reka-ui.com) (shadcn-vue style components), and [vue-i18n](https://vue-i18n.intlify.dev) for Chinese / English.

## Features

- **First-run onboarding**: the app opens to a setup screen and does **not** start counting down until you finish configuration and press "Start focusing".
- **Menu-bar tray** with a live countdown of the time to your next break.
- **Focus → pre-break warning → break** cycle, with frequent short breaks and occasional longer breaks.
- **Break overlay** that gently blurs every screen (multi-monitor aware) and shows a ring countdown; a chime plays when the break ends.
- **Pre-break notice** so you can wrap up your task before the screen dims.
- **Idle-aware**: the focus timer pauses and resets when you've been away, so breaks only trigger while you're actually working. Survives system sleep.
- **Global keyboard shortcuts** for start / skip / postpone / preferences.
- **Fully configurable** via a modern settings UI (timing sliders, options, shortcuts, about) and persisted to `~/Library/Application Support/PocketMind/settings.json`.
- **Bilingual UI** (简体中文 / English): defaults to the system language, switchable in settings; the native tray menu localises too.

## Getting started

```bash
wails3 dev      # hot-reload development
wails3 build    # production binary -> bin/pocketmind
wails3 task package   # -> bin/pocketmind.app (codesign ad-hoc)
```

On first launch the setup window appears. Pick a language and press **Start focusing** (or tweak timings first). The tray icon then shows a live countdown. Open the menu-bar item (or press `Cmd+Shift+,`) for Preferences, `Cmd+Shift+B` to take a break now.

## Architecture

| Package | Responsibility |
| --- | --- |
| `internal/breakengine` | The timer state machine (phases, idle, sleep-reset, break windows, start/stop) |
| `internal/config` | Settings struct + JSON persistence (incl. `onboarded`, `autoStart`, `language`) |
| `internal/platform` | CGo: system idle time (`CGEventSource`) and `NSSound` chime on macOS; no-op stubs elsewhere |
| `main.go` / `tray.go` / `shortcuts.go` / `prefs.go` / `breakservice.go` | App wiring: tray+menu (i18n), global shortcuts, preferences window, the Wails service the frontend calls (onboarding gate + engine controls) |
| `frontend/src` | Vue 3 SPA - a hash router selects Settings / BreakOverlay / PreBreakNotice per window; shadcn-vue components in `components/ui`, locales in `locales/` |

> Note: conference/call detection and media-playback detection (LookAway's auto-pause features) are out of scope for this initial cut.
