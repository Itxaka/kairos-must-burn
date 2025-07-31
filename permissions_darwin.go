//go:build darwin

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func CheckElevatedPermissions() (bool, error) {
	if os.Geteuid() != 0 {
		// Re-exec with osascript and admin prompt
		exePath, _ := filepath.Abs(os.Args[0])
		cmd := exec.Command("osascript", "-e", fmt.Sprintf(`
		do shell script "%s" with administrator privileges with prompt "Kairos-must-burn needs permission to write to your USB device."`, exePath))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			os.Exit(1)
		}
		return true, nil
	}
	os.Exit(0)
	return false, nil
}
