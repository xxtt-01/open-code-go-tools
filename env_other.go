//go:build !windows

package main

func setWindowsUserEnvironment(name, value string) error {
	return nil
}

func unsetWindowsUserEnvironment(name string) error {
	return nil
}
