# internal/config

Single file (`config.go`): reads/writes a small JSON config at `~/Library/Application Support/whisper-gui/config.json`.

- `Config` holds optional user-override fields: `MlxWhisperPath`, `FFmpegPath`, `WhisperCliPath` (binary path overrides for when auto-detection in `whisperbin` fails), `WhisperCppModelDir` (directory scanned for `*.bin` model files, since whisper.cpp has no HF-repo-id-style auto-download), and `DefaultEngine` (`"mlx"` | `"whispercpp"` | `""` — empty means the frontend auto-picks whichever engine is actually resolvable at startup).
- `Dir()` is the single source of truth for the app's data directory (`~/Library/Application Support/whisper-gui`) — `jobmanager.Manager` also builds its `baseDir` from this, so job history and config live side by side.
- `Load()` treats a missing file as an empty `Config{}`, not an error — there's no "first run" ceremony.
- No caching here; `whisperbin` is the layer that caches resolved paths and calls `Refresh()` after a `Save()`.
