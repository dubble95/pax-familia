package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

//go:embed web/*
var webFS embed.FS

// withCORS 모든 API에 CORS 헤더를 추가하는 미들웨어
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	port := flag.Int("port", 8080, "PaxFamilia 서버 포트")
	dataPath := flag.String("data", "data/pax_state.json", "상태 저장 JSON 파일 경로")
	flag.Parse()

	// 클라우드 PaaS 호스팅 환경 변수 지원 (Render, Fly.io, Cloud Run 등)
	if envPort := os.Getenv("PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil {
			*port = p
		}
	}
	if envData := os.Getenv("DATA_PATH"); envData != "" {
		*dataPath = envData
	}

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

	handler := withCORS(mux)

	addr := fmt.Sprintf("0.0.0.0:%d", *port)
	httpServer := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	log.Printf("==================================================")
	log.Printf("🕊️ PaxFamilia (가정의 평화) 서버가 가동되었습니다!")
	log.Printf("🔗 부모 대시보드 웹: http://localhost:%d", *port)
	log.Printf("📡 기기 Heartbeat API: http://localhost:%d/api/heartbeat", *port)
	log.Printf("📁 데이터 파일 위치: %s", *dataPath)
	log.Printf("==================================================")

	// Graceful Shutdown 처리 (Fly.io / Docker 재시작 시 데이터 보존)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("🛑 종료 시그널 수신: 안전한 서버 종료 절차를 시작합니다...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(ctx)
		store.Close()
		os.Exit(0)
	}()

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("서버 실행 오류: %v", err)
	}
}
