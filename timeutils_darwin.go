//go:build darwin
// +build darwin

package main

import (
	"syscall"
	"time"
)

// setSystemTime sets the system time on macOS using the Darwin syscall.
func setSystemTime(t time.Time) error {
	utc := t.UTC()
	tv := syscall.Timeval{
		Sec:  utc.Unix(),
		Usec: int32(utc.Nanosecond() / 1000), // Ensure this is int32
	}

	return syscall.Settimeofday(&tv)
}
