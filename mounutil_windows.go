//go:build windows

package main

func IsDeviceMounted(device string) ([]string, error) {
	return []string{}, nil
}

func UnmountDevice(partitions []string) error {
	return nil
}
