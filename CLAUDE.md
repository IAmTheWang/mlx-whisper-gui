# whisper-gui

A local macOS-only web GUI for batch video transcription to `.srt`. Go backend serves a small vanilla TS/Vite frontend (no framework) and drives one of two transcription engines as a subprocess: `mlx_whisper` (Python/MLX) or whisper.cpp's `whisper-cli` (no Python dependency at all). `ffmpeg` is used by both, either internally (mlx_whisper) or invoked directly (whisper.cpp's video→WAV preprocessing).

## Architecture

- `cmd/whisper-gui` — binary entrypoint; starts the HTTP server on `:8787` (flag `-port`), handles graceful shutdown.
- `internal/jobmanager` — the core: job state machine, a single-worker FIFO queue, the `Engine` abstraction (`mlx_whisper` vs whisper.cpp) that runs the transcription subprocess, SSE event fan-out.
- `internal/server` — HTTP handlers (`net/http` `ServeMux`, Go 1.22+ method+pattern routing), embeds the built frontend via `go:embed`.
- `internal/config` — tiny JSON config file (`~/Library/Application Support/whisper-gui/config.json`) for user-overridden binary paths, engine preference, and an optional whisper.cpp VAD model path.
- `internal/whisperbin` — resolves the `mlx_whisper`/`ffmpeg`/`whisper-cli` binary paths (config override → `PATH` → known pip fallback for mlx_whisper only).
- `web/` — the frontend: vanilla TypeScript + `el()`/`clear()` DOM helpers, no framework, built by Vite straight into `internal/server/dist` so `go:embed` needs no copy step.
- `srt/` — local scratch folder for subtitle files pulled off this machine's whisper-gui job history; gitignored, not part of the app itself.
- `notes/` — personal reference notes (Go-learning notes, investigation write-ups); not part of the app, not imported by any code.

Each subdirectory listed above has its own `CLAUDE.md` with more detail. Generated/vendor directories (`bin/`, `web/node_modules/`, `web/dist/`, `internal/server/dist/`) intentionally do not have one.

## Runtime state (not in git)

All job history, subprocess output logs, and generated `.srt` files live under `~/Library/Application Support/whisper-gui/` (`config.json`, `jobs.json`, `jobs/<id>/`) — never under the repo. `srt/` in this repo is just a manually-curated copy pulled from there.

## Build & run

```bash
make dev-api    # go run ./cmd/whisper-gui (expects a separately-built/dev frontend)
make dev-web    # cd web && bun run dev (Vite dev server, proxies /api to :8787)
make dev-api-watch  # like dev-api, but auto-rebuilds/restarts on saved .go files (needs `air`, see README)
make dev-watch      # dev-api-watch + dev-web together
make build      # builds frontend into internal/server/dist, then go build ./cmd/whisper-gui
make run        # build + run the binary
make test       # go test ./...
```

External dependencies the app shells out to (not vendored): `mlx_whisper` (pip, Apple Silicon MLX build), whisper.cpp's `whisper-cli` (`brew install whisper-cpp`, no Python needed), and `ffmpeg` (Homebrew). All three are resolved at runtime by `internal/whisperbin`; the Settings panel in the UI lets the user override the paths if auto-detection fails, set the whisper.cpp model directory, pick a default engine, and optionally point at a downloaded VAD model to reduce whisper.cpp's silence-triggered repeated/hallucinated text (see `internal/jobmanager/CLAUDE.md`).

## Conventions

- Go: standard library only for the server (no web framework, no router library).
- Concurrency: transcription jobs run **one at a time**, regardless of engine — MLX's unified-memory GPU path gains nothing from parallel jobs on a single Mac and it would only add OOM risk. Don't add a worker pool without revisiting that assumption.
- Frontend: no framework, no JSX/build-time templating — DOM built directly via the `el()` helper in `web/src/dom.ts`.
- Package manager: [Bun](https://bun.sh), not npm — `web/` has no `package-lock.json`; use `bun install`/`bun run <script>` (or the `make` targets, which already do this).
- Git commits: see the user-level instructions — commit messages are in Japanese.
