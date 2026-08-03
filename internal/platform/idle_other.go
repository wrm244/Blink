//go:build !darwin

package platform

// IdleSeconds 在非 macOS 平台上返回 0（视为始终活跃）。
func IdleSeconds() float64 { return 0 }
