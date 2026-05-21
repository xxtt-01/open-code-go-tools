//go:build windows

package main

import (
	"os/exec"
	"syscall"
	"unsafe"
)

const (
	hwndBroadcast      = 0xffff
	wmSettingChange    = 0x001A
	smtoAbortIfHung    = 0x0002
	environmentMessage = "Environment"
)

var (
	user32                 = syscall.NewLazyDLL("user32.dll")
	procSendMessageTimeout = user32.NewProc("SendMessageTimeoutW")
)

func setWindowsUserEnvironment(name, value string) error {
	cmd := exec.Command("setx.exe", name, value)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Run(); err != nil {
		return err
	}
	broadcastEnvironmentChange()
	return nil
}

func unsetWindowsUserEnvironment(name string) error {
	cmd := exec.Command("reg.exe", "delete", `HKCU\Environment`, "/v", name, "/f")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return nil
		}
		return err
	}
	broadcastEnvironmentChange()
	return nil
}

func broadcastEnvironmentChange() {
	msg, err := syscall.UTF16PtrFromString(environmentMessage)
	if err != nil {
		return
	}
	procSendMessageTimeout.Call(
		uintptr(hwndBroadcast),
		uintptr(wmSettingChange),
		0,
		uintptr(unsafe.Pointer(msg)),
		uintptr(smtoAbortIfHung),
		uintptr(5000),
		0,
	)
}
