# 🌐 PaxFamilia 무료 클라우드 호스팅 & 서버리스 완벽 비교 가이드

PaxFamilia는 24시간 홈서버나 라즈베리파이를 켜두지 않아도, **100% 무료 티어를 제공하는 최신 클라우드 플랫폼**에서 평생 $0으로 운영할 수 있도록 설계되었습니다.

---

## 📊 무료 플랫폼 종합 비교표

| 플랫폼 | 유형 | 무료 제공량 | 영구 스토리지 | 추천 리전 | 특징 및 추천 대상 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **🥇 Cloudflare Workers** | 서버리스 (Edge) | **일 100,000건 무료** | Workers KV (무료) | 전 세계 300+ 엣지 | **초강력 추천**: 0ms 콜드스타트, 텔레그램 웹훅 기본 내장, 웹 콘솔 복사-붙여넣기 1분 배포 |
| **🥈 Fly.io** | 마이크로 VM (PaaS) | **VM 3대 + 3GB 볼륨 무료** | **1GB Persistent Disk 무료** | Tokyo (`nrt`) | **Go 네이티브 추천**: 파일 기반 DB(`pax_state.json`)가 영구 보존되는 24시간 상시 가동 VM |
| **🥉 Render.com** | 컨테이너 PaaS | **월 750시간 무료 (1개 상시)** | 임시 디스크 (재배포 시 초기화) | Singapore (`sin`) | **GitHub 연동 추천**: CLI 없이 GitHub 저장소 연결 후 클릭 한 번으로 자동 빌드/배포 |
| **Koyeb** | 서버리스 CaaS | **Nano 1대 무료 (512MB RAM)** | 임시 디스크 | Singapore (`sin`) | 고성능 글로벌 엣지, 빠른 기동 속도 |
| **Google Cloud Run** | 컨테이너 서버리스 | **월 200만 건 무료 요청** | Cloud Storage 연동 | Seoul (`asia-northeast3`) | 구글 클라우드 기반, 트래픽 없을 때 0으로 축소되어 비용 0원 |

---

## 1. 🥇 Cloudflare Workers 배포 (가장 빠르고 간편함)

별도의 빌드 과정 없이, 웹 브라우저에서 코드 복사-붙여넣기만으로 1분 만에 가동됩니다.

1. [Cloudflare 대시보드](https://dash.cloudflare.com/)에 무료 가입 및 로그인.
2. **Workers & Pages** -> **Create Application** -> **Create Worker** 클릭.
3. **Deploy** 후 **Edit Code** 클릭.
4. [`serverless/cloudflare/worker.js`](../serverless/cloudflare/worker.js) 파일의 전체 내용을 복사하여 붙여넣고 **Deploy** 클릭.
5. 발급된 주소: `https://pax-familia.<서브도메인>.workers.dev`
   - 모바일 브라우저로 접속 시 대시보드 즉시 로드.
   - 윈도우/안드로이드 클라이언트에 이 주소를 그대로 입력하면 연동 완료!

👉 자세한 내용: [서버리스 상세 가이드 (SERVERLESS.md)](SERVERLESS.md)

---

## 2. 🥈 Fly.io 배포 (영구 볼륨 지원 24시간 Go 서버)

Fly.io는 1GB의 영구 디스크 볼륨(`pax_data`)을 무료로 제공하므로, Go 서버가 재부팅되어도 자녀 사용시간 데이터(`pax_state.json`)가 영구적으로 안전하게 보존됩니다.

### 배포 방법:
```bash
# 1. Fly CLI 설치 (Mac)
brew install flyctl

# 2. 로그인
fly auth login

# 3. 앱 런칭 (fly.toml 기반)
fly launch --no-deploy

# 4. 무료 영구 볼륨 생성 (Tokyo 리전 1GB)
fly volumes create pax_data --region nrt --size 1

# 5. 배포
fly deploy
```

- 배포 완료 시 `https://pax-familia.fly.dev` 형태의 HTTPS 주소가 발급됩니다.

---

## 3. 🥉 Render.com 배포 (GitHub 클릭 1번 자동 배포)

CLI 명령 없이 GitHub 저장소를 연결하여 자동으로 배포하고 싶을 때 가장 편리합니다.

1. [Render.com](https://render.com)에 GitHub 계정으로 무료 가입.
2. 대시보드 우측 상단 **New +** -> **Web Service** 클릭.
3. 본인의 `pax-familia` 저장소 선택.
4. 설정값 확인:
   - **Environment**: `Docker`
   - **Region**: `Singapore` (한국과 가장 가까움)
   - **Instance Type**: `Free`
5. **Create Web Service** 클릭!
6. 몇 초 뒤 빌드가 완료되면 `https://pax-familia-xxxx.onrender.com` 주소로 즉시 서비스가 시작됩니다.

> **💡 팁:**
> Render 무료 티어는 15분 동안 요청이 없으면 절전 모드로 진입하지만, PaxFamilia 자녀 기기(폰, 태블릿, PC)가 켜지면 15초마다 Heartbeat를 보내므로 자녀가 기기를 사용하는 동안에는 자동으로 깨어나서 항상 활성 상태를 유지합니다.

---

## 4. 🔗 클라이언트 연동 (URL만 입력하면 끝!)

어떤 호스팅 서비스를 선택하든, 발급된 HTTPS 주소를 클라이언트에 넘겨주기만 하면 됩니다:

### 윈도우 PC 에이전트:
```bash
pax-winagent.exe -server https://<발급받은-호스팅-주소>
```

### 안드로이드 스마트폰 & 태블릿:
- 앱 첫 화면의 "서버 URL"에 `https://<발급받은-호스팅-주소>` 입력 후 저장.

### 가상 시뮬레이터:
```bash
./bin/pax-sim -server https://<발급받은-호스팅-주소>
```
