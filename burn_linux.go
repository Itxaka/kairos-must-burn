//go:build linux

package main

import (
	"fmt"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"os"
)

func reallyBurn(isoPath, devicePath string, totalSize int64, progress *gtk.ProgressBar, status *gtk.Label) error {
	// Open ISO file for reading
	isoFile, err := os.Open(isoPath)
	if err != nil {
		return fmt.Errorf("failed to open ISO file: %w", err)
	}
	defer isoFile.Close()

	// Open device file for writing
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
	panic("Implement!")
	return nil
}
