//go:build windows

package main

import (
	"fmt"
	"github.com/bi-zone/wmi"
	"strings"
)

func ListUSBDrives() []string {
	type Win32Diskdrive struct {
		DeviceID      string
		Model         string
		InterfaceType string
		MediaType     string
		Size          uint64
	}

	var drives []string
	var dst []Win32Diskdrive
	err := wmi.Query("SELECT DeviceID, Model, InterfaceType, MediaType, Size, MediaType FROM Win32_DiskDrive", &dst)
	if err != nil {
		return []string{"Error querying WMI: " + err.Error()}
	}

	for _, d := range dst {
		if d.InterfaceType == "USB" || strings.Contains(strings.ToLower(d.MediaType), "external") || d.MediaType == "Removable Media" {
			fmt.Println("Found USB drive:", d)
			formattedSize := fmt.Sprintf("%.2f GB", float64(d.Size/(1024*1024*1024))) // Convert to GB
			drives = append(drives, fmt.Sprintf("%s (%s %s)", d.DeviceID, d.Model, formattedSize))
		}
	}

	if len(drives) == 0 {
		return []string{"No USB devices found"}
	}
	return append([]string{"🖴 Select a USB device"}, drives...)
}
