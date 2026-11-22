# 🗺️ PaxFamilia 단계별 개발 로드맵 (ROADMAP)

본 로드맵은 **최소 실행 가능 제품(MVP)**을 빠르게 완성하여 실제 가정에서 체감할 수 있는 효과를 내고, 점진적으로 완성도를 높여가는 3단계 계획입니다.

---

## 📅 마일스톤 개요

```
[Phase 1: 통합 쿼터 코어 & 윈도우 PC 연동] 
   └── 중앙 서버 구축 + Windows 트레이 에이전트 + 기본 대시보드
[Phase 2: 안드로이드 클라이언트 (폰/태블릿) 통합]
   └── 모바일 에이전트 개발 + 실시간 3기기 동기화 완성
[Phase 3: 부모 편의 기능 & 고도화]
   └── 텔레그램 봇 연동 + 앱별 차등 쿼터 + 주간 리포트
```

---

## 🚀 Phase 1: MVP 코어 및 윈도우 PC 연동 (완료 ✅)
> **목표**: 윈도우 PC의 사용시간을 중앙 서버에서 관리하고 원격 잠금/시간 관리가 실제로 작동하는 기반 구축

- [x] **중앙 서버 (Server Core)**:
  - 언어: **Go** (경량, 단일 바이너리, 홈서버/라즈베리파이/Mac 어디서든 손쉬운 배포)
  - DB: JSON State Store (단일 파일로 백업/관리가 용이)
  - 기능:
    - 자녀 계정 및 기기 등록 API
    - 일일 총량 쿼터 설정 및 잔여 시간 차감 엔진
    - 기기 상태 Heartbeat 수신 및 락/언락 명령 발행
- [x] **윈도우 PC 에이전트 (Windows Agent)**:
  - 시스템 트레이 앱 (남은 시간 표시, 경고 메시지 팝업)
  - 키보드/마우스 유휴 시간(Idle time) 감지
  - 시간 소진 시 `LockWorkStation` 잠금 실행
- [x] **간이 관리자 웹 (Parent Mini Web)**:
  - 현재 기기별 켜짐 상태 표시
  - 오늘 남은 시간 조회 및 수동 즉시 잠금/해제 버튼
- [x] **3대 기기 통합 가상 시뮬레이터 (`cmd/simulator`)**:
  - 폰/태블릿/PC 간 전환 및 실시간 쿼터 소진 검증 환경

---

## ☁️ Serverless Extension: 100% 무료 서버리스 백엔드 (완료 ✅)
> **목표**: 24시간 홈서버/라즈베리파이 없이도 Cloudflare Workers 엣지에서 평생 $0으로 운영

- [x] **Cloudflare Workers 백엔드 (`serverless/cloudflare/worker.js`)**:
  - 일 10만 건 무료 티어 완벽 대응 (3자녀 가동 시 충분)
  - KV 스토리지 기반 영속성 + 인메모리 폴백
  - 6개 REST API 규격 100% 호환
  - 웹 대시보드 내장 서빙 (`GET /`)
  - 텔레그램 봇 Webhook 직접 수신 (`POST /api/telegram`) 및 인라인 버튼 처리
- [x] **Google Cloud Run 배포 지원 (`serverless/cloudrun/Dockerfile`)**:
  - 기존 Go 바이너리의 15MB 초경량 서버리스 컨테이너화

---

## 📱 Phase 2: 안드로이드 폰 & 태블릿 에이전트 구축 (완료 ✅)
> **목표**: 안드로이드 폰과 태블릿을 연결하여 '폰 ↔ 태블릿 ↔ PC' 3자 간의 실시간 스크린타임 공유 완성

- [x] **안드로이드 에이전트 (Android Client APK 코드베이스)**:
  - 기술 스택: Kotlin + Coroutines + OkHttp + Gson
  - `UsageTracker` (`UsageStatsManager`)로 화면 켜짐 및 포그라운드 활성 앱(유튜브, 로블록스 등) 감지
  - `PaxService` (`Foreground Service`)로 15초 주기 백그라운드 상주 Heartbeat 루프
  - `ScreenReceiver`로 화면 꺼짐 시 자동 측정 일시정지 (배터리 및 Doze 친화적)
  - `OverlayLockActivity` (`SYSTEM_ALERT_WINDOW` 및 전면 잠금)로 시간 초과 시 뒤로가기/홈 방어 차단 화면 표시
  - `BootReceiver`로 폰 재부팅 시 자동 가동
- [x] **우회 방지 기초 조치**:
  - `PaxDeviceAdminReceiver` (기기 관리자) 등록을 통한 앱 무단 삭제 및 강제 종료 방지
- [x] **크로스 기기 실시간 동기화**:
  - 폰에서 사용 시 태블릿과 PC의 잔여 시간도 15초 내 동기화 차감

---

## 🕊️ Phase 3: 편의 기능 및 가정의 평화 고도화 (완료 ✅)
> **목표**: 부모의 번거로움을 0으로 만들고, 긍정적인 디지털 습관 형성 유도

- [x] **학습 vs 오락 차등 규칙 (Whitelist)**:
  - 인터넷 강의(EBS, 메가스터디, 클래스101, 구글 클래스룸 등) 및 교육용 앱은 스크린타임 차감 제외
  - 부모 대시보드에서 화이트리스트 앱 실시간 추가/삭제 관리 UI 지원
- [x] **스케줄 및 취침 모드 (Bedtime Lock)**:
  - 남은 시간이 있더라도 취침 시간대(기본 22:00 ~ 07:00)에는 모든 기기 강제 잠금
  - 부모 대시보드에서 취침 시간대 설정 및 On/Off 토글 지원
- [x] **기기별 사용시간 비율 및 통계 리포트**:
  - 오늘 사용한 총 시간 및 스마트폰 vs 태블릿 vs PC 점유율 게이지 차트 제공
- [x] **텔레그램 알림 & 원클릭 제어 Webhook**:
  - `[🛑 즉시 잠금]`, `[🔓 잠금 해제]`, `[🎁 +15분]`, `[🎁 +30분]` 인라인 버튼 지원 (`/api/telegram`)

---

## 🛠️ 권장 기술 스택 요약

| 구성 요소 | 추천 기술 | 선택 이유 |
| :--- | :--- | :--- |
| **Server** | **Go (Gin or Fiber)** | 메모리 점유율 20MB 미만, 크로스 컴파일 용이, 홈서버/라즈베리파이 최적 |
| **Database** | **SQLite (WAL 모드)** | 별도 DB 서버 불필요, 백업이 파일 복사 하나로 끝남 |
| **Windows Client** | **Go (wails or systray)** | 가볍고 백그라운드 리소스 소모가 거의 없음 |
| **Android Client** | **Kotlin** | OS 시스템 API(UsageStats, Overlay) 제어에 가장 안정적 |
| **Frontend** | **Tailwind + HTMX / React** | 반응형 모바일 브라우저에 최적화된 심플 대시보드 |
| **Notification** | **Telegram Bot API** | 별도 부모 전용 앱 설치 없이 카카오톡처럼 간편하게 수신 및 버튼 제어 |
