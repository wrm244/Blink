// Package platform exposes the small set of host-OS behaviours that the break
// engine needs but that Wails does not wrap: how long since the user last
// touched the keyboard/mouse, and playing a short system sound.
//
// The concrete implementations are split by build constraint (see
// idle_darwin.go / sound_darwin.go and their *_other.go stubs). On platforms
// without a native API the behaviour degrades gracefully: IdleSeconds returns
// 0 (treated as "always active") and PlaySound is a no-op.
package platform
