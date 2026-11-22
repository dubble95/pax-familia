# ☁️ PaxFamilia 서버리스(Serverless) 아키텍처 및 배포 가이드

PaxFamilia는 기존의 단일 Go 홈서버 배포 방식뿐만 아니라, **100% 서버리스(Serverless) 아키텍처**를 완벽하게 지원합니다.

---

## 1. 왜 PaxFamilia에 Serverless가 훨씬 이상적인가?

일반적인 가정에서 자녀의 스마트폰, 태블릿, PC를 통합 관리할 때 직면하는 가장 큰 문제는 **"홈 서버의 유지보수 부담"**입니다.

| 비교 항목 | 홈서버 / 라즈베리파이 방식 | **☁️ 서버리스 방식 (Cloudflare Workers)** |
| :--- | :--- | :--- |
| **운영 비용** | 전기세, 기기 구매 비용 발생 | **완전 무료 ($0 / 평생)** (CF Workers 일 10만 건 무료) |
| **외부 접속 (LTE/5G)** | 공유기 포트포워딩, DDNS 설정 필요 | **글로벌 HTTPS 도메인 즉시 자동 제공** (`*.workers.dev`) |
| **보안 (SSL/TLS)** | Let's Encrypt 인증서 갱신 신경 써야 함 | **Cloudflare 자동 TLS 인증서 기본 탑재** |
| **안정성 (가용성)** | 집 정전이나 공유기 재부팅 시 중단 | **전 세계 300+ 엣지 데이터센터 무중단 99.99%** |
| **콜드 스타트** | 없음 (단, 서버 다운 시 복구 수동) | **0ms (V8 Isolate 엔진으로 지연시간 전무)** |
| **텔레그램 봇 연동** | 24시간 봇 프로세스 폴링 상주 필요 | **서버리스 Webhook 수신으로 추가 비용/프로세스 0개** |

> **💡 트래픽 비용 계산:**
> - 자녀 1~3명이 하루 3~4시간 활성 상태로 15~30초마다 Heartbeat를 전송할 경우: **하루 약 3,000 ~ 15,000건 HTTP 요청**.
> - Cloudflare Workers의 일일 무료 제공량은 **100,000건**이므로, 일반 가정에서는 무료 티어의 10~15%만 소모하여 **영구 무료**로 운영 가능합니다.

---

## 2. 지원하는 서버리스 아키텍처

### 🥇 Option A: Cloudflare Workers (강력 추천)
- **위치**: `serverless/cloudflare/worker.js`
- **스토리지**: Cloudflare Workers KV (`PAX_KV`)
- **특징**:
  - 단일 자바스크립트 파일에 **API 백엔드 + 내장 웹 대시보드 + 텔레그램 봇 웹훅**이 모두 통합되어 있음.
  - 별도 빌드 도구 없이 Cloudflare 웹 콘솔에 복사-붙여넣기만 해도 1분 만에 가동.
  - 모든 PaxFamilia 클라이언트(Windows Agent, Android APK, Simulator)와 100% 호환.

### 🥈 Option B: Google Cloud Run (Go 컨테이너 서버리스)
- **위치**: `serverless/cloudrun/Dockerfile`
- **특징**:
  - 기존의 Go 코드를 100% 동일하게 유지하면서 컨테이너로 패키징하여 Cloud Run에 배포.
  - 트래픽이 없을 때 인스턴스가 0으로 축소(Scale-to-zero)되어 비용 0원 유지.
  - 월 200만 건 무료 요청 제공.

---

## 3. Cloudflare Workers 5분 무료 배포 가이드

### 🚀 가장 쉬운 방법: Cloudflare 웹 브라우저 콘솔에서 1분 배포 (추천)

1. [Cloudflare 대시보드](https://dash.cloudflare.com/)에 무료 가입 후 로그인합니다.
2. 좌측 메뉴에서 **Compute (Workers & Pages)** -> **Overview** 클릭 후 **Create Application** 클릭.
3. **Create Worker**를 선택하고 이름을 지정합니다 (예: `pax-familia`).
4. **Deploy**를 클릭한 후, 화면 우측 상단의 **Edit Code** 버튼을 누릅니다.
5. 코드 에디터의 기존 내용을 모두 지우고, 이 저장소의 [`serverless/cloudflare/worker.js`](../serverless/cloudflare/worker.js) 파일 내용을 그대로 복사하여 붙여넣습니다.
6. 우측 상단의 **Deploy** 버튼을 누르면 즉시 전 세계에 배포 완료!
   - 생성된 URL: `https://pax-familia.<당신의-서브도메인>.workers.dev`
   - 이 URL을 모바일 브라우저로 접속하면 곧바로 아름다운 **부모 대시보드**가 열립니다.

#### (선택) 영구 데이터 보존을 위한 KV 스토리지 연결:
1. Cloudflare 좌측 메뉴에서 **Workers & Pages** -> **KV** 클릭.
2. **Create a namespace** 클릭 -> 이름에 `PAX_KV` 입력 후 생성.
3. 생성한 Worker의 **Settings** -> **Variables and Secrets** -> **KV Namespace Bindings**로 이동.
4. **Add binding** 클릭:
   - Variable name: `PAX_KV`
   - KV namespace: 방금 생성한 `PAX_KV` 선택.
5. 저장하면 데이터가 재부팅이나 배포 후에도 영구 유지됩니다.

---

### 💻 터미널 CLI로 배포하는 방법 (Wrangler)

```bash
cd serverless/cloudflare

# 1. Cloudflare 로그인
npx wrangler login

# 2. KV 네임스페이스 생성
npx wrangler kv:namespace create PAX_KV
# 출력된 id를 wrangler.toml에 입력

# 3. 배포 실행
npx wrangler deploy
```

---

## 4. 기존 클라이언트 연동 (URL만 변경)

서버리스 배포 후 발급받은 URL을 기존 클라이언트에 전달하기만 하면 됩니다.

### 1) 3대 기기 시뮬레이터 테스트
```bash
./bin/pax-sim -server https://pax-familia.<서브도메인>.workers.dev
```

### 2) 윈도우 PC 에이전트 실행
```bash
# 윈도우 PC의 시작프로그램 또는 서비스에 등록
pax-winagent.exe -server https://pax-familia.<서브도메인>.workers.dev
```

### 3) 안드로이드 클라이언트 (스마트폰 / 태블릿)
- 앱 최초 실행 화면의 "중앙 서버 주소" 입력 칸에 `https://pax-familia.<서브도메인>.workers.dev`를 입력하고 연결 버튼을 터치합니다.

---

## 5. 텔레그램 부모 봇 연동 (원클릭 제어)

Cloudflare Workers에는 텔레그램 봇 웹훅 수신기가 내장되어 있어 별도의 봇 서버 없이 즉각적인 부모 알림 및 버튼 제어가 가능합니다.

1. 텔레그램에서 `@BotFather`를 검색하고 `/newbot` 명령으로 봇을 생성하여 **BOT_TOKEN**을 발급받습니다.
2. Cloudflare Worker 환경변수(Secrets)에 `TELEGRAM_BOT_TOKEN`을 등록합니다:
   ```bash
   npx wrangler secret put TELEGRAM_BOT_TOKEN
   ```
   (또는 Cloudflare 웹 콘솔 Settings -> Variables에서 등록)
3. 텔레그램 웹훅을 Cloudflare Worker 주소로 등록합니다:
   ```bash
   curl -F "url=https://pax-familia.<서브도메인>.workers.dev/api/telegram" https://api.telegram.org/bot<YOUR_BOT_TOKEN>/setWebhook
   ```
4. 이제 부모의 텔레그램에서 생성한 봇으로 아무 메시지나 보내면:
   - 남은 시간 및 기기별 현황이 즉시 표시됩니다.
   - `[🛑 즉시 잠금]`, `[🔓 잠금 해제]`, `[🎁 +15분]`, `[🎁 +30분]` 인라인 버튼을 눌러 어디서든 원터치로 자녀 기기를 제어할 수 있습니다!
