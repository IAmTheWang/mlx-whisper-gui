# internal/whisperbin

Single file (`locate.go`): resolves the `mlx_whisper` and `ffmpeg` executable paths so the rest of the app never has to guess.

Resolution order:
- `mlx_whisper`: `config.json` override → `PATH` → known pip user-install fallback (`~/Library/Python/3.9/bin/mlx_whisper`).
- `ffmpeg`: `config.json` override → `PATH` only (normally on a standard Homebrew `PATH`, so no fallback path is needed).

Key behaviors:
- `LocateMlxWhisper()`/`LocateFFmpeg()` **never error** — they return a `Resolution` with `ResolvedVia: ViaNone` when nothing is found. Callers must check `ResolvedVia`, not treat an empty path as impossible.
- Both are cached process-wide (`sync.Mutex` + package-level vars) after first resolution. Call `Refresh()` after the user saves a new path in Settings (`server/settings.go` already does this) — otherwise the old cached path keeps being used.
- `IsExecutable()` is exported specifically so the Settings PUT handler can validate a user-supplied path with the exact same check used internally, rather than duplicating the stat+mode logic.
