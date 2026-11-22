package main

import (
	"encoding/json"
	"net/http"

	"pax-familia/shared"
)

type Server struct {
	store *Store
}

func NewServer(store *Store) *Server {
	return &Server{store: store}
}

// HandleHeartbeat 클라이언트 Heartbeat 수신
func (s *Server) HandleHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req shared.HeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	resp := s.store.ProcessHeartbeat(req)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// HandleStatus 전체 상태 조회
func (s *Server) HandleStatus(w http.ResponseWriter, r *http.Request) {
	childID := r.URL.Query().Get("child_id")
	if childID == "" {
		childID = "child_1"
	}

	status := s.store.GetStatus(childID)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status)
}

// HandleLock 즉시 잠금 / 잠금 해제 토글
func (s *Server) HandleLock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		ChildID string `json:"child_id"`
		Lock    bool   `json:"lock"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if payload.ChildID == "" {
		payload.ChildID = "child_1"
	}

	s.store.SetForceLock(payload.ChildID, payload.Lock)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"success":true}`))
}

// HandleBonus 보너스 시간 부여
func (s *Server) HandleBonus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		ChildID string `json:"child_id"`
		Minutes int    `json:"minutes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if payload.ChildID == "" {
		payload.ChildID = "child_1"
	}

	s.store.AddBonus(payload.ChildID, payload.Minutes)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"success":true}`))
}

// HandleQuota 일일 기본 한도 수정
func (s *Server) HandleQuota(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		ChildID string `json:"child_id"`
		Minutes int    `json:"minutes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if payload.ChildID == "" {
		payload.ChildID = "child_1"
	}

	if payload.Minutes <= 0 {
		payload.Minutes = 60
	}

	s.store.SetDailyQuota(payload.ChildID, payload.Minutes)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"success":true}`))
}

// HandleResetToday 오늘 사용량 초기화 (테스트 및 편의용)
func (s *Server) HandleResetToday(w http.ResponseWriter, r *http.Request) {
	childID := r.URL.Query().Get("child_id")
	if childID == "" {
		childID = "child_1"
	}
	s.store.ResetToday(childID)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"success":true}`))
}
