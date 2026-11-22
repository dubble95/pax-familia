# 🕊️ PaxFamilia Google Cloud Run Serverless Dockerfile
# 멀티스테이지 빌드로 15MB 미만의 초경량 서버리스 컨테이너 생성

FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod ./
COPY shared/ ./shared/
COPY server/ ./server/

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o pax-server ./server

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=builder /app/pax-server .
RUN mkdir -p data

ENV PORT=8080
EXPOSE 8080

CMD ["./pax-server", "-port", "8080", "-data", "/app/data/pax_state.json"]
