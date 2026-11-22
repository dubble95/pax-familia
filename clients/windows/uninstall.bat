@echo off
chcp 65001 > nul
title PaxFamilia 윈도우 에이전트 제거

echo ==========================================================
echo   🕊️ PaxFamilia (가정의 평화) 윈도우 에이전트 제거
echo ==========================================================
echo.

:: 관리자 권한 확인
net session >nul 2>&1
if %errorlevel% neq 0 (
    powershell -Command "Start-Process '%~f0' -Verb RunAs"
    exit /b
)

echo [*] 실행 중인 에이전트 프로세스 종료 중...
taskkill /F /IM pax-winagent.exe >nul 2>&1

echo [*] 자동 시작 작업 스케줄러 삭제 중...
schtasks /Delete /TN "PaxFamiliaAgent" /F >nul 2>&1

echo [*] 레지스트리 시작 항목 삭제 중...
reg delete "HKCU\Software\Microsoft\Windows\CurrentVersion\Run" /v "PaxFamiliaAgent" /f >nul 2>&1
reg delete "HKCU\Software\PaxFamilia" /f >nul 2>&1
reg delete "HKLM\Software\Microsoft\Windows\CurrentVersion\Uninstall\PaxFamilia" /f >nul 2>&1

set "TARGET_DIR=C:\Program Files\PaxFamilia"
echo [*] 설치 파일 정리 중...
del /F /Q "%TARGET_DIR%\pax-winagent.exe" >nul 2>&1
del /F /Q "%TARGET_DIR%\pax-config.json" >nul 2>&1
del /F /Q "%TARGET_DIR%\install.bat" >nul 2>&1

echo.
echo ==========================================================
echo   ✅ PaxFamilia가 컴퓨터에서 성공적으로 제거되었습니다.
echo ==========================================================
pause
