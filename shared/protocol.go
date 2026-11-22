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

// User 부모 사용자 계정 모델 (Google OAuth 연동)
type User struct {
	ID        string    `json:"id"`        // Google sub 또는 고유 유저 ID
	Email     string    `json:"email"`     // 구글 이메일
	Name      string    `json:"name"`      // 이름 / 닉네임
	Picture   string    `json:"picture"`   // 프로필 이미지 URL
	CreatedAt time.Time `json:"created_at"`
}

// Child 자녀 정보 모델
type Child struct {
	ID                string    `json:"id"`
	UserID            string    `json:"user_id,omitempty"`   // 소속된 부모 사용자 ID
	Name              string    `json:"name"`
	DailyQuotaMinutes int       `json:"daily_quota_minutes"` // 일일 기본 스크린타임 (분)
	BonusMinutes      int       `json:"bonus_minutes"`       // 오늘 지급된 추가 보너스 시간 (분)
	UsedSeconds       int       `json:"used_seconds"`        // 오늘 총 사용한 시간 (초)
	ForceLocked       bool      `json:"force_locked"`        // 부모에 의한 강제 즉시 잠금 상태
	LastResetDate     string    `json:"last_reset_date"`     // "YYYY-MM-DD" 마지막 리셋 날짜
	UpdatedAt         time.Time `json:"updated_at"`

	// Phase 3 신규 규칙: 취침 모드 및 학습용 화이트리스트
	BedtimeEnabled  bool     `json:"bedtime_enabled"`  // 취침 시간 자동 잠금 활성화
	BedtimeStart    string   `json:"bedtime_start"`    // "22:00"
	BedtimeEnd      string   `json:"bedtime_end"`      // "07:00"
	WhitelistedApps []string `json:"whitelisted_apps"` // 학습/교육용 예외 앱 (차감 면제)
}

// Device 등록된 기기 정보 모델
type Device struct {
	ID           string       `json:"id"`
	ChildID      string       `json:"child_id"`
	UserID       string       `json:"user_id,omitempty"` // 소속된 부모 사용자 ID
	Name         string       `json:"name"`              // 예: "민우의 갤럭시폰", "공용 갤럭시탭", "민우 PC"
	Type         DeviceType   `json:"type"`              // android_phone, android_tablet, windows_pc
	Status       DeviceStatus `json:"status"`            // online, idle, locked, offline
	CurrentApp   string       `json:"current_app"`        // 현재 활성화된 앱 또는 창 제목
	BatteryLevel int          `json:"battery_level"`      // 배터리 잔량 (안드로이드)
	LastSeen     time.Time    `json:"last_seen"`
	UsedSeconds  int          `json:"used_seconds"`       // 오늘 이 기기에서 사용한 시간 (초)
	CreatedAt    time.Time    `json:"created_at,omitempty"`
}

// DeviceRegisterRequest 기기 등록 요청 (페어링 코드 사용)
type DeviceRegisterRequest struct {
	PairingCode string     `json:"pairing_code"`
	DeviceName  string     `json:"device_name"`
	DeviceType  DeviceType `json:"device_type"`
}

// DeviceRegisterResponse 기기 등록 결과
type DeviceRegisterResponse struct {
	Success   bool   `json:"success"`
	DeviceID  string `json:"device_id"`
	ChildID   string `json:"child_id"`
	ChildName string `json:"child_name"`
	ServerURL string `json:"server_url"`
	Error     string `json:"error,omitempty"`
}

// DeviceUnregisterRequest 기기 삭제/연동 해제 요청
type DeviceUnregisterRequest struct {
	DeviceID string `json:"device_id"`
}

// PairingCodeInfo 부모가 생성한 6자리 페어링 코드
type PairingCodeInfo struct {
	Code      string    `json:"code"`
	UserID    string    `json:"user_id,omitempty"`
	ChildID   string    `json:"child_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

// GoogleAuthRequest Google OAuth 로그인 요청
type GoogleAuthRequest struct {
	Credential string `json:"credential,omitempty"` // Google Identity Services JWT
	Email      string `json:"email,omitempty"`      // 계정 이메일
	Name       string `json:"name,omitempty"`       // 사용자 이름
	Picture    string `json:"picture,omitempty"`    // 프로필 사진 URL
}

// GoogleAuthResponse Google 로그인 응답
type GoogleAuthResponse struct {
	Success bool   `json:"success"`
	Token   string `json:"token"`
	User    *User  `json:"user,omitempty"`
	Error   string `json:"error,omitempty"`
}

// AuthLoginRequest 부모 로그인 요청
type AuthLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// AuthLoginResponse 부모 로그인 응답
type AuthLoginResponse struct {
	Success  bool   `json:"success"`
	Token    string `json:"token"`
	Username string `json:"username"`
	Error    string `json:"error,omitempty"`
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
	IsWhitelisted    bool   `json:"is_whitelisted"`    // 현재 실행 중인 앱이 학습용 예외인지 여부
}

// SystemStatus 부모 대시보드 및 텔레그램용 종합 상태
type SystemStatus struct {
	User    *User    `json:"user,omitempty"`
	Child   *Child   `json:"child"`
	Devices []Device `json:"devices"`
}
