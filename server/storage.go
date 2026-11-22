package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"pax-familia/shared"
)

var DefaultWhitelistedApps = []string{
	"EBS", "메가스터디", "엠베스트", "클래스101", "구글 클래스룸",
	"Google Classroom", "Zoom", "Notion", "Wikipedia", "나무위키",
}

type Store struct {
	mu       sync.RWMutex
	filePath string
	Data     StoreData `json:"data"`
	dirty    bool
	stopCh   chan struct{}
	doneCh   chan struct{}
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
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}

	// 기존 데이터 파일 로드 또는 기본값 초기화
	if err := s.load(); err != nil {
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
			BedtimeEnabled:    false,
			BedtimeStart:      "22:00",
			BedtimeEnd:        "07:00",
			WhitelistedApps:   DefaultWhitelistedApps,
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

		_ = s.saveDirect()
	} else {
		// 기존 데이터 파일이 로드되었을 때 Phase 3 필드가 비어있으면 기본값 보정
		s.mu.Lock()
		for _, child := range s.Data.Children {
			if len(child.WhitelistedApps) == 0 {
				child.WhitelistedApps = DefaultWhitelistedApps
				s.dirty = true
			}
			if child.BedtimeStart == "" {
				child.BedtimeStart = "22:00"
				child.BedtimeEnd = "07:00"
				s.dirty = true
			}
		}
		s.mu.Unlock()
	}

	// 백그라운드 디바운스 플러셔 가동
	s.startFlusher()

	return s, nil
}

func (s *Store) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &s.Data)
}

func (s *Store) saveLocked() error {
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

func (s *Store) saveDirect() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveLocked()
}

func (s *Store) startFlusher() {
	ticker := time.NewTicker(2 * time.Second)
	go func() {
		defer close(s.doneCh)
		for {
			select {
			case <-ticker.C:
				s.mu.Lock()
				if s.dirty {
					if err := s.saveLocked(); err != nil {
						log.Printf("⚠️ 디스크 저장 오류: %v", err)
					} else {
						s.dirty = false
					}
				}
				s.mu.Unlock()
			case <-s.stopCh:
				ticker.Stop()
				s.mu.Lock()
				if s.dirty {
					_ = s.saveLocked()
					s.dirty = false
				}
				s.mu.Unlock()
				return
			}
		}
	}()
}

func (s *Store) Close() {
	close(s.stopCh)
	<-s.doneCh
	log.Println("💾 모든 데이터가 디스크에 안전하게 보존되었습니다.")
}

func (s *Store) checkDailyReset(child *shared.Child) {
	today := time.Now().Format("2006-01-02")
	if child.LastResetDate != today {
		child.LastResetDate = today
		child.UsedSeconds = 0
		child.BonusMinutes = 0
		child.ForceLocked = false
		child.UpdatedAt = time.Now()

		// 기기별 오늘 사용량도 리셋
		for _, dev := range s.Data.Devices {
			if dev.ChildID == child.ID {
				dev.UsedSeconds = 0
			}
		}
		s.dirty = true
	}
}

// isBedtime 현재 시각(KST 또는 로컬)이 취침 시간 범위에 포함되는지 검사
func isBedtime(start, end string) bool {
	if start == "" || end == "" {
		return false
	}
	cur := time.Now().Format("15:04")
	if start < end {
		return cur >= start && cur < end
	}
	// 자정을 넘기는 경우 (예: 22:00 ~ 07:00)
	return cur >= start || cur < end
}

// isAppWhitelisted 현재 앱이 학습용 예외 화이트리스트에 포함되는지 검사 (대소문자 무시 부분 일치)
func isAppWhitelisted(currentApp string, whitelist []string) bool {
	if currentApp == "" || len(whitelist) == 0 {
		return false
	}
	lowerApp := strings.ToLower(currentApp)
	for _, w := range whitelist {
		if strings.Contains(lowerApp, strings.ToLower(w)) {
			return true
		}
	}
	return false
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
			BedtimeEnabled:    false,
			BedtimeStart:      "22:00",
			BedtimeEnd:        "07:00",
			WhitelistedApps:   DefaultWhitelistedApps,
		}
		s.Data.Children[childID] = child
		s.dirty = true
	}

	s.checkDailyReset(child)

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
		s.dirty = true
	}

	dev.LastSeen = time.Now()
	dev.CurrentApp = req.CurrentApp
	dev.BatteryLevel = req.BatteryLevel

	// 학습용 앱(화이트리스트) 여부 검사
	isWhitelisted := isAppWhitelisted(req.CurrentApp, child.WhitelistedApps)

	// 유휴(Idle)가 아닐 때 시간 누적 처리
	warningMsg := ""
	if !req.IsIdle && req.ActiveSeconds > 0 {
		dev.UsedSeconds += req.ActiveSeconds

		if isWhitelisted {
			// 학습용 앱은 스크린타임 차감 제외!
			warningMsg = fmt.Sprintf("📖 [%s] 학습용 앱 사용 중 (스크린타임 차감 제외)", req.CurrentApp)
		} else {
			child.UsedSeconds += req.ActiveSeconds
		}

		child.UpdatedAt = time.Now()
		s.dirty = true
	}

	// 총 허용 시간(초) 계산
	totalAllowedMinutes := child.DailyQuotaMinutes + child.BonusMinutes
	totalAllowedSeconds := totalAllowedMinutes * 60
	remainingSeconds := totalAllowedSeconds - child.UsedSeconds
	if remainingSeconds < 0 {
		remainingSeconds = 0
	}
	remainingMinutes := (remainingSeconds + 59) / 60

	// 잠금 여부 판정 (우선순위: 부모 즉시 잠금 > 취침 모드 > 시간 소진)
	shouldLock := false
	lockReason := ""

	if child.ForceLocked {
		shouldLock = true
		lockReason = "부모에 의한 즉시 잠금"
		dev.Status = shared.StatusLocked
	} else if child.BedtimeEnabled && isBedtime(child.BedtimeStart, child.BedtimeEnd) {
		shouldLock = true
		lockReason = fmt.Sprintf("취침 시간 잠금 모드 (%s ~ %s)", child.BedtimeStart, child.BedtimeEnd)
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

		if warningMsg == "" && remainingMinutes <= 5 {
			warningMsg = fmt.Sprintf("남은 시간이 %d분 남았습니다. 작업을 마무리하세요.", remainingMinutes)
		}
	}

	// 적응형 동적 주기 계산
	nextCheckSeconds := 30
	if shouldLock {
		nextCheckSeconds = 30
	} else if req.IsIdle {
		nextCheckSeconds = 60
	} else if remainingMinutes <= 5 {
		nextCheckSeconds = 10
	} else if remainingMinutes <= 15 {
		nextCheckSeconds = 20
	} else {
		nextCheckSeconds = 45
	}

	return shared.HeartbeatResponse{
		RemainingMinutes: remainingMinutes,
		RemainingSeconds: remainingSeconds,
		ShouldLock:       shouldLock,
		LockReason:       lockReason,
		WarningMessage:   warningMsg,
		NextCheckSeconds: nextCheckSeconds,
		IsWhitelisted:    isWhitelisted,
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
			BedtimeEnabled:    false,
			BedtimeStart:      "22:00",
			BedtimeEnd:        "07:00",
			WhitelistedApps:   DefaultWhitelistedApps,
		}
		s.Data.Children[childID] = child
		s.dirty = true
	}
	s.checkDailyReset(child)

	now := time.Now()
	var devList []shared.Device
	for _, dev := range s.Data.Devices {
		if dev.ChildID == childID {
			dCopy := *dev
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
		s.dirty = true
	}
}

// SetForceLock 부모 강제 즉시 잠금 설정/해제
func (s *Store) SetForceLock(childID string, lock bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if child, exists := s.Data.Children[childID]; exists {
		child.ForceLocked = lock
		child.UpdatedAt = time.Now()
		s.dirty = true
	}
}

// SetDailyQuota 일일 기본 한도 변경
func (s *Store) SetDailyQuota(childID string, minutes int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if child, exists := s.Data.Children[childID]; exists {
		child.DailyQuotaMinutes = minutes
		child.UpdatedAt = time.Now()
		s.dirty = true
	}
}

// SetBedtime 취침 모드 설정
func (s *Store) SetBedtime(childID string, enabled bool, start, end string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if child, exists := s.Data.Children[childID]; exists {
		child.BedtimeEnabled = enabled
		if start != "" {
			child.BedtimeStart = start
		}
		if end != "" {
			child.BedtimeEnd = end
		}
		child.UpdatedAt = time.Now()
		s.dirty = true
	}
}

// SetWhitelistedApps 화이트리스트 전체 교체
func (s *Store) SetWhitelistedApps(childID string, apps []string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if child, exists := s.Data.Children[childID]; exists {
		child.WhitelistedApps = apps
		child.UpdatedAt = time.Now()
		s.dirty = true
	}
}

// AddWhitelistedApp 화이트리스트 앱 단일 추가
func (s *Store) AddWhitelistedApp(childID string, app string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if child, exists := s.Data.Children[childID]; exists {
		trimmed := strings.TrimSpace(app)
		if trimmed == "" {
			return
		}
		for _, existing := range child.WhitelistedApps {
			if strings.EqualFold(existing, trimmed) {
				return
			}
		}
		child.WhitelistedApps = append(child.WhitelistedApps, trimmed)
		child.UpdatedAt = time.Now()
		s.dirty = true
	}
}

// RemoveWhitelistedApp 화이트리스트 앱 단일 삭제
func (s *Store) RemoveWhitelistedApp(childID string, app string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if child, exists := s.Data.Children[childID]; exists {
		var filtered []string
		for _, existing := range child.WhitelistedApps {
			if !strings.EqualFold(existing, strings.TrimSpace(app)) {
				filtered = append(filtered, existing)
			}
		}
		child.WhitelistedApps = filtered
		child.UpdatedAt = time.Now()
		s.dirty = true
	}
}

// ResetToday 오늘 사용량 초기화
func (s *Store) ResetToday(childID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if c, ok := s.Data.Children[childID]; ok {
		c.UsedSeconds = 0
		c.BonusMinutes = 0
		c.ForceLocked = false
		c.UpdatedAt = time.Now()
		for _, dev := range s.Data.Devices {
			if dev.ChildID == childID {
				dev.UsedSeconds = 0
			}
		}
		s.dirty = true
	}
}
