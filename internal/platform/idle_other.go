//go:build !darwin

package platform

func IdleSeconds() float64 { return 0 }
