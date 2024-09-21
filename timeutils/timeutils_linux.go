//go:build linux
// +build linux

package timeutils

import (
	"syscall"
	"time"
)

// SetSystemTime sets the system time on Linux using syscalls.
func SetSystemTime(t time.Time) error {
	tv := syscall.Timeval{
		Sec:  t.Unix(),
		Usec: int64(t.Nanosecond() / 1000),
	}
	return syscall.Settimeofday(&tv)
}
