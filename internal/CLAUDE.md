# internal/

All application logic, not importable outside this module. Four packages, layered:

- `config` — lowest layer. Reads/writes `~/Library/Application Support/whisper-gui/config.json` (user-overridden binary paths, whisper.cpp model directory, default engine). No dependencies on other `internal/` packages.
- `whisperbin` — depends on `config`. Resolves the actual `mlx_whisper`/`ffmpeg`/`whisper-cli` executable paths (config override → `PATH` → fallback) and caches the result until `Refresh()` is called.
- `jobmanager` — depends on `config` (indirectly, via `whisperbin`) and `whisperbin`. Owns job state, the transcription queue, the `Engine` abstraction (mlx_whisper vs whisper.cpp), and subprocess execution. No dependency on `server`.
- `server` — depends on `jobmanager` and `whisperbin`/`config` for the Settings/models endpoints (also imports `jobmanager`'s engine name constants directly, e.g. for the `?engine=` query param and job-creation validation). All HTTP concerns live here; `jobmanager` knows nothing about HTTP.

Keep that dependency direction one-way (`server` → `jobmanager` → `whisperbin` → `config`). `jobmanager` and `whisperbin` must stay HTTP-agnostic so they can be unit-tested without spinning up a server (see the fake-binary test pattern in `jobmanager/manager_test.go`).

See each package's own `CLAUDE.md` for details.
