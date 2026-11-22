//go:build windows

package main

import (
	"syscall"
	"time"
	"unsafe"
)

var (
	user32                = syscall.NewLazyDLL("user32.dll")
	procLockWorkStation   = user32.NewProc("LockWorkStation")
	procGetLastInputInfo  = user32.NewProc("GetLastInputInfo")
	procGetForegroundWin  = user32.NewProc("GetForegroundWindow")
	procGetWindowTextW    = user32.NewProc("GetWindowTextW")
	kernel32              = syscall.NewLazyDLL("kernel32.dll")
	procGetTickCount      = kernel32.NewProc("GetTickCount")
)

type LASTINPUTINFO struct {
	cbSize uint32
	dwTime uint32
}

type WindowsPlatformOps struct{}

func NewPlatformOps() PlatformOps {
	return &WindowsPlatformOps{}
}

// LockScreen Windows 화면 즉시 잠금 (LockWorkStation)
func (w *WindowsPlatformOps) LockScreen() {
	procLockWorkStation.Call()
}

// GetIdleDuration 사용자의 마지막 마우스/키보드 입력 이후 경과 시간
func (w *WindowsPlatformOps) GetIdleDuration() time.Duration {
	var lii LASTINPUTINFO
	lii.cbSize = uint32(unsafe.Sizeof(lii))

	ret, _, _ := procGetLastInputInfo.Call(uintptr(unsafe.Pointer(&lii)))
	if ret == 0 {
		return 0
	}

	tick, _, _ := procGetTickCount.Call()
	elapsedMs := uint32(tick) - lii.dwTime
	return time.Duration(elapsedMs) * time.Millisecond
}

// GetForegroundWindowTitle 현재 활성화된 전면 창의 제목 추출
func (w *WindowsPlatformOps) GetForegroundWindowTitle() string {
	hwnd, _, _ := procGetForegroundWin.Call()
	if hwnd == 0 {
		return ""
	}

	buf := make([]uint16, 256)
	ret, _, _ := procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), 256)
	if ret == 0 {
		return ""
	}

	return syscall.UTF16ToString(buf)
}

func (w *WindowsPlatformOps) ShowNotification(title, message string) {
	// 기본 로그로 표시 (향후 Windows Balloon Notification 확장 가능)
}
