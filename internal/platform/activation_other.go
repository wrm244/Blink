//go:build !darwin

package platform

// SetDockVisible is a no-op off macOS: other platforms always show a taskbar
// entry for a visible window, so there is no activation policy to toggle.
func SetDockVisible(visible bool) {}

// Activate is a no-op off macOS.
func Activate() {}
