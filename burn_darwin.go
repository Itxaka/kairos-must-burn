//go:build darwin

package main

import (
	"fmt"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

func reallyBurn(isoPath, devicePath string, totalSize int64, progress *gtk.ProgressBar, status *gtk.Label) error {
	// Open ISO file for reading
	isoFile, err := os.Open(isoPath)
	if err != nil {
		return fmt.Errorf("failed to open ISO file: %w", err)
	}
	defer isoFile.Close()

	// Open device file for writing
	// Write to rdisk. disk goes through the OS cache, rdisk writes directly to the device, its more like a raw block device
	// This speeds up the process significantly
	devicePath = strings.Replace(devicePath, "disk", "rdisk", 1)
	deviceFile, err := os.OpenFile(devicePath, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("failed to open device %s: %w", devicePath, err)
	}
	defer deviceFile.Close()

	// Copy with progress tracking
	return copyWithProgress(isoFile, deviceFile, totalSize, progress, status)
}

func Sync() {
	syscall.Sync()
}

func FormatDriveGPT(deviceID string) error {
	// Unmount the disk first
	unmountCmd := exec.Command("diskutil", "unmountDisk", deviceID)
	if output, err := unmountCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to unmount disk %s: %v, output: %s", deviceID, err, string(output))
	}

	// Erase the disk and create a GPT partition map with no partitions (just free space)
	partitionCmd := exec.Command("diskutil", "partitionDisk", deviceID, "GPT", "Free Space", "%noformat%", "100%")
	if output, err := partitionCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to partition disk %s as GPT: %v, output: %s", deviceID, err, string(output))
	}

	return nil
}
