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
			drives = append(drives, fmt.Sprintf("%s (%s)", d.Model, d.DeviceID))
		}
	}

	if len(drives) == 0 {
		return []string{"No USB devices found"}
	}
	return append([]string{"🖴 Select a USB device"}, drives...)
}
