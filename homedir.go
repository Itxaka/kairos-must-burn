package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// getHomeDirectory attempts to get the real user's home directory even when running with elevated permissions
func getHomeDirectory() (string, error) {
	// Try the standard way first
	homeDir, err := os.UserHomeDir()
	if err == nil && homeDir != "" && isValidHomeDir(homeDir) {
		return homeDir, nil
	}

	// Platform-specific fallbacks when running with elevated permissions
	switch runtime.GOOS {
	case "linux":
		return getLinuxHomeDir()
	case "darwin":
		return getMacHomeDir()
	case "windows":
		return getWindowsHomeDir()
	}

	// Last resort - try environment variables
	for _, env := range []string{"HOME", "USERPROFILE"} {
		if home := os.Getenv(env); home != "" {
			return home, nil
		}
	}

	// If all else fails, use a common location
	return "/home", nil
}

// isValidHomeDir checks if a directory looks like a valid home directory
func isValidHomeDir(dir string) bool {
	// Check if it exists and is a directory
	stat, err := os.Stat(dir)
	if err != nil || !stat.IsDir() {
		return false
	}

	// Check for common subdirectories that would exist in a home dir
	for _, subdir := range []string{"Downloads", "Documents", "Desktop"} {
		if _, err := os.Stat(filepath.Join(dir, subdir)); err == nil {
			return true
		}
	}

	return false
}

// getLinuxHomeDir attempts to get the real user's home directory on Linux when running as root
func getLinuxHomeDir() (string, error) {
	// If we're running with sudo, try to get the original user's home
	sudoUser := os.Getenv("SUDO_USER")
	if sudoUser != "" {
		out, err := exec.Command("getent", "passwd", sudoUser).Output()
		if err == nil {
			fields := strings.Split(string(out), ":")
			if len(fields) >= 6 {
				return fields[5], nil // 6th field is home directory
			}
		}
	}

	// Check if we're running with pkexec
	if pkexecUID := os.Getenv("PKEXEC_UID"); pkexecUID != "" {
		out, err := exec.Command("getent", "passwd", pkexecUID).Output()
		if err == nil {
			fields := strings.Split(string(out), ":")
			if len(fields) >= 6 {
				return fields[5], nil
			}
		}
	}

	// Try to determine the real user (not root) by checking for the first non-system user
	out, err := exec.Command("getent", "passwd").Output()
	if err == nil {
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			fields := strings.Split(line, ":")
			if len(fields) >= 7 {
				uid := fields[2]
				homeDir := fields[5]
				// UID >= 1000 is usually a regular user
				if uid >= "1000" && homeDir != "/root" && isValidHomeDir(homeDir) {
					return homeDir, nil
				}
			}
		}
	}

	// Default to /home directory
	return "/home", nil
}

// getMacHomeDir attempts to get the real user's home directory on macOS when running as root
func getMacHomeDir() (string, error) {
	// If we're running with sudo, try to get the original user's home
	sudoUser := os.Getenv("SUDO_USER")
	if sudoUser != "" {
		cmd := exec.Command("dscl", ".", "read", "/Users/"+sudoUser, "NFSHomeDirectory")
		out, err := cmd.Output()
		if err == nil {
			parts := strings.Fields(string(out))
			if len(parts) >= 2 {
				return parts[1], nil
			}
		}
	}

	// Check common locations
	candidates := []string{"/Users", "/home"}
	for _, candidate := range candidates {
		if entries, err := os.ReadDir(candidate); err == nil {
			for _, entry := range entries {
				if entry.IsDir() && entry.Name() != "Shared" && entry.Name() != "Guest" {
					homeDir := filepath.Join(candidate, entry.Name())
					if isValidHomeDir(homeDir) {
						return homeDir, nil
					}
				}
			}
		}
	}

	return "/Users", nil
}

// getWindowsHomeDir attempts to get the user's home directory on Windows when running as admin
func getWindowsHomeDir() (string, error) {
	// Try to get the username first
	username := os.Getenv("USERNAME")
	if username == "" {
		// Fallback
		out, err := exec.Command("whoami").Output()
		if err == nil {
			username = strings.TrimSpace(string(out))
			if parts := strings.Split(username, "\\"); len(parts) > 1 {
				username = parts[1]
			}
		}
	}

	if username != "" {
		// Common paths for user profiles
		candidates := []string{
			filepath.Join(os.Getenv("SystemDrive")+"\\", "Users", username),
			filepath.Join(os.Getenv("SystemDrive")+"\\", "Documents and Settings", username),
		}

		for _, path := range candidates {
			if isValidHomeDir(path) {
				return path, nil
			}
		}
	}

	// If all else fails
	return filepath.Join(os.Getenv("SystemDrive")+"\\", "Users"), nil
}
