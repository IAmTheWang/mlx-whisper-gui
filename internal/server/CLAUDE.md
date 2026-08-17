# internal/server

HTTP layer only — all business logic lives in `internal/jobmanager`. Standard library `net/http` with Go 1.22+ method+pattern `ServeMux` routing; no router library, no middleware framework beyond the one panic-recovery wrapper.

## Files

- `server.go` — `Server` struct, `Handler()` (wraps the mux in `recoverMiddleware` so a panic in one request 500s only that request), `writeJSON`/`writeJSONError` helpers, `NewHTTPServer`.
- `routes.go` — the full route table. Note the `/api/` catch-all registered *after* the specific `/api/...` routes: any unmatched API path gets a JSON 404 instead of falling through to the SPA handler — a typo'd endpoint must never silently return 200 + HTML.
- `embed.go` — `go:embed dist` embeds the Vite-built frontend directly into the binary; `spaHandler()` serves it with an SPA-style fallback to `index.html` (this app is single-screen with no client routing, so the fallback is mostly a robustness net).
- `browse.go` — `GET /api/browse`: server-side filesystem browser used by the frontend's file picker. Validates the path is absolute; on macOS TCC permission errors, returns a 403 with an explicit "grant Full Disk Access" message rather than a bare 403.
- `jobs.go` — job CRUD/control endpoints (`POST/GET /api/jobs`, `GET/POST /api/jobs/{id}...`), all thin wrappers over `jobmanager.Manager`.
- `models.go` — curated model/language lists (`curatedModels`, `curatedLanguages`) and Hugging Face Hub cache detection (`isModelCached`, reads `~/.cache/huggingface/hub/...` directly rather than shelling out).
- `settings.go` — `GET/PUT /api/settings` for the mlx_whisper/ffmpeg path overrides; validates with `whisperbin.IsExecutable` before saving and calls `whisperbin.Refresh()` after.
- `srt.go` — `GET /api/jobs/{id}/srt`: streams the finished `.srt` back with a `Content-Disposition` filename derived from the *video's* basename (not the job ID) so downloads are human-readable. Path always comes from the server-side job record, never from client input.
- `sse.go` — `GET /api/jobs/{id}/events`: Server-Sent Events stream (`log`/`progress`/`state`). Disables the write deadline explicitly (`SetWriteDeadline(time.Time{})`) since a job can run for tens of minutes. Falls back to replaying the persisted `output.log` for a historical job with an empty in-memory buffer (e.g. after a server restart).
- `dist/` — Vite build output, embedded via `embed.go`. Generated, gitignored, no `CLAUDE.md`.

## Conventions

- Every handler validates client input before touching the filesystem/subprocess layer (`isKnownModel`, `isKnownLanguage`, `filepath.IsAbs`, `whisperbin.IsExecutable`).
- `NewHTTPServer` deliberately leaves `WriteTimeout` at its zero value (unlimited) — a global timeout would silently sever the long-lived SSE connections with no diagnosable error. Don't add one at this level; put per-handler timeouts on short-lived endpoints only if ever needed.
- New endpoints go in a new file (one concern per file, as above) and get registered in `routes.go`.
