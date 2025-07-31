//go:build linux || darwin

package main

import (
	"fmt"
	"github.com/jaypipes/ghw"
	"github.com/jaypipes/ghw/pkg/block"
	"path/filepath"
	"strings"
)

func ListUSBDrives() []string {
	var drives []string
	b, err := block.New(ghw.WithDisableTools())
	if err != nil {
		fmt.Println("Error detecting block devices:", err)
		return []string{"No USB devices found"}
	}
	for _, d := range b.Disks {
		if strings.Contains(d.BusPath, "usb") {
			// Check if the disk is a USB drive
			if d.Name != "" {
				// Format the drive name and size
				size := d.SizeBytes / (1024 * 1024 * 1024) // Convert to GB
				formattedSize := fmt.Sprintf("%.2f GB", float64(size))
				drives = append(drives, fmt.Sprintf("%s (%s)", filepath.Join("/dev", d.Name), formattedSize))
			}
		}

	}
	if len(drives) == 0 {
		return []string{"No USB devices found"}
	}
	return append([]string{"🖴 Select a USB device"}, drives...)
}
