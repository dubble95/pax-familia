# 📱 PaxFamilia 안드로이드 클라이언트 (Android Agent)

안드로이드 스마트폰과 안드로이드 태블릿을 동시에 지원하는 공식 클라이언트입니다.

---

## 1. 아키텍처 및 구현 완성 내역

본 안드로이드 클라이언트는 Kotlin 기반으로 작성되었으며, 다음과 같은 핵심 기능이 완전히 구현되어 있습니다:

```
clients/android/
├── build.gradle.kts
├── settings.gradle.kts
└── app/
    ├── build.gradle.kts
    └── src/main/
        ├── AndroidManifest.xml          # 필수 권한 및 컴포넌트 선언
        ├── java/com/paxfamilia/agent/
        │   ├── PaxApp.kt                # Application 클래스 & 알림 채널 생성
        │   ├── MainActivity.kt          # 서버 URL 설정 및 필수 권한 허용 마법사
        │   ├── PaxService.kt            # 상주 포그라운드 서비스 & 15초 Heartbeat 루프
        │   ├── OverlayLockActivity.kt   # 시간 초과/즉시 잠금 시 전면 오버레이 차단 액티비티
        │   ├── UsageTracker.kt          # UsageStatsManager 기반 현재 실행 앱 감지
        │   ├── BootReceiver.kt          # 폰/태블릿 부팅 시 PaxService 자동 가동
        │   ├── PaxDeviceAdminReceiver.kt # 무단 삭제 방지 기기 관리자(Device Admin)
        │   └── model/
        │       └── Protocol.kt          # JSON 직렬화/역직렬화 데이터 모델
        └── res/
            ├── layout/
            │   ├── activity_main.xml    # 부모 설정 및 권한 상태 가이드 UI
            │   └── activity_lock.xml    # 전면 잠금 화면 레이아웃
            ├── values/
            │   ├── strings.xml, colors.xml, themes.xml
            └── xml/
                └── device_admin_policy.xml # 기기 관리자 정책 설정
```

---

## 2. 핵심 동작 방식

1. **포그라운드 활성 시간 수집 (`UsageTracker`)**:
   - `UsageStatsManager`를 통해 현재 실행 중인 앱(유튜브, 로블록스, 브롤스타즈 등)을 감지.
   - `BroadcastReceiver`로 화면 켜짐/꺼짐(`ACTION_SCREEN_ON`, `ACTION_SCREEN_OFF`)을 감지하여, **화면이 꺼져 있을 때는 배터리를 소모하지 않고 사용 시간 측정을 일시 정지(Doze 친화적)**합니다.
2. **중앙 서버 / Cloudflare Workers 동기화 (`PaxService`)**:
   - 15초 주기로 `POST /api/heartbeat` 전송 (서버리스 `https://pax-familia.workers.dev` 또는 홈서버 `http://192.168.0.x:8080`).
   - 서버 응답에 따라 `remaining_minutes`를 상단 상주 알림에 실시간 업데이트.
3. **전면 잠금 오버레이 (`OverlayLockActivity`)**:
   - `should_lock == true` (시간 소진 or 부모 원클릭 잠금) 수신 시 즉시 전면을 덮는 잠금 액티비티 기동.
   - 뒤로가기 버튼(`onBackPressedDispatcher`), 홈/최근앱 전환 방어.
   - 부모가 웹 대시보드나 텔레그램에서 보너스 부여 또는 잠금 해제 시, 다음 동기화에서 자동으로 잠금 화면 종료.
4. **우회 방지 (`PaxDeviceAdminReceiver`)**:
   - 기기 관리자(Device Admin)로 등록되어 자녀가 앱 관리 화면에서 임의로 삭제하거나 강제 종료할 수 없음.

---

## 3. 빌드 및 설치 방법

### Android Studio에서 열기:
1. Android Studio를 실행하고 **Open**을 누른 뒤 `clients/android` 폴더를 선택합니다.
2. Gradle 동기화(Sync)가 완료되면 기기를 USB 또는 Wi-Fi로 연결하고 **Run 'app'** (Shift + F10)을 누릅니다.

### 터미널에서 APK 빌드 (Gradle Wrapper):
```bash
cd clients/android
gradle assembleDebug
# 생성 위치: app/build/outputs/apk/debug/app-debug.apk
```

---

## 4. Google Family Link와의 병행 운영 요령

- Google Family Link 부모 앱에서 PaxFamilia 앱을 **"항상 허용(Always Allowed)"** 또는 사용시간 제한 없음으로 설정해 둡니다.
- Family Link의 취침 시간, 유해 웹사이트 차단, 앱 설치 승인은 그대로 유지하고, **"PC와 태블릿, 스마트폰의 총량 스크린타임 통합 제어"**는 PaxFamilia가 전담합니다.
