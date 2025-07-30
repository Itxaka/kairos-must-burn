package main

import (
	"fmt"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"io"
	"os"
	"strings"
	"syscall"
)

const (
	BufferSize = 4 * 1024 * 1024 // 4MB buffer
)

// Burn writes the ISO file to the USB device with progress updates
func Burn(isoPath, drive string, progress *gtk.ProgressBar, status *gtk.Label, exitBtn *gtk.Button) {
	// Get the global variables containing selected ISO and drive

	// Validate paths
	if isoPath == "" || drive == "" {
		glib.IdleAdd(func() {
			status.SetLabel("Error: No ISO or drive selected")
			exitBtn.SetSensitive(true)
		})
		return
	}

	// Extract raw device path from the drive string (which might include description)
	devicePath := strings.Fields(drive)[0]

	// Get file size for progress calculation
	fileInfo, err := os.Stat(isoPath)
	if err != nil {
		reportError(status, exitBtn, fmt.Sprintf("Error accessing ISO: %v", err))
		return
	}

	totalSize := fileInfo.Size()

	err = reallyBurn(isoPath, devicePath, totalSize, progress, status)
	// Handle result
	if err != nil {
		reportError(status, exitBtn, fmt.Sprintf("Error during burn: %v", err))
	} else {
		glib.IdleAdd(func() {
			status.SetLabel("Burn complete! 🔥")
			exitBtn.SetSensitive(true)
		})
	}
}

// copyWithProgress copies data from src to dst with progress updates
func copyWithProgress(src io.Reader, dst io.Writer, totalSize int64, progress *gtk.ProgressBar, status *gtk.Label) error {
	buf := make([]byte, BufferSize)
	written := int64(0)

	for {
		n, err := src.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}

		if n == 0 {
			break
		}

		if _, err := dst.Write(buf[:n]); err != nil {
			return err
		}

		syscall.Sync()

		written += int64(n)
		percent := float64(written) / float64(totalSize)
		percentInt := int(percent * 100)
		if percentInt >= 100 {
			glib.IdleAdd(func() {
				progress.SetFraction(percent)
				status.SetLabel("Finalizing...")
			})
		} else {
			glib.IdleAdd(func() {
				progress.SetFraction(percent)
				status.SetLabel(fmt.Sprintf("Burning... %d%%", percentInt))
			})
		}

	}

	return nil
}

// reportError displays error message in the UI
func reportError(status *gtk.Label, exitBtn *gtk.Button, message string) {
	glib.IdleAdd(func() {
		status.SetLabel(message)
		exitBtn.SetSensitive(true)
	})
}
