# internal/config

Single file (`config.go`): reads/writes a small JSON config at `~/Library/Application Support/whisper-gui/config.json`.

- `Config` currently holds exactly two optional fields: `MlxWhisperPath`, `FFmpegPath` — user overrides for when auto-detection in `whisperbin` fails.
- `Dir()` is the single source of truth for the app's data directory (`~/Library/Application Support/whisper-gui`) — `jobmanager.Manager` also builds its `baseDir` from this, so job history and config live side by side.
- `Load()` treats a missing file as an empty `Config{}`, not an error — there's no "first run" ceremony.
- No caching here; `whisperbin` is the layer that caches resolved paths and calls `Refresh()` after a `Save()`.
