//go:build !linux

package securemem

func LockProcessMemory() error { return nil }
