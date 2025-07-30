//go:build !linux && !darwin && !windows

package main

// CheckElevatedPermissions checks if the application is running with elevated permissions
func CheckElevatedPermissions() (bool, error) {
	return false, nil
}
