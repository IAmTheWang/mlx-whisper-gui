# cmd/whisper-gui

The binary entrypoint (`main.go`). Responsibilities, in order:

1. Parse `-port` flag (default `8787`).
2. Build a `jobmanager.Manager` (`jobmanager.NewManager()` — loads persisted job history from `~/Library/Application Support/whisper-gui/jobs.json`, starts the worker goroutine).
3. Resolve `mlx_whisper`/`ffmpeg`/`whisper-cli` via `whisperbin` and log how each was found (config/PATH/fallback/none) — this is diagnostic only; a job created with its engine's binary unresolved will fail with an actionable error message from `jobmanager.runJob`, not at startup. (The whisper.cpp VAD model path isn't logged here — it's a plain `config.WhisperVadModelPath` string, not something `whisperbin` resolves, so there's no "found via/not found" state to report.)
4. Start the HTTP server (`server.NewHTTPServer`) in a goroutine.
5. Block on `SIGINT`/`SIGTERM`, then shut down: **`jobs.Close()` runs before `httpServer.Shutdown`** — it force-kills any in-flight transcription subprocess (either engine) immediately so no orphaned process keeps holding GPU unified memory after the server exits. Don't reorder this.

No other files belong in this package — application logic lives in `internal/*`, this is wiring only.
