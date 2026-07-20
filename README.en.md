<p align="center">
  <img src="docs/images/logo.png" width="160" alt="Blink Logo" />
</p>

<h1 align="center">Blink</h1>
<p align="center">
  <strong>Smart Eye-Care Break Reminders · Make the 20-20-20 Rule Stick</strong>
</p>

<p align="center">
  English · <a href="./README.md">简体中文</a>
</p>

---

<p align="center">
  <img src="docs/images/设置界面.png" width="720" alt="Blink Settings" />
</p>

## Why Blink?

Staring at screens for hours keeps your ciliary muscles contracted — a leading cause of eye strain, dry eyes, and worsening vision. Ophthalmologists recommend the **[20-20-20 rule](https://www.aoa.org/healthy-eyes/caring-for-your-eyes/protecting-your-eyes/20-20-20-rule)**:

> **Every 20 minutes → look ~20 feet away → for 20 seconds**

Simple in theory, but nearly impossible to remember consistently.

Blink automates it. It lives in your menu bar, counts quietly, and gently reminds you to look up when the time comes. You just follow along — Blink handles the rest.

## Key Features

| Feature | Details |
| --- | --- |
| 🧭 **First-run onboarding** | Setup screen on first launch — no surprise pop-ups before you're ready |
| ⏱️ **Live menu-bar countdown** | See exactly how long until your next break at a glance |
| 🔄 **Focus → Pre-break → Break** cycle | Frequent short breaks + occasional longer ones, natural rhythm |
| 🖥️ **Full-screen break overlay** | Covers all displays (multi-monitor aware), large countdown + chime |
| 🔔 **Pre-break notice** | Heads-up before screen dims so you can wrap up |
| 😴 **Idle-aware** | Pauses when you step away, resumes on return; survives system sleep |
| ⌨️ **Global keyboard shortcuts** | Start / skip / postpone / preferences without touching the mouse |
| 🎨 **Fully configurable** | Timing, frequency, shortcuts, appearance, language — your call |
| 🌐 **Bilingual UI** | Chinese / English; defaults to system language, switchable anytime |

## Screenshots

### Full-Screen Break Countdown

<p align="center">
  <img src="docs/images/休息全屏倒计时.png" width="640" alt="Full-screen break countdown" />
</p>

Immersive dark full-screen overlay with a centered countdown, eye-care tip, and skip button. A gentle chime plays when the break ends.

### Pre-Break Notice

<p align="center">
  <img src="docs/images/休息弹窗.png" width="560" alt="Pre-break notice" />
</p>

A glass-morphic notification card that appears before the screen dims — tells you how many seconds left and lets you postpone or wait.

### Menu Bar

<p align="center">
  <img src="docs/images/菜单.png" width="320" alt="Menu bar dropdown" />
</p>

Click the tray icon to: take a break now, skip, postpone, pause/resume, reset cycle, open preferences, or quit. The header shows current status and next break time.

### Settings Panel

<p align="center">
  <img src="docs/images/设置界面.png" width="720" alt="Settings panel" />
</p>

Tabbed settings: timing (presets + custom sliders), options, shortcuts, about. Includes Pomodoro, high-efficiency rhythm, and other built-in presets.

## Quick Start

### Install from DMG (Recommended)

1. Download the latest `Blink-<version>-arm64.dmg` (Apple Silicon) or `-amd64.dmg` (Intel) from [Releases](../../releases)
2. Open the `.dmg`, drag **Blink** into **Applications**
3. In Finder, right-click Blink → **Open** → confirm **Open** (first time only)

> ⚠️ The app is ad-hoc signed (not Apple Developer ID), so the first launch requires right-click → Open to bypass Gatekeeper. After that, double-click works normally.

### Build from Source

```bash
# Clone
git clone https://github.com/<your-org>/PocketMind.git
cd PocketMind

# Install frontend deps
pnpm install

# Dev mode (hot-reload)
wails3 dev -config ./build/config.yml

# Production build
wails3 build                    # -> bin/blink
wails3 task package             # -> bin/blink.app (ad-hoc signed)
wails3 task darwin:dmg          # -> bin/Blink-<version>-<arch>.dmg
```

### System Requirements

- **macOS 12.0 Monterey** or later
- Apple Silicon (M1/M2/M3/M4) or Intel Mac

## Tech Stack

| Layer | Technology |
| --- | --- |
| Desktop Framework | [Wails v3](https://v3.wails.io) |
| Backend | Go |
| Frontend | Vue 3 + TypeScript |
| Styling | Tailwind CSS v4 |
| i18n | vue-i18n |
| Design | Slate design system — single accent, no gradients, calm & unobtrusive |

## Architecture Overview

```
├── main.go                 # Entry point & Wails app init
├── tray.go                 # Menu-bar tray + dropdown (i18n)
├── shortcuts.go            # Global hotkey registration
├── prefs.go                # Preferences window management
├── breakservice.go         # Wails Service for frontend calls (onboarding gate + engine control)
│
├── internal/
│   ├── breakengine/        # Timer state machine (phases, idle detection, sleep recovery)
│   ├── config/             # Settings struct + JSON persistence
│   └── platform/           # CGo: macOS idle time + NSSound chime
│
└── frontend/src/
    ├── views/              # Settings / BreakOverlay / PreBreakNotice
    ├── components/         # Glass-panel component library
    └── locales/            # EN / ZH language packs
```

## Keyboard Shortcuts

| Shortcut | Action |
| --- | --- |
| `Cmd+Shift+B` | Take a break now |
| `Cmd+Shift+,` | Open preferences |
| `Cmd+Shift+/` | Skip current break |
| `Cmd+Shift+.` | Postpone break |

> Customizable in Settings.

## Roadmap

- [ ] Meeting/call detection — auto-pause during video conferences
- [ ] Media playback detection — auto-pause during video watching
- [ ] Statistics dashboard — daily/weekly break completion rate
- [ ] Windows / Linux support
- [ ] Apple Developer ID signing + notarization

## Contributing

Issues and PRs are welcome! To get started:

```bash
pnpm install        # Install frontend dependencies
wails3 dev          # Start dev server with hot-reload
```

## License

[MIT](./LICENSE)

© 2026 Blink. Free and open source.
