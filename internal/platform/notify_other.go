//go:build !windows

package platform

// NotifyEnvChange is a no-op outside Windows.
func NotifyEnvChange() error { return nil }
