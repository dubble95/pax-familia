# 🚀 PaxFamilia Fly.io 3분 배포 가이드

Fly.io는 PaxFamilia의 Go 백엔드를 **24시간 상시 가동(무중단)**하면서, **1GB 영구 디스크 볼륨(`pax_data`)을 평생 무료**로 제공하는 가장 추천하는 호스팅 플랫폼입니다.

---

## 📌 주요 혜택 요약

- **비용**: **$0 (완전 무료)**
- **리전**: Tokyo (`nrt`) — 한국 기준 지연시간 ~25ms로 초저지연
- **영구 보존**: 서버가 재부팅되거나 새 버전으로 업데이트되어도 자녀 스크린타임 사용량 데이터(`pax_state.json`)가 1GB 영구 디스크에 안전하게 유지됨
- **HTTPS 자동 지원**: `https://<앱이름>.fly.dev` 자동 발급

---

## 🛠️ 배포 3단계 (터미널)

### 1단계: Fly.io 로그인
터미널에서 아래 명령을 실행하면 브라우저 로그인 창이 열립니다:
```bash
fly auth login
```
*(계정이 없다면 [fly.io](https://fly.io)에서 무료 회원가입 후 로그인)*

---

### 2단계: 무료 1GB 영구 디스크 볼륨 생성
데이터가 유실되지 않도록 도쿄 리전(`nrt`)에 무료 1GB 볼륨을 생성합니다:
```bash
fly volumes create pax_data --region nrt --size 1
```

---

### 3단계: 배포 실행
프로젝트 루트에서 바로 배포 명령을 실행합니다:
```bash
fly deploy
```

배포가 완료되면 다음과 같은 메시지가 나타납니다:
```
Visit your newly deployed app at https://pax-familia-xxxx.fly.dev
```

---

## 📱 배포 후 클라이언트 연동

발급받은 Fly.io 주소를 클라이언트에 입력하기만 하면 즉시 연동됩니다!

### 1) 3대 기기 시뮬레이터 테스트
```bash
./bin/pax-sim -server https://<앱이름>.fly.dev
```

### 2) 윈도우 PC 에이전트 실행
```bash
pax-winagent.exe -server https://<앱이름>.fly.dev
```

### 3) 안드로이드 스마트폰 & 태블릿
- 앱 설치 후 최초 화면의 "서버 URL"에 `https://<앱이름>.fly.dev` 입력 후 "서비스 시작" 클릭.

---

## ⚙️ 유용한 Fly CLI 명령어

```bash
# 실시간 서버 로그 확인
fly logs

# 서버 가동 상태 모니터링
fly status

# 웹 대시보드 브라우저로 바로 열기
fly open
```
