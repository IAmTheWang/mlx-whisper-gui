.PHONY: dev dev-web dev-api build run clean test

dev:
	@trap 'kill 0' EXIT; \
	$(MAKE) dev-api & \
	$(MAKE) dev-web & \
	wait

dev-web:
	cd web && npm install && npm run dev

dev-api:
	go run ./cmd/whisper-gui

build:
	cd web && npm install && npm run build
	go build -o bin/whisper-gui ./cmd/whisper-gui

run: build
	./bin/whisper-gui

test:
	go test ./...

clean:
	rm -rf bin web/node_modules web/dist internal/server/dist
