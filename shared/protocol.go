package shared

import "time"

// DeviceType 기기 유형
type DeviceType string

const (
	DeviceAndroidPhone  DeviceType = "android_phone"
	DeviceAndroidTablet DeviceType = "android_tablet"
	DeviceWindowsPC     DeviceType = "windows_pc"
)

// DeviceStatus 기기 상태
type DeviceStatus string

const (
	StatusOnline  DeviceStatus = "online"
	StatusIdle    DeviceStatus = "idle"
	StatusOffline DeviceStatus = "offline"
	StatusLocked  DeviceStatus = "locked"
)

// HeartbeatRequest 클라이언트에서 30초마다 전송하는 심장박동 요청
type HeartbeatRequest struct {
	DeviceID      string     `json:"device_id"`
	ChildID       string     `json:"child_id"`
	DeviceType    DeviceType `json:"device_type"`
	ActiveSeconds int        `json:"active_seconds"` // 이전 하트비트 이후 실제 사용한 초
	CurrentApp    string     `json:"current_app"`    // 현재 포그라운드 앱/창 제목
	BatteryLevel  int        `json:"battery_level"`  // 배터리 잔량 (옵션)
}

// HeartbeatResponse 서버가 클라이언트에게 전달하는 제어 명령
type HeartbeatResponse struct {
	RemainingMinutes int    `json:"remaining_minutes"` // 오늘 남은 총 사용시간 (분)
	ShouldLock       bool   `json:"should_lock"`       // 지금 즉시 잠금 여부
	LockReason       string `json:"lock_reason"`       // 잠금 사유 ("시간 소진", "취침 시간", "부모 강제 잠금")
	WarningMessage   string `json:"warning_message"`   // 경고 메시지 ("5분 남았습니다")
	NextCheckSeconds int    `json:"next_check_seconds"` // 다음 하트비트 주기 (기본 30초)
}
