//go:build linux
// +build linux

package main

import (
	"syscall"
	"time"
)

// setSystemTime sets the system time on Linux using syscalls.
func setSystemTime(t time.Time) error {
	tv := syscall.Timeval{
		Sec:  t.Unix(),
		Usec: int64(t.Nanosecond() / 1000), // Change this line to use int64
	}
	return syscall.Settimeofday(&tv)
}
