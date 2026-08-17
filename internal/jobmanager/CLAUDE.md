# internal/jobmanager

The core of the app: job state machine, a single-worker FIFO transcription queue, subprocess execution, and SSE event fan-out. Fully HTTP-agnostic — `internal/server` is the only consumer.

## Files

- `job.go` — `Job` type: state machine (`Queued → Running → {Done, Failed, Cancelled}`, enforced by `transition()`/`validTransitions`), capped in-memory log ring buffer (500 lines), pub/sub for SSE (`Subscribe`/`publish`, non-blocking sends so a slow client can't stall the subprocess reader).
- `manager.go` — `Manager`: job registry + `jobs.json` persistence + the worker loop that actually runs `mlx_whisper`.
- `runner.go` — `Runner`: spawns a subprocess (own process group via `Setpgid`), streams stdout/stderr line-by-line, supports graceful `Cancel()` (SIGTERM → SIGKILL after 5s) and immediate `ForceKill()` (SIGKILL, used on server shutdown).
- `linesplit.go` — `lineSplitter`, a custom `bufio.SplitFunc` that splits on **both** `\r` and `\n`, because `tqdm` redraws progress bars with a bare `\r`. A stock `bufio.Scanner` would block until the whole subprocess exits before emitting anything.

## Key design points

- **One job runs at a time.** `Manager.queueCh` is drained by a single `workerLoop` goroutine — MLX's unified-memory GPU path gains nothing from concurrent jobs on one Mac; it would only add contention/OOM risk. Don't parallelize without revisiting this.
- **Progress vs. log events are split** in `Manager.dispatchEvent`: a bare-`\r` tqdm redraw is throttled to ~1/sec and published as a `progress` event *without* touching the capped log buffer (otherwise a few seconds of progress-bar spam would evict real early log lines before any client even connects). A real `\n`-terminated line goes to the log buffer, `output.log`, and a `log` SSE event.
- **State transitions are validated** (`validTransitions` in `job.go`) — an invalid transition returns an error instead of silently corrupting state. Terminal states (`Done`/`Failed`/`Cancelled`) never transition further.
- **Cancellation is two-phase**: `CancelJob` on a `Queued` job transitions it directly to `Cancelled`; on a `Running` job it only sets `cancelRequested` and calls `Runner.Cancel()` — the actual `Cancelled` transition happens later in `runJob` once `Run()` returns. Don't assume `CancelJob` returning means the job has actually stopped for a running job.
- **`runJobSafely` wraps every job in a `recover()`** — a panic in one job must never take down the worker loop and silently stop all future jobs.
- **`jobs.json` persistence**: `Manager.persist()` writes the full job list (newest-first) after every state change. `loadPersisted()` on startup marks any job still `Queued`/`Running` as `Failed` ("Interrupted by server restart") — there is no resume-in-place for interrupted jobs.
- **Sidecar copy is best-effort**: after a job completes, `copySrtNextToVideo` writes a copy of the `.srt` next to the source video. A failure here (read-only volume, disk full, permissions) is recorded in `Job.srtSidecarError` but **never** flips a `Done` job to `Failed` — the transcription itself already succeeded.
- **Job IDs double as filenames**: `mlx_whisper` is invoked with `--output-name <job.ID>`, so the primary `.srt` lands at `<OutputDir>/<job.ID>.srt`. Output directories live under `~/Library/Application Support/whisper-gui/jobs/<job.ID>/`, not in this repo.
- **Args are passed as `[]string`, never through a shell** (`exec.Command` in `runner.go`) — filenames with spaces/special characters are handled safely with no manual escaping and no shell-injection risk. Don't introduce `sh -c` here.

## Testing

`manager_test.go` and `runner_test.go` substitute a fake short-lived shell script for `mlx_whisper`/`ffmpeg` (via the `resolveMlx`/`resolveFFmpeg` indirection on `Manager`) so the full state machine and subprocess-cancellation paths are exercised without needing a real install. Follow that pattern for new tests rather than requiring real binaries in CI.
