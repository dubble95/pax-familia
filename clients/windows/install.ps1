# PaxFamilia PowerShell Installer
# Run with: powershell -ExecutionPolicy Bypass -File install.ps1

Write-Host "==========================================================" -ForegroundColor Cyan
Write-Host "   🕊️ PaxFamilia 윈도우 에이전트 설치 마법사" -ForegroundColor Cyan
Write-Host "==========================================================" -ForegroundColor Cyan

$isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $isAdmin) {
    Write-Host "[!] 관리자 권한으로 재실행합니다..." -ForegroundColor Yellow
    Start-Process powershell -ArgumentList "-NoProfile -ExecutionPolicy Bypass -File `"$PSCommandPath`"" -Verb RunAs
    exit
}

$targetDir = "C:\Program Files\PaxFamilia"
if (-not (Test-Path $targetDir)) {
    New-Item -ItemType Directory -Path $targetDir -Force | Out-Null
}

Stop-Process -Name "pax-winagent" -Force -ErrorAction SilentlyContinue

$currentDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Copy-Item "$currentDir\pax-winagent.exe" "$targetDir\" -Force
if (Test-Path "$currentDir\uninstall.bat") {
    Copy-Item "$currentDir\uninstall.bat" "$targetDir\" -Force
}

Write-Host ""
$defaultServer = "https://pax-familia.fly.dev"
$server = Read-Host "서버 주소 입력 (기본값 엔터: $defaultServer)"
if ([string]::IsNullOrWhiteSpace($server)) { $server = $defaultServer }

$code = Read-Host "부모 대시보드의 6자리 페어링 코드 입력 (생략 시 엔터)"
if (-not [string]::IsNullOrWhiteSpace($code)) {
    Write-Host "[*] 서버에 기기 등록 요청 중..." -ForegroundColor Green
    & "$targetDir\pax-winagent.exe" -server $server -code $code
}

# 작업 스케줄러 등록
Write-Host "[*] Windows 작업 스케줄러 부팅 시 자동 시작 등록..." -ForegroundColor Green
schtasks /Create /F /TN "PaxFamiliaAgent" /TR "`"$targetDir\pax-winagent.exe`"" /SC ONLOGON /RL HIGHEST | Out-Null

# 시작
Start-Process "$targetDir\pax-winagent.exe"

Write-Host "==========================================================" -ForegroundColor Cyan
Write-Host "  🎉 PaxFamilia 설치가 성공적으로 완료되었습니다!" -ForegroundColor Green
Write-Host "  위치: $targetDir" -ForegroundColor White
Write-Host "==========================================================" -ForegroundColor Cyan
