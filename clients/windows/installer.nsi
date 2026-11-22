; PaxFamilia Windows Installer Script (NSIS 3.x)
; --------------------------------------------------
!include "MUI2.nsh"
!include "FileFunc.nsh"

Name "PaxFamilia (가정의 평화)"
OutFile "PaxFamiliaSetup.exe"
InstallDir "$PROGRAMFILES64\PaxFamilia"
InstallDirRegKey HKCU "Software\PaxFamilia" "InstallDir"
RequestExecutionLevel admin

; 현대적 UI 테마 설정
!define MUI_ABORTWARNING

; 페이지 설정
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "Korean"

Section "PaxFamilia Core" SecCore
  SetOutPath "$INSTDIR"

  ; 기존 프로세스 종료
  DetailPrint "실행 중인 기존 에이전트 종료 중..."
  nsExec::Exec 'taskkill /F /IM pax-winagent.exe'

  ; 실행 바이너리 및 설정 복사
  File "..\..\bin\pax-winagent.exe"
  File "install.bat"
  File "uninstall.bat"

  ; 언인스톨러 생성
  WriteUninstaller "$INSTDIR\uninstall.exe"

  ; 윈도우 부팅 시 자동 실행 등록 (작업 스케줄러 & 레지스트리)
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Run" "PaxFamiliaAgent" '"$INSTDIR\pax-winagent.exe"'
  WriteRegStr HKCU "Software\PaxFamilia" "InstallDir" "$INSTDIR"

  ; 제어판 프로그램 추가/제거 등록
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\PaxFamilia" "DisplayName" "PaxFamilia - 가정의 평화 스크린타임 에이전트"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\PaxFamilia" "UninstallString" '"$INSTDIR\uninstall.exe"'
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\PaxFamilia" "DisplayIcon" '"$INSTDIR\pax-winagent.exe"'
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\PaxFamilia" "Publisher" "PaxFamilia Team"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\PaxFamilia" "DisplayVersion" "1.0.0"

  ; 시작 메뉴 바로가기
  CreateDirectory "$SMPROGRAMS\PaxFamilia"
  CreateShortcut "$SMPROGRAMS\PaxFamilia\PaxFamilia 실행.lnk" "$INSTDIR\pax-winagent.exe"
  CreateShortcut "$SMPROGRAMS\PaxFamilia\제거.lnk" "$INSTDIR\uninstall.exe"

  ; 작업 스케줄러 최고 권한 자동 시작 작업 등록
  nsExec::Exec 'schtasks /Create /F /TN "PaxFamiliaAgent" /TR "\"$INSTDIR\pax-winagent.exe\"" /SC ONLOGON /RL HIGHEST'

  ; 설치 완료 후 즉시 백그라운드 실행
  Exec '"$INSTDIR\pax-winagent.exe"'
SectionEnd

Section "Uninstall"
  ; 실행 중인 프로세스 종료
  nsExec::Exec 'taskkill /F /IM pax-winagent.exe'
  nsExec::Exec 'schtasks /Delete /TN "PaxFamiliaAgent" /F'

  DeleteRegValue HKCU "Software\Microsoft\Windows\CurrentVersion\Run" "PaxFamiliaAgent"
  DeleteRegKey HKCU "Software\PaxFamilia"
  DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\PaxFamilia"

  Delete "$SMPROGRAMS\PaxFamilia\*.lnk"
  RMDir "$SMPROGRAMS\PaxFamilia"

  Delete "$INSTDIR\pax-winagent.exe"
  Delete "$INSTDIR\pax-config.json"
  Delete "$INSTDIR\install.bat"
  Delete "$INSTDIR\uninstall.bat"
  Delete "$INSTDIR\uninstall.exe"
  RMDir "$INSTDIR"
SectionEnd
