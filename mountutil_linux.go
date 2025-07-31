//go:build linux

package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"golang.org/x/sys/unix"
)

// IsDeviceMounted checks if any partition of the given device (e.g. /dev/sdb) is mounted.
// Returns a slice of mounted partition paths (e.g. /dev/sdb1, /dev/sdb2, ...)
func IsDeviceMounted(device string) ([]string, error) {
	file, err := os.Open("/proc/mounts")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var mounted []string
	prefix := device
	if !strings.HasSuffix(prefix, "/") {
		prefix += ""
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) > 0 && strings.HasPrefix(fields[0], device) && fields[0] != device {
			mounted = append(mounted, fields[0])
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return mounted, nil
}

// UnmountDevice tries to unmount all given partitions.
func UnmountDevice(partitions []string) error {
	for _, part := range partitions {
		if err := unmount(part); err != nil {
			return fmt.Errorf("failed to unmount %s: %w", part, err)
		}
	}

	return nil
}

// unmount tries to unmount a partition in a cross-platform way.
func unmount(partition string) error {
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" || runtime.GOOS == "freebsd" {
		// Try syscall first
		err := unix.Unmount(partition, 0)
		if err == nil {
			return nil
		}
		// Fallback to umount command
		cmd := exec.Command("umount", partition)
		return cmd.Run()
	}
	if runtime.GOOS == "windows" {
		// On Windows, try mountvol to remove the drive letter
		cmd := exec.Command("mountvol", partition, "/p")
		return cmd.Run()
	}
	return fmt.Errorf("unmount not supported on %s", runtime.GOOS)
}
