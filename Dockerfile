# 🕊️ PaxFamilia Universal Multi-stage Dockerfile
# Render, Fly.io, Koyeb, Google Cloud Run 등 모든 무료 앱 호스팅에 최적화

FROM golang:1.23-alpine AS builder

WORKDIR /app

# 의존성 복사 및 빌드
COPY go.mod ./
COPY shared/ ./shared/
COPY server/ ./server/

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o pax-server ./server

# 런타임 이미지 (Alpine 기반 초경량 ~15MB)
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
ENV TZ=Asia/Seoul

WORKDIR /app
COPY --from=builder /app/pax-server .
RUN mkdir -p /app/data

ENV PORT=8080
ENV DATA_PATH=/app/data/pax_state.json

EXPOSE 8080

CMD ["./pax-server"]
