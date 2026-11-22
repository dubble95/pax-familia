# 📱 PaxFamilia 안드로이드 클라이언트 (Android Agent)

안드로이드 스마트폰과 안드로이드 태블릿을 동시에 지원하는 공식 클라이언트입니다.

---

## 1. 주요 역할 및 책임

1. **포그라운드 활성 시간 수집 (`UsageStatsManager`)**:
   - 자녀가 현재 화면을 켜고 어떤 앱을 실행 중인지 10~30초 주기로 확인.
   - 화면이 꺼지면(Screen Off) 사용 시간 누적을 즉시 중지하여 배터리를 절약.
2. **중앙 서버와 Heartbeat 동기화**:
   - 30초마다 PaxFamilia 서버(`http://<서버IP>:8080/api/heartbeat`)로 활성 시간을 전송.
   - 서버로부터 `remaining_minutes`, `should_lock` 명령을 수신.
3. **전면 잠금 오버레이 (`SYSTEM_ALERT_WINDOW`)**:
   - `should_lock == true` (시간 소진 or 부모 원클릭 잠금) 시, 즉시 화면 전체를 덮는 잠금 액티비티를 포그라운드로 띄워 다른 앱 접근 차단.
4. **임의 삭제 방지 (`DeviceAdminReceiver`)**:
   - 자녀가 설정에서 임의로 앱을 삭제하거나 프로세스를 강제 종료하지 못하도록 기기 관리자(Device Admin) 권한 등록.

---

## 2. 안드로이드 핵심 구현 아키텍처

```
clients/android/
├── app/src/main/
│   ├── AndroidManifest.xml          # 필수 권한 (PACKAGE_USAGE_STATS, SYSTEM_ALERT_WINDOW 등)
│   ├── java/com/paxfamilia/agent/
│   │   ├── MainActivity.kt          # 최초 설정 UI (서버 URL 입력, 필수 권한 승인 마법사)
│   │   ├── PaxService.kt            # 백그라운드 상주 포그라운드 서비스 & Heartbeat 루프
│   │   ├── OverlayLockActivity.kt   # 시간 초과 시 표시되는 전면 차단 화면
│   │   ├── UsageTracker.kt          # UsageStatsManager 기반 현재 실행 앱 감지
│   │   ├── BootReceiver.kt          # 폰/태블릿 부팅 시 자동 시작 리시버
│   │   └── DeviceAdminPolicy.kt     # 무단 삭제 방지 기기 관리자
│   └── res/layout/
│       ├── activity_lock.xml        # 잠금 화면 레이아웃 ("오늘 사용시간이 모두 끝났습니다")
│       └── activity_main.xml        # 권한 허용 가이드 레이아웃
```

---

## 3. 필수 권한 목록

```xml
<manifest xmlns:android="http://schemas.android.com/apk/res/android" package="com.paxfamilia.agent">
    <uses-permission android:name="android.permission.INTERNET" />
    <uses-permission android:name="android.permission.FOREGROUND_SERVICE" />
    <uses-permission android:name="android.permission.SYSTEM_ALERT_WINDOW" />
    <uses-permission android:name="android.permission.RECEIVE_BOOT_COMPLETED" />
    <uses-permission android:name="android.permission.PACKAGE_USAGE_STATS" tools:ignore="ProtectedPermissions" />
</manifest>
```

---

## 4. Google Family Link와의 공존 요령

- Family Link의 "취침 시간", "성인 사이트 필터링", "앱 설치 승인" 기능은 그대로 켜둡니다.
- PaxFamilia 앱을 자녀 기기에 설치 후, Family Link 부모 앱에서 PaxFamilia 앱의 "항상 허용(Always Allowed)" 또는 일일 한도 없음을 부여합니다.
- 이렇게 하면 Family Link가 PaxFamilia를 종료하지 않으며, PaxFamilia가 PC와 태블릿, 폰 전체의 스크린타임 총량을 완벽하게 일원화하여 통제하게 됩니다.
