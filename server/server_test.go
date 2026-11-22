package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"pax-familia/shared"
)

func setupTestServer(t *testing.T) (*Server, func()) {
	tmpFile, err := os.CreateTemp("", "pax_test_*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	os.Remove(tmpPath)

	store, err := NewStore(tmpPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	srv := NewServer(store)
	cleanup := func() {
		store.Close()
		os.Remove(tmpPath)
	}
	return srv, cleanup
}

func TestUnauthenticatedAccess(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// GET /api/status without token should return 401
	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	w := httptest.NewRecorder()
	srv.HandleStatus(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestGoogleAuthAndMultiTenantIsolation(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// 1. Google Login for User A (철수 아빠)
	loginA := shared.GoogleAuthRequest{
		Email: "chulsoo.dad@gmail.com",
		Name:  "김철수 아빠",
	}
	loginABody, _ := json.Marshal(loginA)
	reqA := httptest.NewRequest(http.MethodPost, "/api/auth/google", bytes.NewReader(loginABody))
	wA := httptest.NewRecorder()
	srv.HandleGoogleLogin(wA, reqA)

	if wA.Code != http.StatusOK {
		t.Fatalf("User A login failed: code %d, body: %s", wA.Code, wA.Body.String())
	}
	var resA shared.GoogleAuthResponse
	if err := json.Unmarshal(wA.Body.Bytes(), &resA); err != nil || !resA.Success {
		t.Fatalf("failed to parse login response A: %v", err)
	}
	tokenA := resA.Token
	userA := resA.User

	// 2. Google Login for User B (영희 엄마)
	loginB := shared.GoogleAuthRequest{
		Email: "younghee.mom@gmail.com",
		Name:  "이영희 엄마",
	}
	loginBBody, _ := json.Marshal(loginB)
	reqB := httptest.NewRequest(http.MethodPost, "/api/auth/google", bytes.NewReader(loginBBody))
	wB := httptest.NewRecorder()
	srv.HandleGoogleLogin(wB, reqB)

	if wB.Code != http.StatusOK {
		t.Fatalf("User B login failed: code %d", wB.Code)
	}
	var resB shared.GoogleAuthResponse
	if err := json.Unmarshal(wB.Body.Bytes(), &resB); err != nil || !resB.Success {
		t.Fatalf("failed to parse login response B: %v", err)
	}
	tokenB := resB.Token
	userB := resB.User

	if userA.ID == userB.ID {
		t.Fatalf("User A and User B should have different user IDs: %s vs %s", userA.ID, userB.ID)
	}

	// 3. User A checks status: should have User A's child and 0 devices
	reqStatusA := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	reqStatusA.Header.Set("Authorization", "Bearer "+tokenA)
	wStatusA := httptest.NewRecorder()
	srv.HandleStatus(wStatusA, reqStatusA)

	if wStatusA.Code != http.StatusOK {
		t.Fatalf("User A status failed: code %d", wStatusA.Code)
	}
	var statusA shared.SystemStatus
	json.Unmarshal(wStatusA.Body.Bytes(), &statusA)
	if statusA.Child.UserID != userA.ID {
		t.Errorf("expected child UserID %s, got %s", userA.ID, statusA.Child.UserID)
	}
	if len(statusA.Devices) != 0 {
		t.Errorf("new user A should have 0 devices, got %d", len(statusA.Devices))
	}

	// 4. User A creates pairing code
	reqPair := httptest.NewRequest(http.MethodPost, "/api/devices/pairing-code", bytes.NewReader([]byte(`{}`)))
	reqPair.Header.Set("Authorization", "Bearer "+tokenA)
	wPair := httptest.NewRecorder()
	srv.HandleCreatePairingCode(wPair, reqPair)

	if wPair.Code != http.StatusOK {
		t.Fatalf("pairing code creation failed: %d, body: %s", wPair.Code, wPair.Body.String())
	}
	var pairResp struct {
		Success bool   `json:"success"`
		Code    string `json:"code"`
		ChildID string `json:"child_id"`
	}
	json.Unmarshal(wPair.Body.Bytes(), &pairResp)
	if !pairResp.Success || pairResp.Code == "" {
		t.Fatalf("invalid pairing code response: %+v", pairResp)
	}

	// 5. A device registers with User A's pairing code
	regReq := shared.DeviceRegisterRequest{
		PairingCode: pairResp.Code,
		DeviceName:  "철수 폰",
		DeviceType:  shared.DeviceAndroidPhone,
	}
	regBody, _ := json.Marshal(regReq)
	reqReg := httptest.NewRequest(http.MethodPost, "/api/devices/register", bytes.NewReader(regBody))
	wReg := httptest.NewRecorder()
	srv.HandleRegisterDevice(wReg, reqReg)

	if wReg.Code != http.StatusOK {
		t.Fatalf("device registration failed: %d, body: %s", wReg.Code, wReg.Body.String())
	}
	var regResp shared.DeviceRegisterResponse
	json.Unmarshal(wReg.Body.Bytes(), &regResp)
	if !regResp.Success || regResp.DeviceID == "" {
		t.Fatalf("invalid reg response: %+v", regResp)
	}
	deviceAID := regResp.DeviceID

	// 6. User A checks status again: should see 1 device ("철수 폰")
	reqStatusA2 := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	reqStatusA2.Header.Set("Authorization", "Bearer "+tokenA)
	wStatusA2 := httptest.NewRecorder()
	srv.HandleStatus(wStatusA2, reqStatusA2)

	var statusA2 shared.SystemStatus
	json.Unmarshal(wStatusA2.Body.Bytes(), &statusA2)
	if len(statusA2.Devices) != 1 || statusA2.Devices[0].ID != deviceAID {
		t.Fatalf("expected User A to have 1 device with ID %s, got: %+v", deviceAID, statusA2.Devices)
	}

	// 7. ⭐️ User B checks status: MUST NOT see User A's device!
	reqStatusB := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	reqStatusB.Header.Set("Authorization", "Bearer "+tokenB)
	wStatusB := httptest.NewRecorder()
	srv.HandleStatus(wStatusB, reqStatusB)

	var statusB shared.SystemStatus
	json.Unmarshal(wStatusB.Body.Bytes(), &statusB)
	if len(statusB.Devices) != 0 {
		t.Fatalf("User B MUST NOT see User A's devices! Found %d devices: %+v", len(statusB.Devices), statusB.Devices)
	}
	if statusB.Child.UserID != userB.ID {
		t.Errorf("expected User B's child UserID %s, got %s", userB.ID, statusB.Child.UserID)
	}

	// 8. ⭐️ User B attempts to delete User A's device: MUST BE REJECTED
	delReq := bytes.NewReader([]byte(`{"device_id":"` + deviceAID + `"}`))
	reqDel := httptest.NewRequest(http.MethodPost, "/api/devices/delete", delReq)
	reqDel.Header.Set("Authorization", "Bearer "+tokenB)
	wDel := httptest.NewRecorder()
	srv.HandleDeleteDevice(wDel, reqDel)

	if wDel.Code == http.StatusOK {
		t.Fatalf("User B was able to delete User A's device! Security breach!")
	}
}

func TestDownloadRedirects(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Windows EXE
	reqWin := httptest.NewRequest(http.MethodGet, "/download/windows", nil)
	wWin := httptest.NewRecorder()
	srv.HandleDownloadWindows(wWin, reqWin)
	if wWin.Code != http.StatusFound || wWin.Header().Get("Location") != DownloadWindowsExe {
		t.Errorf("expected 302 redirect to %s, got code %d, loc: %s", DownloadWindowsExe, wWin.Code, wWin.Header().Get("Location"))
	}

	// Android APK
	reqAnd := httptest.NewRequest(http.MethodGet, "/download/android", nil)
	wAnd := httptest.NewRecorder()
	srv.HandleDownloadAndroid(wAnd, reqAnd)
	if wAnd.Code != http.StatusFound || wAnd.Header().Get("Location") != DownloadAndroidAPK {
		t.Errorf("expected 302 redirect to %s, got code %d, loc: %s", DownloadAndroidAPK, wAnd.Code, wAnd.Header().Get("Location"))
	}

	// Windows ZIP
	reqZip := httptest.NewRequest(http.MethodGet, "/download/windows-zip", nil)
	wZip := httptest.NewRecorder()
	srv.HandleDownloadWindowsZip(wZip, reqZip)
	if wZip.Code != http.StatusFound || wZip.Header().Get("Location") != DownloadWindowsZip {
		t.Errorf("expected 302 redirect to %s, got code %d, loc: %s", DownloadWindowsZip, wZip.Code, wZip.Header().Get("Location"))
	}
}
