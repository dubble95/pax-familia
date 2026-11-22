package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

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

// HandleBedtime 취침 모드 설정
func (s *Server) HandleBedtime(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		ChildID string `json:"child_id"`
		Enabled bool   `json:"enabled"`
		Start   string `json:"start"`
		End     string `json:"end"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if payload.ChildID == "" {
		payload.ChildID = "child_1"
	}

	s.store.SetBedtime(payload.ChildID, payload.Enabled, payload.Start, payload.End)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"success":true}`))
}

// HandleWhitelist 학습용 화이트리스트 관리
func (s *Server) HandleWhitelist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		ChildID string   `json:"child_id"`
		Action  string   `json:"action"` // "add", "remove", "set"
		App     string   `json:"app"`
		Apps    []string `json:"apps"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if payload.ChildID == "" {
		payload.ChildID = "child_1"
	}

	switch payload.Action {
	case "add":
		s.store.AddWhitelistedApp(payload.ChildID, payload.App)
	case "remove":
		s.store.RemoveWhitelistedApp(payload.ChildID, payload.App)
	case "set":
		s.store.SetWhitelistedApps(payload.ChildID, payload.Apps)
	default:
		if payload.App != "" {
			s.store.AddWhitelistedApp(payload.ChildID, payload.App)
		}
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"success":true}`))
}

// HandleResetToday 오늘 사용량 초기화
func (s *Server) HandleResetToday(w http.ResponseWriter, r *http.Request) {
	childID := r.URL.Query().Get("child_id")
	if childID == "" {
		childID = "child_1"
	}
	s.store.ResetToday(childID)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"success":true}`))
}

// HandleTelegram 텔레그램 봇 Webhook 핸들러
func (s *Server) HandleTelegram(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		http.Error(w, `{"error":"TELEGRAM_BOT_TOKEN not configured"}`, http.StatusBadRequest)
		return
	}

	var update struct {
		Message *struct {
			Chat struct {
				ID int64 `json:"id"`
			} `json:"chat"`
			Text string `json:"text"`
		} `json:"message"`
		CallbackQuery *struct {
			ID      string `json:"id"`
			Message struct {
				Chat struct {
					ID int64 `json:"id"`
				} `json:"chat"`
			} `json:"message"`
			Data string `json:"data"`
		} `json:"callback_query"`
	}

	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var chatID int64
	var text string

	if update.Message != nil {
		chatID = update.Message.Chat.ID
		text = strings.TrimSpace(update.Message.Text)
	} else if update.CallbackQuery != nil {
		chatID = update.CallbackQuery.Message.Chat.ID
		text = strings.TrimSpace(update.CallbackQuery.Data)

		// Answer callback query
		go func(cbID string) {
			ansURL := fmt.Sprintf("https://api.telegram.org/bot%s/answerCallbackQuery", token)
			body, _ := json.Marshal(map[string]string{"callback_query_id": cbID, "text": "명령 처리 완료!"})
			_, _ = http.Post(ansURL, "application/json", bytes.NewReader(body))
		}(update.CallbackQuery.ID)
	}

	if chatID == 0 {
		w.WriteHeader(http.StatusOK)
		return
	}

	// 상태 조회 및 명령 처리
	status := s.store.GetStatus("child_1")
	child := status.Child
	totalAllowed := (child.DailyQuotaMinutes + child.BonusMinutes) * 60
	remSec := totalAllowed - child.UsedSeconds
	if remSec < 0 {
		remSec = 0
	}
	remMin := (remSec + 59) / 60

	replyText := ""
	switch {
	case strings.HasPrefix(text, "/lock") || text == "lock":
		s.store.SetForceLock("child_1", true)
		replyText = "🛑 **모든 자녀 기기를 즉시 잠갔습니다.**"
	case strings.HasPrefix(text, "/unlock") || text == "unlock":
		s.store.SetForceLock("child_1", false)
		replyText = fmt.Sprintf("🟢 **잠금이 해제되었습니다.** (남은 시간: %d분)", remMin)
	case strings.HasPrefix(text, "/bonus") || strings.HasPrefix(text, "bonus_"):
		addMins := 30
		if strings.Contains(text, "15") {
			addMins = 15
		} else if strings.Contains(text, "60") {
			addMins = 60
		}
		s.store.AddBonus("child_1", addMins)
		s.store.SetForceLock("child_1", false)
		newRem := remMin + addMins
		replyText = fmt.Sprintf("🎁 **보너스 %d분이 지급되었습니다!**\n현재 남은 시간: **%d분**", addMins, newRem)
	default:
		lockState := "🟢 정상 이용 가능"
		if child.ForceLocked {
			lockState = "🔴 즉시 잠김"
		} else if remSec <= 0 {
			lockState = "🔒 시간 소진으로 잠김"
		}
		replyText = fmt.Sprintf("🕊️ **PaxFamilia 현재 현황**\n\n"+
			"👤 자녀: **%s**\n"+
			"⏱️ 남은 시간: **%d분** (기본 %d분 + 보너스 %d분)\n"+
			"⏳ 사용 시간: **%d분**\n"+
			"🛡️ 상태: **%s**\n\n"+
			"아래 버튼으로 즉시 제어할 수 있습니다:",
			child.Name, remMin, child.DailyQuotaMinutes, child.BonusMinutes, child.UsedSeconds/60, lockState)
	}

	// 텔레그램 메시지 전송
	go sendTelegram(token, chatID, replyText, child.ForceLocked)

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"ok":true}`))
}

func sendTelegram(token string, chatID int64, text string, isLocked bool) {
	tgURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	lockBtnText := "🛑 즉시 올스톱 잠금"
	lockBtnData := "lock"
	if isLocked {
		lockBtnText = "🔓 잠금 해제"
		lockBtnData = "unlock"
	}

	payload := map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "Markdown",
		"reply_markup": map[string]interface{}{
			"inline_keyboard": [][]map[string]string{
				{
					{"text": lockBtnText, "callback_data": lockBtnData},
					{"text": "🎁 +15분", "callback_data": "bonus_15"},
					{"text": "🎁 +30분", "callback_data": "bonus_30"},
				},
			},
		},
	}

	body, _ := json.Marshal(payload)
	resp, err := http.Post(tgURL, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("⚠️ 텔레그램 메시지 전송 오류: %v", err)
		return
	}
	defer resp.Body.Close()
}
