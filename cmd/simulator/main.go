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
	flag.Parse()

	log.Println("==========================================================")
	log.Println("🎮 PaxFamilia 3대 기기 통합 가상 시뮬레이터 시작")
	log.Println("   - 기기 1: 안드로이드 폰 (갤럭시 S24)")
	log.Println("   - 기기 2: 안드로이드 태블릿 (갤럭시 탭 S9)")
	log.Println("   - 기기 3: 윈도우 PC (거실 데스크톱)")
	log.Printf("📡 서버 주소: %s", *serverURL)
	log.Println("==========================================================")

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
