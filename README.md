# 🕊️ PaxFamilia (팍스 파밀리아)
> **"가정의 평화를 수호하는 자녀 크로스 플랫폼 통합 스크린타임 관리 시스템"**

*라틴어로 '가정의 평화'를 뜻하는 PaxFamilia는 스마트폰, 태블릿, PC로 파편화된 자녀의 디지털 사용시간 갈등을 해결하고 온 가족의 화목을 지키기 위한 오픈소스 솔루션입니다.*

---

## 📌 1. 배경 및 문제 정의 (Why PaxFamilia?)

현재 부모들은 자녀의 기기를 관리하기 위해 **Google Family Link**(안드로이드 폰, 태블릿)와 **Microsoft Family Safety**(윈도우 PC)를 각각 사용하고 있습니다. 

하지만 두 시스템이 완벽히 분리되어 있어 다음과 같은 치명적인 문제가 발생합니다:

1. **기기 간 "메뚜기(풍선) 효과"**:
   - 폰에서 일일 한도(예: 1시간)가 다 소진되면, 자녀는 즉시 안드로이드 태블릿을 켜고, 태블릿이 끝나면 다시 윈도우 PC로 옮겨가 게임을 합니다.
   - 결과적으로 부모가 의도한 하루 총 사용량(예: 1시간 30분)을 훌쩍 넘어 하루 4~5시간 이상 스크린에 노출됩니다.
2. **부모의 피로도와 잔소리 유발**:
   - "폰 몇 분 봤어? 이제 PC 끄고 태블릿도 그만해!"라는 끝없는 감시와 잔소리로 인해 가정의 평화가 깨집니다.
3. **공식 API의 부재**:
   - 구글과 MS 모두 서드파티 통합을 위한 공식 퍼블릭 API를 제공하지 않아, 기존 도구만으로는 단일한 "총량 예산(Shared Pool Quota)" 제어가 불가능합니다.

---

## 🎯 2. 핵심 목표 및 해결책

PaxFamilia는 기존 도구를 강제로 대체하는 것이 아니라, **기존의 강점(콘텐츠 필터링, 앱 승인 등)은 유지하면서 "통합 스크린타임 총량 관리"를 완벽히 메워주는 하이브리드 보완 시스템**입니다.

* **통합 일일 쿼터제 (Unified Daily Screen Budget)**:
  - 기기가 몇 대이든 자녀 1명당 하루 "공유 사용시간(예: 평일 2시간, 주말 3시간)"을 부여합니다.
  - 폰에서 40분을 쓰면, 태블릿과 PC의 잔여 시간도 자동으로 1시간 20분으로 차감됩니다.
* **크로스 디바이스 즉각 락다운 (Instant Cross-Device Lockdown)**:
  - 총량이 소진되면 사용 중인 기기에 5분 전 경고 후 화면이 잠깁니다.
  - "식사 시간", "취침 시간", "숙제 시간" 등 부모가 원클릭으로 모든 기기(폰+태블릿+PC)를 동시 잠금할 수 있습니다.
* **원클릭 보너스 & 텔레그램 연동**:
  - 착한 일을 했거나 과제를 마쳤을 때, 복잡한 포털에 들어갈 필요 없이 텔레그램 버튼 한 번으로 "+30분 보너스"를 지급합니다.

---

## 🏗️ 3. 지원 대상 기기 (우선순위)

1. 📱 **안드로이드 폰 (Android Smartphone)**
2. 📱 **안드로이드 태블릿 (Android Tablet)**
3. 💻 **윈도우 PC (Windows 10 / 11)**

---

## 📂 프로젝트 구조

```
pax-familia/
├── docs/
│   ├── ARCHITECTURE.md       # 시스템 아키텍처 및 통신 프로토콜
│   ├── ROADMAP.md            # 단계별 개발 계획 및 마일스톤
│   ├── SERVERLESS.md         # 서버리스(Cloudflare Workers / Cloud Run) 배포 가이드
│   └── QUICKSTART.md         # 1분 빠른 실행 가이드
├── server/                   # 중앙 제어 서버 (Go 로컬/홈서버 바이너리) & 대시보드
├── serverless/               # ☁️ 100% 무료 서버리스 백엔드 (Cloudflare Workers, Cloud Run)
├── clients/
│   ├── android/              # 안드로이드(폰/태블릿 공용) Kotlin 에이전트 & 잠금 오버레이
│   └── windows/              # 윈도우 백그라운드 트레이 에이전트 (Go)
├── cmd/simulator/            # 3대 기기(폰/태블릿/PC) 통합 가상 시뮬레이터
└── shared/                   # 공통 데이터 모델 및 통신 프로토콜 정의
```

---

## ⚡ 빠른 시작 (선택형 아키텍처)

### Option 1: 🚀 Fly.io 무료 클라우드 배포 (강력 추천 🏆)
24시간 홈서버 없이 **평생 $0**으로 Go 백엔드를 상시 가동합니다. 1GB 영구 디스크 무료 제공으로 데이터가 영구 보존됩니다.
👉 **[Fly.io 3분 배포 가이드 (docs/FLY_DEPLOY.md)](docs/FLY_DEPLOY.md)** 참조

### Option 2: ☁️ 100% 무료 서버리스 배포 (Cloudflare Workers)
Cloudflare Workers로 1분 만에 전 세계 엣지에 배포하고, 집 밖 LTE/5G 환경에서도 HTTPS로 안정적으로 관리할 수 있습니다.
👉 **[서버리스 배포 가이드 (docs/SERVERLESS.md)](docs/SERVERLESS.md)** 참조

### Option 3: 🌐 기타 무료 PaaS (Render.com / Koyeb)
GitHub 저장소 연결로 1클릭 자동 배포할 수 있습니다.
👉 **[무료 호스팅 완벽 가이드 (docs/FREE_HOSTING.md)](docs/FREE_HOSTING.md)** 참조

### Option 4: 🏠 홈서버 / 로컬 Go 실행
```bash
make build
make run   # 부모 대시보드: http://localhost:8080
make sim   # 3대 기기 전환 시뮬레이터 실행
```

---

## 📖 문서 바로가기

- [시스템 아키텍처 상세 보기 (ARCHITECTURE.md)](docs/ARCHITECTURE.md)
- [단계별 개발 로드맵 (ROADMAP.md)](docs/ROADMAP.md)
- [Fly.io 3분 배포 가이드 (FLY_DEPLOY.md)](docs/FLY_DEPLOY.md)
- [무료 앱 호스팅 가이드 (FREE_HOSTING.md)](docs/FREE_HOSTING.md)
- [서버리스 배포 가이드 (SERVERLESS.md)](docs/SERVERLESS.md)
- [1분 빠른 시작 가이드 (QUICKSTART.md)](docs/QUICKSTART.md)
