.PHONY: all build run sim test clean

all: build

build:
	@echo "📦 바이너리 빌드 중..."
	@mkdir -p bin
	go build -o bin/pax-server ./server
	go build -o bin/pax-sim ./cmd/simulator
	go build -o bin/pax-winagent-mock ./clients/windows
	GOOS=windows GOARCH=amd64 go build -o bin/pax-winagent.exe ./clients/windows
	@echo "✅ 모든 바이너리 빌드 완료!"

run:
	@echo "🕊️ PaxFamilia 서버 시작 (http://localhost:8080)"
	./bin/pax-server -port 8080

sim:
	@echo "🎮 3대 기기 시뮬레이터 실행"
	./bin/pax-sim -server http://localhost:8080

clean:
	rm -rf bin/ data/*.json.tmp
