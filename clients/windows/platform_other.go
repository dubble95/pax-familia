//go:build !windows

package main

import (
	"log"
	"time"
)

// Non-Windows (macOS / Linux) 테스트 및 크로스 환경용 Stub
type MockPlatformOps struct{}

func NewPlatformOps() PlatformOps {
	return &MockPlatformOps{}
}

func (m *MockPlatformOps) LockScreen() {
	log.Println("🛑 [Mock Platform] 윈도우 잠금 화면(LockWorkStation)이 호출되었습니다! (화면 잠금 시뮬레이션)")
}

func (m *MockPlatformOps) GetIdleDuration() time.Duration {
	// 모의 환경에서는 유휴 상태 아님으로 반환
	return 0
}

func (m *MockPlatformOps) GetForegroundWindowTitle() string {
	return "Google Chrome - Roblox"
}

func (m *MockPlatformOps) ShowNotification(title, message string) {
	log.Printf("📢 [알림 팝업] %s: %s", title, message)
}
