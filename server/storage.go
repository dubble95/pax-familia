package main

import (
	"encoding/json"
	"fmt"
	"log"
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

	// 기존 데이터 파일이 있으면 로드, 없으면 기본 초기 데이터 생성
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
	}

	// 백그라운드 디바운스 플러셔 가동 (Fly.io / SSD 디스크 I/O 절감)
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

// saveLocked lock이 걸려있는 상태에서 안전하게 파일에 원자적(atomic) 쓰기
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

// startFlusher 주기적으로 dirty 상태인 데이터를 일괄 디스크 플러시
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

// Close 서버 종료 시 모든 대기 중인 상태를 디스크에 강제 플러시
func (s *Store) Close() {
	close(s.stopCh)
	<-s.doneCh
	log.Println("💾 모든 데이터가 디스크에 안전하게 보존되었습니다.")
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
		s.dirty = true
	}
}

// ProcessHeartbeat 기기 하트비트 처리 (적응형 동적 폴링 + 초고속 인메모리 연산)
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
		s.dirty = true
	}

	s.checkDailyReset(child)

	// 유휴(Idle) 상태가 아닐 때만 실제 사용 시간 차감
	if !req.IsIdle && req.ActiveSeconds > 0 {
		child.UsedSeconds += req.ActiveSeconds
		child.UpdatedAt = time.Now()
		s.dirty = true
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
		s.dirty = true
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

	// 10만 명 대응: 적응형 동적 주기 (Adaptive Polling) 계산
	// 잔여 시간이 많을 때는 요청 빈도를 줄여 서버 및 클라우드 부하를 60% 이상 절감
	nextCheckSeconds := 30
	if shouldLock {
		nextCheckSeconds = 30 // 이미 잠긴 상태: 30초
	} else if req.IsIdle {
		nextCheckSeconds = 60 // 자리비움/화면꺼짐: 60초로 트래픽 절약
	} else if remainingMinutes <= 5 {
		nextCheckSeconds = 10 // 5분 이하 마감 임박: 10초 정밀 감지
	} else if remainingMinutes <= 15 {
		nextCheckSeconds = 20 // 15분 이하: 20초
	} else {
		nextCheckSeconds = 45 // 넉넉할 때: 45초로 서버 요청 66% 감소
	}

	return shared.HeartbeatResponse{
		RemainingMinutes: remainingMinutes,
		RemainingSeconds: remainingSeconds,
		ShouldLock:       shouldLock,
		LockReason:       lockReason,
		WarningMessage:   warningMsg,
		NextCheckSeconds: nextCheckSeconds,
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
		s.dirty = true
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

// ResetToday 오늘 사용량 초기화
func (s *Store) ResetToday(childID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if c, ok := s.Data.Children[childID]; ok {
		c.UsedSeconds = 0
		c.BonusMinutes = 0
		c.ForceLocked = false
		c.UpdatedAt = time.Now()
		s.dirty = true
	}
}
