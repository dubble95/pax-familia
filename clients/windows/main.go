package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

	currentInterval := a.config.Interval
	ticker := time.NewTicker(currentInterval)
	defer ticker.Stop()

	// 초기 하트비트
	if next := a.tick(); next > 0 {
		currentInterval = next
		ticker.Reset(currentInterval)
	}

	for range ticker.C {
		if next := a.tick(); next > 0 && next != currentInterval {
			currentInterval = next
			ticker.Reset(currentInterval)
			log.Printf("🔄 [적응형 동적 폴링] 다음 동기화 간격: %v", currentInterval)
		}
	}
}

func (a *WindowsAgent) tick() time.Duration {
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
		return a.config.Interval
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
		if resp.IsWhitelisted {
			log.Printf("📖 [학습 앱 감지] '%s' 사용 중 - 스크린타임 차감 제외", currentApp)
		} else if resp.WarningMessage != "" {
			log.Printf("⚠️ %s", resp.WarningMessage)
		}
		log.Printf("⏱️ [정상 동기화] 오늘 남은 통합 스크린타임: %d분 | 현재 창: %s", resp.RemainingMinutes, currentApp)
	}

	if resp.NextCheckSeconds > 0 {
		return time.Duration(resp.NextCheckSeconds) * time.Second
	}
	return a.config.Interval
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

type DiskConfig struct {
	ServerURL  string `json:"server_url"`
	DeviceID   string `json:"device_id"`
	ChildID    string `json:"child_id"`
	DeviceName string `json:"device_name"`
}

func getConfigPath() string {
	// 1. 현재 디렉토리 우선
	if _, err := os.Stat("pax-config.json"); err == nil {
		return "pax-config.json"
	}
	// 2. Windows %LocalAppData%\PaxFamilia\pax-config.json
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData != "" {
		p := filepath.Join(localAppData, "PaxFamilia", "pax-config.json")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	// 3. Windows %ProgramData%\PaxFamilia\pax-config.json
	programData := os.Getenv("ProgramData")
	if programData != "" {
		p := filepath.Join(programData, "PaxFamilia", "pax-config.json")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "pax-config.json"
}

func registerWithCode(serverURL, code, deviceName string) (*DiskConfig, error) {
	reqBody := shared.DeviceRegisterRequest{
		PairingCode: code,
		DeviceName:  deviceName,
		DeviceType:  shared.DeviceWindowsPC,
	}
	data, _ := json.Marshal(reqBody)
	resp, err := http.Post(serverURL+"/api/devices/register", "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("서버 연결 실패: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("기기 등록 실패: %s", errResp.Error)
	}

	var regResp shared.DeviceRegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&regResp); err != nil {
		return nil, err
	}

	cfg := &DiskConfig{
		ServerURL:  serverURL,
		DeviceID:   regResp.DeviceID,
		ChildID:    regResp.ChildID,
		DeviceName: deviceName,
	}

	cfgPath := getConfigPath()
	_ = os.MkdirAll(filepath.Dir(cfgPath), 0755)
	cfgBytes, _ := json.MarshalIndent(cfg, "", "  ")
	_ = os.WriteFile(cfgPath, cfgBytes, 0644)
	log.Printf("🎉 [기기 등록 완료] 자녀 '%s'에 윈도우 PC가 연동되었습니다! (ID: %s)", regResp.ChildName, regResp.DeviceID)
	return cfg, nil
}

func main() {
	serverURL := flag.String("server", "http://localhost:8080", "PaxFamilia 서버 URL")
	pairingCode := flag.String("code", "", "부모 대시보드에서 발급받은 6자리 페어링 코드")
	deviceID := flag.String("device-id", "", "기기 고유 ID (생략 시 config 사용)")
	deviceName := flag.String("name", "윈도우 PC", "기기 표시 이름")
	childID := flag.String("child-id", "", "자녀 ID (생략 시 config 사용)")
	intervalSec := flag.Int("interval", 15, "하트비트 주기(초)")
	flag.Parse()

	hostname, _ := os.Hostname()
	if *deviceName == "윈도우 PC" && hostname != "" {
		*deviceName = fmt.Sprintf("PC (%s)", hostname)
	}

	cfgPath := getConfigPath()
	var loadedCfg *DiskConfig

	// 1. 이미 저장된 설정 파일이 있는지 확인
	if data, err := os.ReadFile(cfgPath); err == nil {
		var c DiskConfig
		if json.Unmarshal(data, &c) == nil && c.DeviceID != "" {
			loadedCfg = &c
			log.Printf("📂 로컬 설정 로드 완료: %s (기기: %s)", cfgPath, c.DeviceName)
		}
	}

	// 2. 페어링 코드가 명시된 경우 신규 등록 수행
	if *pairingCode != "" {
		newCfg, err := registerWithCode(*serverURL, *pairingCode, *deviceName)
		if err != nil {
			log.Fatalf("❌ 기기 등록 오류: %v", err)
		}
		loadedCfg = newCfg
	} else if loadedCfg == nil && *deviceID == "" {
		// 설정도 없고 device-id도 없고 코드도 없으면 대화형 입력 확인
		log.Println("==================================================")
		log.Println("💻 PaxFamilia 윈도우 PC 에이전트 초기 연동 마법사")
		log.Println("==================================================")
		fmt.Printf("부모 대시보드에서 발급받은 6자리 페어링 코드를 입력하세요: ")
		reader := bufio.NewReader(os.Stdin)
		code, _ := reader.ReadString('\n')
		code = strings.TrimSpace(code)
		if code != "" {
			newCfg, err := registerWithCode(*serverURL, code, *deviceName)
			if err != nil {
				log.Fatalf("❌ 기기 등록 오류: %v", err)
			}
			loadedCfg = newCfg
		}
	}

	// 3. 최종 설정 병합
	finalServer := *serverURL
	finalDevID := "dev_pc_1"
	finalChildID := "child_1"
	finalDevName := *deviceName

	if loadedCfg != nil {
		if loadedCfg.ServerURL != "" {
			finalServer = loadedCfg.ServerURL
		}
		if loadedCfg.DeviceID != "" {
			finalDevID = loadedCfg.DeviceID
		}
		if loadedCfg.ChildID != "" {
			finalChildID = loadedCfg.ChildID
		}
		if loadedCfg.DeviceName != "" {
			finalDevName = loadedCfg.DeviceName
		}
	}

	if *deviceID != "" {
		finalDevID = *deviceID
	}
	if *childID != "" {
		finalChildID = *childID
	}

	agent := NewWindowsAgent(AgentConfig{
		ServerURL:  finalServer,
		DeviceID:   finalDevID,
		ChildID:    finalChildID,
		DeviceName: finalDevName,
		Interval:   time.Duration(*intervalSec) * time.Second,
	})

	agent.Run()
}
