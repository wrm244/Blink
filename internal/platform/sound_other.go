//go:build !darwin && !windows

package platform

// PlaySound 在没有原生音效 API 的平台上为空操作。
func PlaySound(name string) {}
