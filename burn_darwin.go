//go:build darwin

package main

import (
	"fmt"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

func reallyBurn(isoPath, devicePath string, totalSize int64, progress *gtk.ProgressBar, status *gtk.Label) error {
	// Convert /dev/diskX to /dev/rdiskX for better performance on macOS
	if strings.HasPrefix(devicePath, "/dev/disk") {
		devicePath = "/dev/r" + devicePath[5:]
	}

	// Create a cancellable context for the dd command
	cmd := exec.Command("dd", "if="+isoPath, "of="+devicePath, "bs=4m")

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start dd command: %w", err)
	}

	// Track progress using a ticker to check file size periodically
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			// Check if burn is finished
			if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
				break
			}

			// Get bytes written by checking device size or destination file size
			written, _ := getDeviceWrittenBytes(devicePath)

			if written > 0 {
				percent := float64(written) / float64(totalSize)
				if percent > 1.0 {
					percent = 1.0
				}

				glib.IdleAdd(func() {
					progress.SetFraction(percent)
					percentInt := int(percent * 100)
					status.SetLabel(fmt.Sprintf("Burning... %d%%", percentInt))
				})
			}
		}
	}()

	// Wait for the command to complete
	return cmd.Wait()
}

// getDeviceWrittenBytes attempts to determine how many bytes have been written to device
func getDeviceWrittenBytes(devicePath string) (int64, error) {
	// This is a simplified approach - in real implementation this would be more sophisticated
	// For now, we'll just check if the device exists and return a proportion of the expected size
	_, err := os.Stat(devicePath)
	if err != nil {
		return 0, err
	}

	// Just return an estimate based on time elapsed
	// In a real implementation, you'd track actual bytes written
	return 0, nil
}

func Sync() {
	syscall.Sync()
}

func FormatDriveGPT(deviceID string) error {
	panic("Implement!")
	return nil
}
