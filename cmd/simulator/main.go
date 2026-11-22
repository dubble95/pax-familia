package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pax-familia/shared"
)

type SimDevice struct {
	ID       string
	Name     string
	Type     shared.DeviceType
	Apps     []string
	Active   bool
	LastSeen time.Time
}

func main() {
	serverURL := flag.String("server", "http://localhost:8080", "PaxFamilia 서버 URL")
	pairingCode := flag.String("code", "", "부모 대시보드 6자리 페어링 코드 (신규 기기 등록 테스트)")
	customName := flag.String("name", "신규 연동 테스트 기기", "등록할 기기 이름")
	customType := flag.String("type", "android_phone", "기기 종류 (android_phone, android_tablet, windows_pc)")
	flag.Parse()

	log.Println("==========================================================")
	log.Println("🎮 PaxFamilia 가상 시뮬레이터 가동")
	log.Printf("📡 서버 주소: %s", *serverURL)

	devices := []*SimDevice{
		{
			ID:     "dev_phone_1",
			Name:   "안드로이드 스마트폰",
			Type:   shared.DeviceAndroidPhone,
			Apps:   []string{"YouTube", "카카오톡", "브롤스타즈", "인스타그램"},
			Active: true,
		},
		{
			ID:     "dev_tablet_1",
			Name:   "안드로이드 태블릿",
			Type:   shared.DeviceAndroidTablet,
			Apps:   []string{"Roblox", "메가스터디 인강", "신비아파트 게임", "넷플릭스"},
			Active: false,
		},
		{
			ID:     "dev_pc_1",
			Name:   "윈도우 PC",
			Type:   shared.DeviceWindowsPC,
			Apps:   []string{"Minecraft", "Chrome - 나무위키", "Steam", "리그 오브 레전드"},
			Active: false,
		},
	}

	// 6자리 페어링 코드가 주어졌다면 신규 기기 등록 API 호출 테스트
	if *pairingCode != "" {
		regReq := shared.DeviceRegisterRequest{
			PairingCode: *pairingCode,
			DeviceName:  *customName,
			DeviceType:  shared.DeviceType(*customType),
		}
		data, _ := json.Marshal(regReq)
		resp, err := http.Post(*serverURL+"/api/devices/register", "application/json", bytes.NewReader(data))
		if err != nil {
			log.Fatalf("❌ 기기 등록 API 호출 실패: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			var errResp struct {
				Error string `json:"error"`
			}
			_ = json.NewDecoder(resp.Body).Decode(&errResp)
			log.Fatalf("❌ 기기 등록 거절됨: %s", errResp.Error)
		}

		var regResp shared.DeviceRegisterResponse
		_ = json.NewDecoder(resp.Body).Decode(&regResp)
		log.Printf("🎉 [기기 등록 성공] 자녀: %s | 기기 ID: %s | 기기명: %s", regResp.ChildName, regResp.DeviceID, *customName)

		// 새 등록 기기를 단독 시뮬레이션 목록으로 교체
		devices = []*SimDevice{
			{
				ID:     regResp.DeviceID,
				Name:   *customName,
				Type:   shared.DeviceType(*customType),
				Apps:   []string{"YouTube", "EBS 초등", "카카오톡", "게임"},
				Active: true,
			},
		}
	}
	log.Println("==========================================================")

	client := &http.Client{Timeout: 3 * time.Second}
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	// 종료 시그널 처리
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	step := 0
	for {
		select {
		case <-sigCh:
			log.Println("시뮬레이터를 종료합니다.")
			return
		case <-ticker.C:
			step++

			// 20초마다 활성 기기 전환 시뮬레이션 (폰 하다가 태블릿 켜고, PC 켜는 자녀 행동 모사)
			if step%7 == 0 {
				activeIdx := rand.Intn(len(devices))
				for i, dev := range devices {
					dev.Active = (i == activeIdx)
				}
				log.Printf("🔄 [기기 전환] 자녀가 기기를 바꿨습니다! -> 활성 기기: %s", devices[activeIdx].Name)
			}

			for _, dev := range devices {
				currentApp := "(화면 꺼짐 / 미사용)"
				activeSec := 0
				isIdle := !dev.Active

				if dev.Active {
					currentApp = dev.Apps[rand.Intn(len(dev.Apps))]
					activeSec = 3 // 3초 경과
				}

				req := shared.HeartbeatRequest{
					DeviceID:      dev.ID,
					ChildID:       "child_1",
					DeviceName:    dev.Name,
					DeviceType:    dev.Type,
					ActiveSeconds: activeSec,
					CurrentApp:    currentApp,
					BatteryLevel:  85,
					IsIdle:        isIdle,
				}

				data, _ := json.Marshal(req)
				resp, err := client.Post(*serverURL+"/api/heartbeat", "application/json", bytes.NewReader(data))
				if err != nil {
					log.Printf("⚠️ [%s] 서버 연결 실패: %v", dev.Name, err)
					continue
				}

				var res shared.HeartbeatResponse
				_ = json.NewDecoder(resp.Body).Decode(&res)
				resp.Body.Close()

				if dev.Active {
					lockText := "정상 동작"
					if res.ShouldLock {
						lockText = fmt.Sprintf("🔒 [LOCKED: %s]", res.LockReason)
					} else if res.IsWhitelisted {
						lockText = "📖 [학습 앱: 시간 차감 면제]"
					}
					log.Printf("📱 [%s] 앱: %-16s | 남은시간: %3d분 (%d초) | 상태: %s",
						dev.Name, currentApp, res.RemainingMinutes, res.RemainingSeconds, lockText)
				}
			}
		}
	}
}
