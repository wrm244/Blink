# AGENTS.md

Operating guide for AI coding agents in this repository. Detailed architecture and rationale live in [CLAUDE.md](CLAUDE.md) — read it before large changes. Keep this file in sync with CLAUDE.md; do not duplicate its full content here.

## Project

Blink (PocketMind) is a macOS menu-bar app enforcing the 20-20-20 eye-strain rule (20 min screen time → look 20 ft away for 20 s). Built with **Wails v3**: Go backend + Vue 3 frontend in a WebView. Runs as a menu-bar accessory (no Dock icon) with a state machine cycling focus → pre-break warning → short/long break → repeat.

## Commands

```bash
pnpm install                            # frontend dependencies (pnpm only; lockfile: pnpm-lock.yaml)
wails3 dev -config ./build/config.yml   # dev mode with hot-reload (Go + Vite, port 9245)
wails3 build                            # production binary → bin/blink
wails3 task package                     # .app bundle (ad-hoc signed)
wails3 task darwin:dmg                  # distributable DMG
```

- Frontend validation is type-check only: `vue-tsc` (no test/lint scripts in package.json).
- Go unit tests exist for the engine and stats store under `internal/` (`go test ./internal/...`, or `wails3 task test`); CI runs them with `-race` to catch concurrent-engine regressions. Don't claim coverage is exhaustive, but the Go suite must stay green.
- The project-level `Taskfile.yml` delegates to per-platform Taskfiles under `build/`.

## Architecture boundaries

- Entry: `main.go` → loads `config.Settings` → creates `breakengine.Engine` → registers Wails `BreakService` → builds tray menu and global shortcuts.
- `breakservice.go` is a thin Wails Service facade; every method delegates to `engine.*`. Save flows run on the main thread, so shortcut rebinds dispatch to a goroutine.
- `internal/breakengine/` is the core state machine: `loop()` ticks every second and emits `blink:tick` events; `idleLoop()` polls `platform.IdleSeconds()` every 5s; `windowLoop()` is the sole owner of overlay/notice window lifecycle.
- `internal/platform/` uses CGo with `//go:build darwin` constraints; non-darwin stubs degrade gracefully.
- Frontend is a Vue 3 SPA with hash routing: `#settings` → `Settings.vue`, `#break` → `BreakOverlay.vue`, `#notice` → `PreBreakNotice.vue`. Go opens windows at these URLs.
- `frontend/bindings/` is **generated** by Wails — do not hand-edit; regenerate after Go service changes.

## Concurrency discipline (critical)

- **Never call Wails window APIs (`Show`, `Hide`, `Focus`, `NewWithOptions`) from the main thread** — they block on `InvokeSync` and deadlock. Dispatch to a goroutine first.
- The engine mutex must never be held while calling Wails window APIs.
- The state machine only sends commands to `windowLoop` via the buffered channel `cmdCh`; overlay windows are closed (destroyed), not hidden, when breaks end.
- `shortcuts.go` rebinding is serialized by `rebindMu`; safe from any goroutine.

## Risk controls

- Settings persist to `~/Library/Application Support/Blink/settings.json`; running the app writes real user state — never run against a fake config without noting it.
- Break overlays are full-screen at `MacWindowLevelScreenSaver` (cover Dock and menu bar).
- No commits that switch the package manager away from pnpm or rewrite `pnpm-lock.yaml` without approval.

## Traps

- `config.Settings.withDefaults()` fills zero-value fields so old saved files survive new-field additions — always use it when adding settings.
- System sleep detection: a >5s gap between ticks resets the focus period instead of firing a stale break.
