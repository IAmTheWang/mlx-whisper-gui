# cmd/whisper-gui

The binary entrypoint (`main.go`). Responsibilities, in order:

1. Parse `-port` flag (default `8787`).
2. Build a `jobmanager.Manager` (`jobmanager.NewManager()` — loads persisted job history from `~/Library/Application Support/whisper-gui/jobs.json`, starts the worker goroutine).
3. Resolve `mlx_whisper`/`ffmpeg` via `whisperbin` and log how each was found (config/PATH/fallback/none) — this is diagnostic only; a job created before either is fixed will fail with an actionable error message from `jobmanager.runJob`, not at startup.
4. Start the HTTP server (`server.NewHTTPServer`) in a goroutine.
5. Block on `SIGINT`/`SIGTERM`, then shut down: **`jobs.Close()` runs before `httpServer.Shutdown`** — it force-kills any in-flight `mlx_whisper` subprocess immediately so no orphaned process keeps holding GPU unified memory after the server exits. Don't reorder this.

No other files belong in this package — application logic lives in `internal/*`, this is wiring only.
