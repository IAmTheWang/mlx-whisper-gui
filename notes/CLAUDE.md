# notes/

Personal reference notes for whoever's working on this repo — not application code; nothing here is imported or read by `cmd/`, `internal/`, or `web/`.

- `go-cli-basics.md` — Go/CLI fundamentals notes (Makefile targets, `go:embed`, dependency management), written while learning Go through this project.
- `mlx-whisper-repetition-loop-bugfix.md` — investigation write-up for the `condition_on_previous_text` repetition-loop bug fixed in `internal/jobmanager/manager.go` (2026-09-04): symptoms, root cause, before/after verification data, follow-ups.

Add new files here for anything worth remembering long-term that doesn't fit a per-directory `CLAUDE.md` (which documents current-state architecture, not investigation history).
