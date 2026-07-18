# Blink

[English](./README.md) | [简体中文](./README.zh-CN.md)

A macOS menu-bar app that reminds you to rest your eyes, modelled on the [20-20-20 rule](https://www.aoa.org/healthy-eyes/caring-for-your-eyes/protecting-your-eyes/20-20-20-rule): every 20 minutes, look at something ~20 feet away for 20 seconds.

Built with [Wails v3](https://v3.wails.io) (Go + Vue 3 + TypeScript), [Tailwind CSS v4](https://tailwindcss.com), and [vue-i18n](https://vue-i18n.intlify.dev) for Chinese / English.

## Features

- **First-run onboarding** — the app opens to a setup screen and does **not** start counting down until you finish configuration and press "Start focusing".
- **Menu-bar tray** with a live countdown of the time to your next break.
- **Focus → pre-break warning → break** cycle, with frequent short breaks and occasional longer breaks.
- **Full-screen break overlay** that covers every display (multi-monitor aware) with a large countdown; a chime plays when the break ends. The overlay sits at window-status level, above everything system-wide.
- **Pre-break notice** so you can wrap up your task before the screen dims.
- **Idle-aware** — the focus timer pauses and resets when you've been away, so breaks only trigger while you're actually working. Survives system sleep.
- **Global keyboard shortcuts** for start / skip / postpone / preferences.
- **Fully configurable** via a settings UI (timing sliders, options, shortcuts, about) and persisted to `~/Library/Application Support/Blink/settings.json`.
- **Bilingual UI** (简体中文 / English): defaults to the system language, switchable in settings; the native tray menu localises too.

## Design

Blink uses a **slate design system** — a single neutral accent, no gradients, desaturated semantic colours. The break overlay is a calm dark slate field with the countdown as the hero element. The goal is sturdy and unobtrusive: the app should fade into your workflow and only speak up when it's time to rest.

## Getting started

```bash
wails3 dev        # hot-reload development
wails3 build      # production binary -> bin/blink
wails3 task package   # -> bin/blink.app (codesign ad-hoc)
```

On first launch the setup window appears. Pick a language and press **Start focusing** (or tweak timings first). The tray icon then shows a live countdown. Open the menu-bar item (or press `Cmd+Shift+,`) for Preferences, `Cmd+Shift+B` to take a break now.

## Architecture

| Package | Responsibility |
| --- | --- |
| `internal/breakengine` | The timer state machine (phases, idle, sleep-reset, break windows, start/stop) |
| `internal/config` | Settings struct + JSON persistence (incl. `onboarded`, `autoStart`, `language`) |
| `internal/platform` | CGo: system idle time (`CGEventSource`) and `NSSound` chime on macOS; no-op stubs elsewhere |
| `main.go` / `tray.go` / `shortcuts.go` / `prefs.go` / `breakservice.go` | App wiring: tray+menu (i18n), global shortcuts, preferences window, the Wails service the frontend calls (onboarding gate + engine controls) |
| `frontend/src` | Vue 3 SPA — a hash router selects Settings / BreakOverlay / PreBreakNotice per window; glass-panel components in `components/`, locales in `locales/` |

> Note: conference/call detection and media-playback detection (auto-pause features) are out of scope for this initial cut.

## License

(c) 2026, Blink
