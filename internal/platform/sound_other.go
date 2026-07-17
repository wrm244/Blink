//go:build !darwin

package platform

// PlaySound is a no-op on platforms without a native sound API.
func PlaySound(name string) {}
