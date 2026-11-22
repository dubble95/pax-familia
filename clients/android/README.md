# 📱 PaxFamilia 안드로이드 클라이언트 (Android Agent)

안드로이드 스마트폰과 안드로이드 태블릿을 동시에 지원하는 공식 클라이언트입니다.

---

## 1. 아키텍처 및 구현 완성 내역

본 안드로이드 클라이언트는 Kotlin 기반으로 작성되었으며, 실전 기기 배포를 위한 핵심 로직과 우회 방어 체계가 완전히 구축되어 있습니다:

```
clients/android/
├── build.gradle.kts
├── settings.gradle.kts
└── app/
    ├── build.gradle.kts
    └── src/main/
        ├── AndroidManifest.xml          # 필수 권한 (Android 14 specialUse 대응)
        ├── java/com/paxfamilia/agent/
        │   ├── PaxApp.kt                # Application 클래스 & 알림 채널 생성
        │   ├── MainActivity.kt          # 서버 URL 설정 및 필수 권한 허용 마법사
        │   ├── PaxService.kt            # 상주 포그라운드 서비스 & 오프라인 우회 방어 타이머
        │   ├── OverlayLockActivity.kt   # 전면 잠금 창 & 부모 비상 4자리 PIN 해제 다이얼로그
        │   ├── UsageTracker.kt          # UsageStatsManager 기반 현재 실행 앱 감지
        │   ├── BootReceiver.kt          # 폰/태블릿 부팅 시 PaxService 자동 가동
        │   ├── PaxDeviceAdminReceiver.kt # 무단 삭제 방지 기기 관리자(Device Admin)
        │   └── model/
        │       └── Protocol.kt          # JSON 직렬화/역직렬화 데이터 모델
        └── res/
            ├── layout/
            │   ├── activity_main.xml    # 부모 설정 및 권한 상태 가이드 UI
            │   └── activity_lock.xml    # 전면 잠금 화면 레이아웃 (보너스 요청 + PIN 해제)
            ├── values/
            │   ├── strings.xml, colors.xml, themes.xml
            └── xml/
                └── device_admin_policy.xml # 기기 관리자 정책 설정
```

---

## 2. 🛡️ 자녀의 우회 수단(Bypass) 완벽 방어 체계

1. **오프라인/비행기 모드 탈출 방어 (Offline Fail-Safe Countdown)**:
   - 자녀가 서버 동기화를 막기 위해 Wi-Fi나 모바일 데이터를 끄더라도, 앱이 마지막으로 수신된 잔여 시간을 기준으로 **로컬에서 1초마다 독립 카운트다운**을 진행합니다.
   - 시간이 0이 되면 인터넷이 끊겨 있어도 화면이 자동으로 즉시 잠깁니다.
2. **무단 삭제 및 강제 종료 방지 (`PaxDeviceAdminReceiver`)**:
   - 안드로이드 **기기 관리자(Device Admin)** 권한으로 등록되어, 부모의 동의 없이 설정 화면에서 앱을 삭제하거나 프로세스를 강제 중지할 수 없습니다.
3. **취침 모드 및 즉시 올스톱 강제 잠금 (`OverlayLockActivity`)**:
   - 뒤로가기 버튼(`onBackPressedDispatcher`), 홈/최근앱 전환을 차단하여 다른 앱 사용을 원천 봉쇄합니다.
4. **부모 긴급 비상 마스터키 (PIN 4자리 해제)**:
   - 인터넷이 끊겼거나 긴급 통화가 필요한 상황에서, 잠금 화면에서 `[🔑 부모 비상 해제]` 버튼을 누르고 4자리 PIN(기본: `1234`)을 입력하면 30분간 즉시 비상 해제됩니다.
5. **학습용 앱(EBS, 메가스터디 등) 시간 차감 면제**:
   - 교육용 앱 실행 시 서버로부터 `is_whitelisted: true`를 수신하여 자녀의 일일 스크린타임 차감을 자동으로 일시 중지합니다.

---

## 3. 📦 APK 빌드 및 설치 방법

### 방법 A: GitHub Actions 자동 빌드 (가장 추천 - 로컬 설치 불필요 🚀)
1. 이 저장소를 본인 GitHub에 Push합니다.
2. GitHub 저장소의 **Actions** 탭으로 이동합니다.
3. **Build Android APK** 워크플로우가 자동으로 실행되어 빌드를 완료합니다.
4. 완료된 빌드의 **Artifacts**에서 `pax-familia-android-debug.apk`를 다운로드하여 자녀 폰에 바로 설치합니다!

### 방법 B: Android Studio에서 열기
1. Android Studio를 실행하고 **Open**을 누른 뒤 `clients/android` 폴더를 선택합니다.
2. 기기를 USB 디버깅 또는 Wi-Fi로 연결하고 **Run 'app'** (Shift + F10)을 누릅니다.

---

## 4. 📱 실기기(갤럭시 등) 설치 후 필수 권한 승인 가이드

앱을 폰이나 태블릿에 처음 설치한 후, `MainActivity`에서 다음 권한을 1회 허용해야 합니다:

1. **사용 기록 접근 허용**: 현재 유튜브/게임 실행 여부 감지
2. **다른 앱 위에 표시**: 시간 초과 시 잠금 창 표시
3. **배터리 사용량 최적화 중지 (제한 없음)**: 화면이 켜져 있을 때 백그라운드 프로세스가 OS에 의해 강제 종료되는 현상 방지
4. **기기 관리자 활성화**: 자녀의 임의 앱 삭제 방지
