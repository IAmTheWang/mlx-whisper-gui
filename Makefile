.PHONY: dev dev-web dev-api dev-api-watch dev-watch build run clean test

dev:
	@trap 'kill 0' EXIT; \
	$(MAKE) dev-api & \
	$(MAKE) dev-web & \
	wait

dev-web:
	cd web && bun install && bun run dev

dev-api:
	go run ./cmd/whisper-gui

dev-api-watch:
	@command -v air >/dev/null 2>&1 || { echo "air not found — run: go install github.com/air-verse/air@latest"; exit 1; }
	air -c .air.toml

dev-watch:
	@trap 'kill 0; exit' INT TERM EXIT; \
	$(MAKE) dev-api-watch & \
	$(MAKE) dev-web & \
	wait

build:
	cd web && bun install && bun run build
	go build -o bin/whisper-gui ./cmd/whisper-gui

run: build
	./bin/whisper-gui

test:
	go test ./...

clean:
	rm -rf bin web/node_modules web/dist internal/server/dist tmp
