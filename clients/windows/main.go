package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"pax-familia/shared"
)

type AgentConfig struct {
	ServerURL  string
	DeviceID   string
	ChildID    string
	DeviceName string
	Interval   time.Duration
}

type WindowsAgent struct {
	config     AgentConfig
	httpClient *http.Client
	platform   PlatformOps
	isLocked   bool
	lastTick   time.Time
}

func NewWindowsAgent(cfg AgentConfig) *WindowsAgent {
	return &WindowsAgent{
		config:     cfg,
		httpClient: &http.Client{Timeout: 5 * time.Second},
		platform:   NewPlatformOps(),
		lastTick:   time.Now(),
	}
}

func (a *WindowsAgent) Run() {
	log.Printf("💻 PaxFamilia 윈도우 PC 에이전트 시작됨")
	log.Printf("📡 서버 주소: %s", a.config.ServerURL)
	log.Printf("🆔 기기 ID: %s (%s)", a.config.DeviceID, a.config.DeviceName)

	ticker := time.NewTicker(a.config.Interval)
	defer ticker.Stop()

	// 초기 하트비트
	a.tick()

	for range ticker.C {
		a.tick()
	}
}

func (a *WindowsAgent) tick() {
	now := time.Now()
	elapsedSeconds := int(now.Sub(a.lastTick).Seconds())
	a.lastTick = now

	// 1. 유휴 상태(Idle) 검사 (마우스/키보드 미입력 시간)
	idleDuration := a.platform.GetIdleDuration()
	isIdle := idleDuration > 2*time.Minute // 2분 이상 미입력 시 유휴로 판정

	activeSeconds := 0
	if !isIdle && !a.isLocked {
		activeSeconds = elapsedSeconds
		if activeSeconds > 60 {
			activeSeconds = 60
		}
	}

	// 2. 현재 활성 창 타이틀 수집
	currentApp := a.platform.GetForegroundWindowTitle()
	if currentApp == "" {
		if isIdle {
			currentApp = "(자리비움 / 유휴 상태)"
		} else {
			currentApp = "바탕화면"
		}
	}

	req := shared.HeartbeatRequest{
		DeviceID:      a.config.DeviceID,
		ChildID:       a.config.ChildID,
		DeviceName:    a.config.DeviceName,
		DeviceType:    shared.DeviceWindowsPC,
		ActiveSeconds: activeSeconds,
		CurrentApp:    currentApp,
		BatteryLevel:  -1,
		IsIdle:        isIdle,
	}

	resp, err := a.sendHeartbeat(req)
	if err != nil {
		log.Printf("⚠️ 서버 하트비트 전송 실패 (오프라인 모드 유지): %v", err)
		return
	}

	// 3. 잠금 처리 로직
	if resp.ShouldLock {
		if !a.isLocked {
			log.Printf("🔒 [잠금 신호 수신] 사유: %s | 화면 잠금을 실행합니다.", resp.LockReason)
			a.platform.ShowNotification("PaxFamilia 안내", fmt.Sprintf("사용시간이 종료되었습니다: %s", resp.LockReason))
			a.platform.LockScreen()
			a.isLocked = true
		} else {
			// 이미 잠긴 상태인데 자녀가 윈도우 비밀번호를 치고 로그인 시도 시 다시 잠금
			a.platform.LockScreen()
		}
	} else {
		if a.isLocked {
			log.Printf("🔓 [잠금 해제] 사용 시간이 충전되어 잠금이 해제되었습니다. 남은 시간: %d분", resp.RemainingMinutes)
			a.isLocked = false
		}
		if resp.WarningMessage != "" {
			log.Printf("⚠️ %s", resp.WarningMessage)
		}
		log.Printf("⏱️ [정상 동기화] 오늘 남은 통합 스크린타임: %d분 | 현재 창: %s", resp.RemainingMinutes, currentApp)
	}
}

func (a *WindowsAgent) sendHeartbeat(req shared.HeartbeatRequest) (*shared.HeartbeatResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := a.httpClient.Post(a.config.ServerURL+"/api/heartbeat", "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("서버 오류 상태코드: %d", resp.StatusCode)
	}

	var res shared.HeartbeatResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return &res, nil
}

func main() {
	serverURL := flag.String("server", "http://localhost:8080", "PaxFamilia 서버 URL")
	deviceID := flag.String("device-id", "dev_pc_1", "기기 고유 ID")
	deviceName := flag.String("name", "윈도우 PC", "기기 표시 이름")
	childID := flag.String("child-id", "child_1", "자녀 ID")
	intervalSec := flag.Int("interval", 15, "하트비트 주기(초)")
	flag.Parse()

	hostname, _ := os.Hostname()
	if *deviceName == "윈도우 PC" && hostname != "" {
		*deviceName = fmt.Sprintf("PC (%s)", hostname)
	}

	agent := NewWindowsAgent(AgentConfig{
		ServerURL:  *serverURL,
		DeviceID:   *deviceID,
		ChildID:    *childID,
		DeviceName: *deviceName,
		Interval:   time.Duration(*intervalSec) * time.Second,
	})

	agent.Run()
}
