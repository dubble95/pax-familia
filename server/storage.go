package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"pax-familia/shared"
)

type Store struct {
	mu       sync.RWMutex
	filePath string
	Data     StoreData `json:"data"`
}

type StoreData struct {
	Children map[string]*shared.Child  `json:"children"`
	Devices  map[string]*shared.Device `json:"devices"`
}

func NewStore(filePath string) (*Store, error) {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	s := &Store{
		filePath: filePath,
		Data: StoreData{
			Children: make(map[string]*shared.Child),
			Devices:  make(map[string]*shared.Device),
		},
	}

	// 기존 데이터 파일이 있으면 로드, 없으면 기본 초기 데이터 생성
	if err := s.load(); err != nil {
		// 파일이 없으면 기본 자녀 생성
		today := time.Now().Format("2006-01-02")
		defaultChild := &shared.Child{
			ID:                "child_1",
			Name:              "우리 아이",
			DailyQuotaMinutes: 120, // 하루 2시간
			BonusMinutes:      0,
			UsedSeconds:       0,
			ForceLocked:       false,
			LastResetDate:     today,
			UpdatedAt:         time.Now(),
		}
		s.Data.Children[defaultChild.ID] = defaultChild

		// 기본 대상 기기 3대 초기 등록
		s.Data.Devices["dev_phone_1"] = &shared.Device{
			ID:         "dev_phone_1",
			ChildID:    "child_1",
			Name:       "안드로이드 스마트폰",
			Type:       shared.DeviceAndroidPhone,
			Status:     shared.StatusOffline,
			CurrentApp: "대기 중",
			LastSeen:   time.Now().Add(-1 * time.Hour),
		}
		s.Data.Devices["dev_tablet_1"] = &shared.Device{
			ID:         "dev_tablet_1",
			ChildID:    "child_1",
			Name:       "안드로이드 태블릿",
			Type:       shared.DeviceAndroidTablet,
			Status:     shared.StatusOffline,
			CurrentApp: "대기 중",
			LastSeen:   time.Now().Add(-1 * time.Hour),
		}
		s.Data.Devices["dev_pc_1"] = &shared.Device{
			ID:         "dev_pc_1",
			ChildID:    "child_1",
			Name:       "윈도우 PC",
			Type:       shared.DeviceWindowsPC,
			Status:     shared.StatusOffline,
			CurrentApp: "대기 중",
			LastSeen:   time.Now().Add(-1 * time.Hour),
		}

		_ = s.save()
	}

	return s, nil
}

func (s *Store) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &s.Data)
}

func (s *Store) save() error {
	bytes, err := json.MarshalIndent(s.Data, "", "  ")
	if err != nil {
		return err
	}
	tmpFile := fmt.Sprintf("%s.tmp", s.filePath)
	if err := os.WriteFile(tmpFile, bytes, 0644); err != nil {
		return err
	}
	return os.Rename(tmpFile, s.filePath)
}

// checkDailyReset 날짜가 바뀌면 사용 시간 초기화
func (s *Store) checkDailyReset(child *shared.Child) {
	today := time.Now().Format("2006-01-02")
	if child.LastResetDate != today {
		child.LastResetDate = today
		child.UsedSeconds = 0
		child.BonusMinutes = 0
		child.ForceLocked = false
		child.UpdatedAt = time.Now()
	}
}

// ProcessHeartbeat 기기 하트비트 처리
func (s *Store) ProcessHeartbeat(req shared.HeartbeatRequest) shared.HeartbeatResponse {
	s.mu.Lock()
	defer s.mu.Unlock()

	childID := req.ChildID
	if childID == "" {
		childID = "child_1"
	}

	child, exists := s.Data.Children[childID]
	if !exists {
		today := time.Now().Format("2006-01-02")
		child = &shared.Child{
			ID:                childID,
			Name:              "우리 아이",
			DailyQuotaMinutes: 120,
			LastResetDate:     today,
		}
		s.Data.Children[childID] = child
	}

	s.checkDailyReset(child)

	// 유휴(Idle) 상태가 아닐 때만 실제 사용 시간 차감
	if !req.IsIdle && req.ActiveSeconds > 0 {
		child.UsedSeconds += req.ActiveSeconds
		child.UpdatedAt = time.Now()
	}

	// 기기 정보 등록 및 업데이트
	dev, devExists := s.Data.Devices[req.DeviceID]
	if !devExists {
		devName := req.DeviceName
		if devName == "" {
			devName = string(req.DeviceType)
		}
		dev = &shared.Device{
			ID:      req.DeviceID,
			ChildID: childID,
			Name:    devName,
			Type:    req.DeviceType,
		}
		s.Data.Devices[req.DeviceID] = dev
	}

	dev.LastSeen = time.Now()
	dev.CurrentApp = req.CurrentApp
	dev.BatteryLevel = req.BatteryLevel

	// 총 허용 시간(초) 계산
	totalAllowedMinutes := child.DailyQuotaMinutes + child.BonusMinutes
	totalAllowedSeconds := totalAllowedMinutes * 60
	remainingSeconds := totalAllowedSeconds - child.UsedSeconds
	if remainingSeconds < 0 {
		remainingSeconds = 0
	}
	remainingMinutes := (remainingSeconds + 59) / 60

	// 잠금 여부 판정
	shouldLock := false
	lockReason := ""
	warningMsg := ""

	if child.ForceLocked {
		shouldLock = true
		lockReason = "부모에 의한 즉시 잠금"
		dev.Status = shared.StatusLocked
	} else if remainingSeconds <= 0 {
		shouldLock = true
		lockReason = "일일 허용 사용시간 모두 소진"
		dev.Status = shared.StatusLocked
	} else {
		if req.IsIdle {
			dev.Status = shared.StatusIdle
		} else {
			dev.Status = shared.StatusOnline
		}

		if remainingMinutes <= 5 {
			warningMsg = fmt.Sprintf("남은 시간이 %d분 남았습니다. 작업을 마무리하세요.", remainingMinutes)
		}
	}

	_ = s.save()

	return shared.HeartbeatResponse{
		RemainingMinutes: remainingMinutes,
		RemainingSeconds: remainingSeconds,
		ShouldLock:       shouldLock,
		LockReason:       lockReason,
		WarningMessage:   warningMsg,
		NextCheckSeconds: 15,
	}
}

// GetStatus 전체 현황 반환
func (s *Store) GetStatus(childID string) shared.SystemStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	if childID == "" {
		childID = "child_1"
	}

	child, exists := s.Data.Children[childID]
	if !exists {
		today := time.Now().Format("2006-01-02")
		child = &shared.Child{
			ID:                childID,
			Name:              "우리 아이",
			DailyQuotaMinutes: 120,
			LastResetDate:     today,
		}
		s.Data.Children[childID] = child
	}
	s.checkDailyReset(child)

	now := time.Now()
	var devList []shared.Device
	for _, dev := range s.Data.Devices {
		if dev.ChildID == childID {
			dCopy := *dev
			// 1분 이상 응답이 없으면 offline 처리
			if now.Sub(dCopy.LastSeen) > 60*time.Second {
				dCopy.Status = shared.StatusOffline
			}
			devList = append(devList, dCopy)
		}
	}

	return shared.SystemStatus{
		Child:   child,
		Devices: devList,
	}
}

// AddBonus 보너스 시간 추가
func (s *Store) AddBonus(childID string, minutes int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if child, exists := s.Data.Children[childID]; exists {
		child.BonusMinutes += minutes
		child.UpdatedAt = time.Now()
		_ = s.save()
	}
}

// SetForceLock 부모 강제 즉시 잠금 설정/해제
func (s *Store) SetForceLock(childID string, lock bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if child, exists := s.Data.Children[childID]; exists {
		child.ForceLocked = lock
		child.UpdatedAt = time.Now()
		_ = s.save()
	}
}

// SetDailyQuota 일일 기본 한도 변경
func (s *Store) SetDailyQuota(childID string, minutes int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if child, exists := s.Data.Children[childID]; exists {
		child.DailyQuotaMinutes = minutes
		child.UpdatedAt = time.Now()
		_ = s.save()
	}
}
