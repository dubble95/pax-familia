package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"sort"
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
	sessions map[string]*shared.User // token -> User
}

type StoreData struct {
	Users        map[string]*shared.User            `json:"users"`
	Children     map[string]*shared.Child           `json:"children"`
	Devices      map[string]*shared.Device          `json:"devices"`
	ParentUsers  map[string]string                  `json:"parent_users"` // username -> sha256 hex
	PairingCodes map[string]*shared.PairingCodeInfo `json:"pairing_codes"`
}

func hashPassword(password string) string {
	sum := sha256.Sum256([]byte(password))
	return hex.EncodeToString(sum[:])
}

func NewStore(filePath string) (*Store, error) {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	s := &Store{
		filePath: filePath,
		Data: StoreData{
			Users:        make(map[string]*shared.User),
			Children:     make(map[string]*shared.Child),
			Devices:      make(map[string]*shared.Device),
			ParentUsers:  make(map[string]string),
			PairingCodes: make(map[string]*shared.PairingCodeInfo),
		},
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
		sessions: make(map[string]*shared.User),
	}

	// 기존 데이터 파일 로드 또는 기본값 초기화
	if err := s.load(); err != nil {
		today := time.Now().Format("2006-01-02")
		defaultChild := &shared.Child{
			ID:                "child_1",
			UserID:            "admin",
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

		baseTime := time.Now().Add(-2 * time.Hour)
		// 기본 대상 기기 3대 초기 등록 (등록 순서 보존)
		s.Data.Devices["dev_phone_1"] = &shared.Device{
			ID:         "dev_phone_1",
			ChildID:    "child_1",
			UserID:     "admin",
			Name:       "안드로이드 스마트폰",
			Type:       shared.DeviceAndroidPhone,
			Status:     shared.StatusOffline,
			CurrentApp: "대기 중",
			LastSeen:   time.Now().Add(-1 * time.Hour),
			CreatedAt:  baseTime,
		}
		s.Data.Devices["dev_tablet_1"] = &shared.Device{
			ID:         "dev_tablet_1",
			ChildID:    "child_1",
			UserID:     "admin",
			Name:       "안드로이드 태블릿",
			Type:       shared.DeviceAndroidTablet,
			Status:     shared.StatusOffline,
			CurrentApp: "대기 중",
			LastSeen:   time.Now().Add(-1 * time.Hour),
			CreatedAt:  baseTime.Add(1 * time.Minute),
		}
		s.Data.Devices["dev_pc_1"] = &shared.Device{
			ID:         "dev_pc_1",
			ChildID:    "child_1",
			UserID:     "admin",
			Name:       "윈도우 PC",
			Type:       shared.DeviceWindowsPC,
			Status:     shared.StatusOffline,
			CurrentApp: "대기 중",
			LastSeen:   time.Now().Add(-1 * time.Hour),
			CreatedAt:  baseTime.Add(2 * time.Minute),
		}

		s.Data.ParentUsers["admin"] = hashPassword("pax1234")
		s.Data.Users["admin"] = &shared.User{
			ID:        "admin",
			Email:     "admin@paxfamilia.com",
			Name:      "관리자 (부모)",
			Picture:   "",
			CreatedAt: time.Now(),
		}

		_ = s.saveDirect()
	} else {
		// 기존 데이터 파일이 로드되었을 때 보정
		s.mu.Lock()
		if s.Data.Users == nil {
			s.Data.Users = make(map[string]*shared.User)
		}
		if _, ok := s.Data.Users["admin"]; !ok {
			s.Data.Users["admin"] = &shared.User{
				ID:        "admin",
				Email:     "admin@paxfamilia.com",
				Name:      "관리자 (부모)",
				Picture:   "",
				CreatedAt: time.Now(),
			}
		}
		if s.Data.ParentUsers == nil {
			s.Data.ParentUsers = make(map[string]string)
		}
		if len(s.Data.ParentUsers) == 0 {
			s.Data.ParentUsers["admin"] = hashPassword("pax1234")
			s.dirty = true
		}
		if s.Data.PairingCodes == nil {
			s.Data.PairingCodes = make(map[string]*shared.PairingCodeInfo)
		}
		if c, ok := s.Data.Children["child_1"]; ok && c.UserID == "" {
			c.UserID = "admin"
			s.dirty = true
		}
		for _, dev := range s.Data.Devices {
			if dev.ChildID == "child_1" && dev.UserID == "" {
				dev.UserID = "admin"
				s.dirty = true
			}
		}
		idx := 0
		for _, dev := range s.Data.Devices {
			if dev.CreatedAt.IsZero() {
				dev.CreatedAt = time.Now().Add(-time.Duration(100-idx) * time.Hour)
				idx++
				s.dirty = true
			}
		}
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

// GetStatus 전체 현황 반환 (사용자별 멀티테넌시 격리)
func (s *Store) GetStatus(user *shared.User, childID string) shared.SystemStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	var child *shared.Child

	if user != nil {
		// 1. 해당 Google 계정 사용자에게 소속된 자녀 탐색
		for _, c := range s.Data.Children {
			if c.UserID == user.ID {
				if childID == "" || c.ID == childID {
					child = c
					break
				}
			}
		}
		// 자녀가 없으면 자동 생성
		if child == nil {
			cid := fmt.Sprintf("child_%s", user.ID)
			childName := fmt.Sprintf("%s의 자녀", user.Name)
			if strings.TrimSpace(user.Name) == "" {
				childName = "우리 아이"
			}
			child = &shared.Child{
				ID:                cid,
				UserID:            user.ID,
				Name:              childName,
				DailyQuotaMinutes: 120,
				LastResetDate:     time.Now().Format("2006-01-02"),
				BedtimeEnabled:    false,
				BedtimeStart:      "22:00",
				BedtimeEnd:        "07:00",
				WhitelistedApps:   DefaultWhitelistedApps,
				UpdatedAt:         time.Now(),
			}
			s.Data.Children[cid] = child
			s.dirty = true
		}
	} else {
		// 비로그인 / 기본 하위 호환
		if childID == "" {
			childID = "child_1"
		}
		var exists bool
		child, exists = s.Data.Children[childID]
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
	}
	s.checkDailyReset(child)

	now := time.Now()
	var devList []shared.Device
	for _, dev := range s.Data.Devices {
		belongs := false
		if user != nil {
			// Google 사용자 격리: 자신의 자녀 기기이거나 자신의 유저ID에 바인딩된 기기
			if dev.UserID == user.ID || dev.ChildID == child.ID {
				belongs = true
			}
		} else {
			if dev.ChildID == child.ID {
				belongs = true
			}
		}

		if belongs {
			dCopy := *dev
			if now.Sub(dCopy.LastSeen) > 60*time.Second {
				dCopy.Status = shared.StatusOffline
			}
			devList = append(devList, dCopy)
		}
	}

	// ⭐️ 연동기기 순서 고정 (대시보드에서 순서가 바뀌지 않도록 정렬)
	// 1) 등록일시(CreatedAt)가 다르면 등록일시 오름차순 (먼저 등록된 기기 우선)
	// 2) 등록일시가 같거나 미지정이면 기기 유형 우선순위 (스마트폰 -> 태블릿 -> PC)
	// 3) 그 외 ID 오름차순
	sort.Slice(devList, func(i, j int) bool {
		if !devList[i].CreatedAt.IsZero() && !devList[j].CreatedAt.IsZero() {
			if !devList[i].CreatedAt.Equal(devList[j].CreatedAt) {
				return devList[i].CreatedAt.Before(devList[j].CreatedAt)
			}
		}
		typePriority := func(t shared.DeviceType) int {
			switch t {
			case shared.DeviceAndroidPhone:
				return 1
			case shared.DeviceAndroidTablet:
				return 2
			case shared.DeviceWindowsPC:
				return 3
			default:
				return 4
			}
		}
		p1, p2 := typePriority(devList[i].Type), typePriority(devList[j].Type)
		if p1 != p2 {
			return p1 < p2
		}
		return devList[i].ID < devList[j].ID
	})

	return shared.SystemStatus{
		User:    user,
		Child:   child,
		Devices: devList,
	}
}

// ResolveChildID 사용자의 자녀 ID를 반환 (인증된 사용자의 자녀 ID 우선 조회)
func (s *Store) ResolveChildID(user *shared.User, explicitID string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if user != nil {
		if explicitID != "" {
			if c, exists := s.Data.Children[explicitID]; exists && (c.UserID == user.ID || c.UserID == "") {
				return explicitID
			}
		}
		for _, c := range s.Data.Children {
			if c.UserID == user.ID {
				return c.ID
			}
		}
		// 자녀 계정이 없는 경우 신규 생성
		cid := fmt.Sprintf("child_%s", user.ID)
		childName := fmt.Sprintf("%s의 자녀", user.Name)
		if strings.TrimSpace(user.Name) == "" {
			childName = "우리 아이"
		}
		c := &shared.Child{
			ID:                cid,
			UserID:            user.ID,
			Name:              childName,
			DailyQuotaMinutes: 120,
			LastResetDate:     time.Now().Format("2006-01-02"),
			BedtimeEnabled:    false,
			BedtimeStart:      "22:00",
			BedtimeEnd:        "07:00",
			WhitelistedApps:   DefaultWhitelistedApps,
			UpdatedAt:         time.Now(),
		}
		if s.Data.Children == nil {
			s.Data.Children = make(map[string]*shared.Child)
		}
		s.Data.Children[cid] = c
		s.dirty = true
		return cid
	}

	if explicitID != "" {
		return explicitID
	}
	return "child_1"
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

// parseGoogleJWT Google Identity Services ID 토큰 디코딩
func parseGoogleJWT(credential string) (sub, email, name, picture string, err error) {
	parts := strings.Split(credential, ".")
	if len(parts) < 2 {
		return "", "", "", "", fmt.Errorf("올바르지 않은 JWT 토큰 형식입니다")
	}

	payloadSeg := parts[1]
	switch len(payloadSeg) % 4 {
	case 2:
		payloadSeg += "=="
	case 3:
		payloadSeg += "="
	}

	decoded, err := base64.URLEncoding.DecodeString(payloadSeg)
	if err != nil {
		decoded, err = base64.RawURLEncoding.DecodeString(parts[1])
		if err != nil {
			return "", "", "", "", fmt.Errorf("JWT 디코딩 실패: %v", err)
		}
	}

	var claims struct {
		Sub     string `json:"sub"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return "", "", "", "", fmt.Errorf("토큰 정보 파싱 실패: %v", err)
	}

	if claims.Sub == "" && claims.Email == "" {
		return "", "", "", "", fmt.Errorf("유효한 Google 사용자 식별자가 없습니다")
	}

	sub = claims.Sub
	if sub == "" {
		sub = fmt.Sprintf("google_%x", sha256.Sum256([]byte(claims.Email)))[:16]
	}
	return sub, claims.Email, claims.Name, claims.Picture, nil
}

// AuthenticateGoogle Google OAuth 로그인 처리 및 세션 발급
func (s *Store) AuthenticateGoogle(req shared.GoogleAuthRequest) (string, *shared.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var sub, email, name, picture string
	var err error

	if req.Credential != "" {
		sub, email, name, picture, err = parseGoogleJWT(req.Credential)
		if err != nil {
			return "", nil, err
		}
	} else if req.Email != "" {
		email = strings.TrimSpace(req.Email)
		sub = fmt.Sprintf("google_%x", sha256.Sum256([]byte(email)))[:16]
		name = strings.TrimSpace(req.Name)
		if name == "" {
			name = strings.Split(email, "@")[0]
		}
		picture = req.Picture
	} else {
		return "", nil, fmt.Errorf("Google 계정 정보가 누락되었습니다")
	}

	if s.Data.Users == nil {
		s.Data.Users = make(map[string]*shared.User)
	}

	user, exists := s.Data.Users[sub]
	if !exists {
		user = &shared.User{
			ID:        sub,
			Email:     email,
			Name:      name,
			Picture:   picture,
			CreatedAt: time.Now(),
		}
		s.Data.Users[sub] = user

		// 신규 부모 사용자를 위한 격리된 자녀 계정 자동 생성
		childID := fmt.Sprintf("child_%s", sub)
		childName := fmt.Sprintf("%s의 자녀", name)
		if strings.TrimSpace(name) == "" {
			childName = "우리 아이"
		}
		newChild := &shared.Child{
			ID:                childID,
			UserID:            sub,
			Name:              childName,
			DailyQuotaMinutes: 120,
			BonusMinutes:      0,
			UsedSeconds:       0,
			ForceLocked:       false,
			LastResetDate:     time.Now().Format("2006-01-02"),
			UpdatedAt:         time.Now(),
			BedtimeEnabled:    false,
			BedtimeStart:      "22:00",
			BedtimeEnd:        "07:00",
			WhitelistedApps:   DefaultWhitelistedApps,
		}
		s.Data.Children[childID] = newChild
		s.dirty = true
	} else {
		if name != "" {
			user.Name = name
		}
		if picture != "" {
			user.Picture = picture
		}
	}

	tokenBytes := make([]byte, 24)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", nil, fmt.Errorf("세션 토큰 생성 실패: %v", err)
	}
	token := hex.EncodeToString(tokenBytes)
	s.sessions[token] = user

	return token, user, nil
}

// AuthenticateParent 부모 로그인 인증 및 세션 토큰 발급
func (s *Store) AuthenticateParent(username, password string) (string, *shared.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	expectedHash, exists := s.Data.ParentUsers[username]
	if !exists {
		return "", nil, fmt.Errorf("존재하지 않는 부모 계정입니다")
	}

	if expectedHash != hashPassword(password) {
		return "", nil, fmt.Errorf("비밀번호가 올바르지 않습니다")
	}

	user, userExists := s.Data.Users[username]
	if !userExists {
		user = &shared.User{
			ID:        username,
			Email:     username + "@paxfamilia.com",
			Name:      username + " (부모)",
			CreatedAt: time.Now(),
		}
		if s.Data.Users == nil {
			s.Data.Users = make(map[string]*shared.User)
		}
		s.Data.Users[username] = user
	}

	tokenBytes := make([]byte, 24)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", nil, fmt.Errorf("세션 토큰 생성 실패: %v", err)
	}
	token := hex.EncodeToString(tokenBytes)
	s.sessions[token] = user
	return token, user, nil
}

// ValidateSession 부모 세션 토큰 유효성 검사
func (s *Store) ValidateSession(token string) (*shared.User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if token == "" {
		return nil, false
	}
	user, exists := s.sessions[token]
	return user, exists
}

// InvalidateSession 부모 세션 만료 (로그아웃)
func (s *Store) InvalidateSession(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
}

// CreatePairingCode 부모 대시보드에서 6자리 기기 등록 코드 발급 (유효시간 15분)
func (s *Store) CreatePairingCode(user *shared.User, childID string) (*shared.PairingCodeInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	userID := ""
	if user != nil {
		userID = user.ID
		if childID == "" {
			for _, c := range s.Data.Children {
				if c.UserID == user.ID {
					childID = c.ID
					break
				}
			}
		}
	}
	if childID == "" {
		childID = "child_1"
	}
	if _, exists := s.Data.Children[childID]; !exists {
		return nil, fmt.Errorf("자녀 ID(%s)를 찾을 수 없습니다", childID)
	}

	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return nil, err
	}
	code := fmt.Sprintf("%06d", n.Int64()+100000)
	expiresAt := time.Now().Add(15 * time.Minute)

	codeInfo := &shared.PairingCodeInfo{
		Code:      code,
		UserID:    userID,
		ChildID:   childID,
		ExpiresAt: expiresAt,
	}

	if s.Data.PairingCodes == nil {
		s.Data.PairingCodes = make(map[string]*shared.PairingCodeInfo)
	}
	s.Data.PairingCodes[code] = codeInfo
	s.dirty = true

	return codeInfo, nil
}

// RegisterDevice 6자리 코드로 실제 기기 등록
func (s *Store) RegisterDevice(req shared.DeviceRegisterRequest) (*shared.DeviceRegisterResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	code := strings.TrimSpace(req.PairingCode)
	codeInfo, exists := s.Data.PairingCodes[code]
	if !exists {
		return nil, fmt.Errorf("유효하지 않거나 존재하지 않는 6자리 페어링 코드입니다")
	}

	if time.Now().After(codeInfo.ExpiresAt) {
		delete(s.Data.PairingCodes, code)
		s.dirty = true
		return nil, fmt.Errorf("페어링 코드가 만료되었습니다. 대시보드에서 새로 발급받으세요")
	}

	child, childExists := s.Data.Children[codeInfo.ChildID]
	if !childExists {
		return nil, fmt.Errorf("해당 코드에 연결된 자녀 계정이 없습니다")
	}

	devType := req.DeviceType
	if devType == "" {
		devType = shared.DeviceAndroidPhone
	}
	devName := strings.TrimSpace(req.DeviceName)
	if devName == "" {
		devName = fmt.Sprintf("새 %s 기기", devType)
	}

	devID := fmt.Sprintf("dev_%s_%d", devType, time.Now().Unix()%1000000)

	newDev := &shared.Device{
		ID:           devID,
		ChildID:      child.ID,
		UserID:       codeInfo.UserID, // ⭐️ 부모 Google 계정과 바인딩
		Name:         devName,
		Type:         devType,
		Status:       shared.StatusOnline,
		CurrentApp:   "연동 완료",
		BatteryLevel: 100,
		LastSeen:     time.Now(),
		UsedSeconds:  0,
		CreatedAt:    time.Now(),
	}

	if s.Data.Devices == nil {
		s.Data.Devices = make(map[string]*shared.Device)
	}
	s.Data.Devices[devID] = newDev

	// 코드 1회성 소모
	delete(s.Data.PairingCodes, code)
	s.dirty = true

	return &shared.DeviceRegisterResponse{
		Success:   true,
		DeviceID:  devID,
		ChildID:   child.ID,
		ChildName: child.Name,
	}, nil
}

// DeleteDevice 기기 등록 해제/삭제
func (s *Store) DeleteDevice(user *shared.User, deviceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dev, exists := s.Data.Devices[deviceID]
	if !exists {
		return fmt.Errorf("해당 기기(%s)가 존재하지 않습니다", deviceID)
	}

	if user != nil {
		if dev.UserID != "" && dev.UserID != user.ID {
			return fmt.Errorf("해당 기기를 삭제할 권한이 없습니다")
		}
	}

	delete(s.Data.Devices, deviceID)
	s.dirty = true
	return nil
}
