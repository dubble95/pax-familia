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

// Child 자녀 정보 모델
type Child struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	DailyQuotaMinutes int       `json:"daily_quota_minutes"` // 일일 기본 스크린타임 (분)
	BonusMinutes      int       `json:"bonus_minutes"`       // 오늘 지급된 추가 보너스 시간 (분)
	UsedSeconds       int       `json:"used_seconds"`        // 오늘 총 사용한 시간 (초)
	ForceLocked       bool      `json:"force_locked"`        // 부모에 의한 강제 즉시 잠금 상태
	LastResetDate     string    `json:"last_reset_date"`     // "YYYY-MM-DD" 마지막 리셋 날짜
	UpdatedAt         time.Time `json:"updated_at"`
}

// Device 등록된 기기 정보 모델
type Device struct {
	ID           string       `json:"id"`
	ChildID      string       `json:"child_id"`
	Name         string       `json:"name"`         // 예: "민우의 갤럭시폰", "공용 갤럭시탭", "민우 PC"
	Type         DeviceType   `json:"type"`         // android_phone, android_tablet, windows_pc
	Status       DeviceStatus `json:"status"`       // online, idle, locked, offline
	CurrentApp   string       `json:"current_app"`   // 현재 활성화된 앱 또는 창 제목
	BatteryLevel int          `json:"battery_level"` // 배터리 잔량 (안드로이드)
	LastSeen     time.Time    `json:"last_seen"`
}

// HeartbeatRequest 클라이언트에서 10~30초마다 전송하는 요청
type HeartbeatRequest struct {
	DeviceID      string     `json:"device_id"`
	ChildID       string     `json:"child_id"`
	DeviceName    string     `json:"device_name"`
	DeviceType    DeviceType `json:"device_type"`
	ActiveSeconds int        `json:"active_seconds"` // 이전 하트비트 이후 실제 사용한 활성 시간 (초)
	CurrentApp    string     `json:"current_app"`    // 현재 포그라운드 앱 / 창 제목
	BatteryLevel  int        `json:"battery_level"`  // 배터리 % (해당 없으면 -1)
	IsIdle        bool       `json:"is_idle"`        // 입력 없음(유휴 상태) 여부
}

// HeartbeatResponse 서버가 클라이언트에게 회신하는 제어 명령
type HeartbeatResponse struct {
	RemainingMinutes int    `json:"remaining_minutes"` // 오늘 남은 총 시간 (분)
	RemainingSeconds int    `json:"remaining_seconds"` // 오늘 남은 총 시간 (초 단위)
	ShouldLock       bool   `json:"should_lock"`       // 지금 즉시 화면을 잠궈야 하는지 여부
	LockReason       string `json:"lock_reason"`       // 잠금 사유 ("시간 소진", "부모 강제 잠금", "취침 시간")
	WarningMessage   string `json:"warning_message"`   // 경고 메시지 ("남은 시간 5분입니다")
	NextCheckSeconds int    `json:"next_check_seconds"` // 다음 하트비트 폴링 간격 (초)
}

// SystemStatus 부모 대시보드 및 텔레그램용 종합 상태
type SystemStatus struct {
	Child   *Child   `json:"child"`
	Devices []Device `json:"devices"`
}
