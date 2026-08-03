//go:build !darwin && !windows

package platform

// IdleSeconds 在没有原生空闲查询 API 的平台上返回 0（视为始终活跃）。
func IdleSeconds() float64 { return 0 }
