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
	fmt.Println("Device Path:", devicePath)

	// Open ISO file for reading
	isoFile, err := os.Open(isoPath)
	if err != nil {
		return fmt.Errorf("failed to open ISO file: %w", err)
	}
	defer isoFile.Close()
	fmt.Println("ISO File:", isoPath)
	// Open device for writing
	deviceFile, err := os.OpenFile(devicePath, os.O_WRONLY, 0)
	if err != nil {
		// Fallback to using PowerShell commands
		return burnWithPowerShell(isoPath, devicePath, totalSize, progress, status)
	}
	defer deviceFile.Close()
	fmt.Println("burning")
	glib.IdleAdd(func() {
		status.SetLabel("Starting burn...")
	})

	// Copy with progress
	return copyWithProgress(isoFile, deviceFile, totalSize, progress, status)
}

// burnWithPowerShell is a fallback method for Windows when direct access fails
// This path is not really tested...
func burnWithPowerShell(isoPath, devicePath string, totalSize int64, progress *gtk.ProgressBar, status *gtk.Label) error {
	// PowerShell command to write ISO to disk
	fmt.Println("burning with powershell")
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

// FormatDriveGPT cleans the drive and converts it to GPT using diskpart
func FormatDriveGPT(deviceID string) error {
	diskNum := extractDiskNumber(deviceID)
	if diskNum == "" {
		return fmt.Errorf("could not extract disk number from deviceID: %s", deviceID)
	}

	diskpartScript := fmt.Sprintf("select disk %s\nclean\nconvert gpt\n", diskNum)
	tmpFile, err := os.CreateTemp("", "diskpart_script_*.txt")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())
	_, err = tmpFile.WriteString(diskpartScript)
	if err != nil {
		return err
	}
	_ = tmpFile.Close()

	cmd := exec.Command("diskpart", "/s", tmpFile.Name())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("diskpart error: %v, output: %s", err, string(output))
	}
	return nil
}

// extractDiskNumber extracts the disk number from a DeviceID string like "\\.\PHYSICALDRIVE1"
func extractDiskNumber(deviceID string) string {
	parts := strings.Split(deviceID, "PHYSICALDRIVE")
	if len(parts) == 2 {
		return strings.TrimSpace(parts[1])
	}
	return ""
}
