package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
)

//go:embed web/*
var webFS embed.FS

func main() {
	port := flag.Int("port", 8080, "PaxFamilia 서버 포트")
	dataPath := flag.String("data", "data/pax_state.json", "상태 저장 JSON 파일 경로")
	flag.Parse()

	log.Println("🕊️ PaxFamilia 서버 초기화 중...")

	store, err := NewStore(*dataPath)
	if err != nil {
		log.Fatalf("저장소 로드 실패: %v", err)
	}

	srv := NewServer(store)

	mux := http.NewServeMux()

	// API 엔드포인트
	mux.HandleFunc("/api/heartbeat", srv.HandleHeartbeat)
	mux.HandleFunc("/api/status", srv.HandleStatus)
	mux.HandleFunc("/api/lock", srv.HandleLock)
	mux.HandleFunc("/api/bonus", srv.HandleBonus)
	mux.HandleFunc("/api/quota", srv.HandleQuota)
	mux.HandleFunc("/api/reset", srv.HandleResetToday)

	// 정적 웹 대시보드 서빙
	subFS, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatalf("웹 리소스 로드 실패: %v", err)
	}
	mux.Handle("/", http.FileServer(http.FS(subFS)))

	addr := fmt.Sprintf("0.0.0.0:%d", *port)
	log.Printf("==================================================")
	log.Printf("🕊️ PaxFamilia (가정의 평화) 서버가 가동되었습니다!")
	log.Printf("🔗 부모 대시보드 웹: http://localhost:%d", *port)
	log.Printf("📡 기기 Heartbeat API: http://localhost:%d/api/heartbeat", *port)
	log.Printf("==================================================")

	if err := http.ListenAndServe(addr, mux); err != nil && err != http.ErrServerClosed {
		log.Fatalf("서버 실행 오류: %v", err)
	}
}
