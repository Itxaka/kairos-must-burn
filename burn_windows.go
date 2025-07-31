//go:build windows

package main

import (
	"fmt"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func reallyBurn(isoPath, devicePath string, totalSize int64, progress *gtk.ProgressBar, status *gtk.Label) error {
	// Format device path for Windows (e.g., "\\.\PHYSICALDRIVE1")
	if !strings.HasPrefix(devicePath, "\\\\.\\") {
		// Extract drive number if it's in form like "X:"
		if len(devicePath) == 2 && devicePath[1] == ':' {
			devicePath = fmt.Sprintf("\\\\.\\%s", devicePath)
		} else {
			// Try to use the device path directly
			devicePath = fmt.Sprintf("\\\\.\\%s", devicePath)
		}
	}

	// Open ISO file for reading
	isoFile, err := os.Open(isoPath)
	if err != nil {
		return fmt.Errorf("failed to open ISO file: %w", err)
	}
	defer isoFile.Close()

	// Open device for writing
	deviceFile, err := os.OpenFile(devicePath, os.O_WRONLY, 0)
	if err != nil {
		// Fallback to using PowerShell commands
		return burnWithPowerShell(isoPath, devicePath, totalSize, progress, status)
	}
	defer deviceFile.Close()

	// Copy with progress
	return copyWithProgress(isoFile, deviceFile, totalSize, progress, status)
}

// burnWithPowerShell is a fallback method for Windows when direct access fails
func burnWithPowerShell(isoPath, devicePath string, totalSize int64, progress *gtk.ProgressBar, status *gtk.Label) error {
	// PowerShell command to write ISO to disk
	psCmd := fmt.Sprintf(
		"$bytes = [System.IO.File]::ReadAllBytes('%s'); "+
			"$file = [System.IO.File]::OpenWrite('%s'); "+
			"$file.Write($bytes, 0, $bytes.Length); "+
			"$file.Close();",
		filepath.ToSlash(isoPath), devicePath)

	cmd := exec.Command("powershell", "-Command", psCmd)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start PowerShell command: %w", err)
	}

	// Simulate progress since we can't easily track PowerShell progress
	go func() {
		for i := 1; i <= 100; i++ {
			if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
				break
			}

			time.Sleep(time.Duration(totalSize/int64(100*BufferSize)) * time.Millisecond)

			percent := float64(i) / 100.0
			glib.IdleAdd(func() {
				progress.SetFraction(percent)
				status.SetLabel(fmt.Sprintf("Burning... %d%%", i))
			})
		}
	}()

	return cmd.Wait()
}

func Sync() {
	// no-op
}
