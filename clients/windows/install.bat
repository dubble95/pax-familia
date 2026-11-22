@echo off
chcp 65001 > nul
title PaxFamilia 윈도우 에이전트 설치 마법사

echo ==========================================================
echo   🕊️ PaxFamilia (가정의 평화) 윈도우 에이전트 설치
echo ==========================================================
echo.

:: 관리자 권한 확인 및 상승
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo [안내] 관리자 권한이 필요합니다. 관리자 권한으로 재실행합니다...
    powershell -Command "Start-Process '%~f0' -Verb RunAs"
    exit /b
)

set "TARGET_DIR=C:\Program Files\PaxFamilia"
echo [*] 설치 디렉토리 생성: %TARGET_DIR%
if not exist "%TARGET_DIR%" mkdir "%TARGET_DIR%"

:: 실행 중인 기존 프로세스 종료
taskkill /F /IM pax-winagent.exe >nul 2>&1

:: 파일 복사
echo [*] 파일 복사 중...
copy /Y "%~dp0pax-winagent.exe" "%TARGET_DIR%\" >nul
copy /Y "%~dp0uninstall.bat" "%TARGET_DIR%\" >nul 2>&1

:: 서버 및 페어링 코드 입력 안내
echo.
echo ----------------------------------------------------------
echo   [기기 연동 설정]
echo ----------------------------------------------------------
set "DEFAULT_SERVER=https://pax-familia.fly.dev"
set /p "SERVER_INPUT=서버 주소 입력 (기본값 엔터: %DEFAULT_SERVER%): "
if "%SERVER_INPUT%"=="" set "SERVER_INPUT=%DEFAULT_SERVER%"

echo.
set /p "PAIRING_CODE=부모 대시보드 6자리 페어링 코드 입력 (생략 시 엔터): "

if not "%PAIRING_CODE%"=="" (
    echo [*] 서버에 기기 연동 등록 요청 중...
    "%TARGET_DIR%\pax-winagent.exe" -server "%SERVER_INPUT%" -code "%PAIRING_CODE%"
)

:: 윈도우 부팅 시 최고 권한으로 자동 실행되도록 작업 스케줄러 등록
echo [*] 부팅 시 자동 시작 작업 스케줄러 등록 중...
schtasks /Create /F /TN "PaxFamiliaAgent" /TR "\"%TARGET_DIR%\pax-winagent.exe\"" /SC ONLOGON /RL HIGHEST >nul

:: 레지스트리 자동 실행 보조 등록
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Run" /v "PaxFamiliaAgent" /t REG_SZ /d "\"%TARGET_DIR%\pax-winagent.exe\"" /f >nul

:: 백그라운드 프로세스로 즉시 시작
echo [*] PaxFamilia 에이전트 백그라운드 시작 중...
start "" "%TARGET_DIR%\pax-winagent.exe"

echo.
echo ==========================================================
echo   🎉 PaxFamilia 설치 및 자동 실행 등록이 완료되었습니다!
echo   - 설치 경로: %TARGET_DIR%
echo   - 컴퓨터가 켜지면 자동으로 백그라운드에서 동작합니다.
echo ==========================================================
pause
