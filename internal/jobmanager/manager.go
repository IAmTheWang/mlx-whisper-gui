package jobmanager

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"whisper-gui/internal/config"
	"whisper-gui/internal/whisperbin"
)

var (
	ErrNotFound        = errors.New("job not found")
	ErrAlreadyTerminal = errors.New("job already in a terminal state")
)

type CreateRequest struct {
	VideoPath string
	Model     string
	Language  string
	Engine    string
}

// Manager owns the job registry, the single-worker FIFO queue (MLX's
// unified-memory GPU path gains nothing from running two transcriptions
// concurrently on one Mac -- it would only add contention/OOM risk), and
// jobs.json persistence.
type Manager struct {
	baseDir string // ~/Library/Application Support/whisper-gui

	mu    sync.Mutex
	jobs  map[string]*Job
	order []string // creation order, oldest first

	nextID  int64
	queueCh chan string

	// Indirection points at whisperbin.LocateMlxWhisper/LocateFFmpeg/
	// LocateWhisperCli in production; tests substitute a fake short-lived
	// binary so the state machine can be exercised without a real
	// mlx_whisper/ffmpeg/whisper-cli install.
	resolveMlx        func() whisperbin.Resolution
	resolveFFmpeg     func() whisperbin.Resolution
	resolveWhisperCli func() whisperbin.Resolution

	closeOnce sync.Once
	closed    chan struct{}
}

func NewManager() (*Manager, error) {
	dir, err := config.Dir()
	if err != nil {
		return nil, err
	}
	return newManagerWithBaseDir(dir, whisperbin.LocateMlxWhisper, whisperbin.LocateFFmpeg, whisperbin.LocateWhisperCli)
}

func newManagerWithBaseDir(baseDir string, resolveMlx, resolveFFmpeg, resolveWhisperCli func() whisperbin.Resolution) (*Manager, error) {
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, err
	}
	m := &Manager{
		baseDir:           baseDir,
		jobs:              make(map[string]*Job),
		queueCh:           make(chan string, 256),
		closed:            make(chan struct{}),
		resolveMlx:        resolveMlx,
		resolveFFmpeg:     resolveFFmpeg,
		resolveWhisperCli: resolveWhisperCli,
	}
	m.loadPersisted()
	go m.workerLoop()
	return m, nil
}

func (m *Manager) jobsJSONPath() string {
	return filepath.Join(m.baseDir, "jobs.json")
}

func (m *Manager) loadPersisted() {
	data, err := os.ReadFile(m.jobsJSONPath())
	if err != nil {
		return // no history yet -- fine on first run
	}
	var snaps []Snapshot
	if err := json.Unmarshal(data, &snaps); err != nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	// snaps was persisted newest-first; restore in that order, then reverse
	// m.order back to oldest-first so ListJobs' own newest-first pass stays correct.
	for i := len(snaps) - 1; i >= 0; i-- {
		s := snaps[i]
		engine := s.Engine
		if engine == "" {
			engine = EngineMlx // back-compat: jobs.json entries persisted before Engine existed
		}
		j := newJob(s.ID, s.VideoPath, s.Model, s.Language, engine, s.OutputDir, s.CreatedAt)
		j.state = s.State
		j.startedAt = s.StartedAt
		j.finishedAt = s.FinishedAt
		j.errMsg = s.Error
		j.srtPath = s.SRTPath
		if j.state == Queued || j.state == Running {
			now := time.Now()
			j.state = Failed
			j.errMsg = "Interrupted by server restart"
			j.finishedAt = &now
		}
		m.jobs[j.ID] = j
		m.order = append(m.order, j.ID)
	}
}

func (m *Manager) persist() {
	m.mu.Lock()
	snaps := make([]Snapshot, 0, len(m.order))
	for i := len(m.order) - 1; i >= 0; i-- { // newest first, matches ListJobs
		snaps = append(snaps, m.jobs[m.order[i]].Snapshot())
	}
	m.mu.Unlock()

	data, err := json.MarshalIndent(snaps, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(m.jobsJSONPath(), data, 0o644)
}

func (m *Manager) CreateJob(req CreateRequest) (*Job, error) {
	if !filepath.IsAbs(req.VideoPath) {
		return nil, fmt.Errorf("videoPath must be an absolute path")
	}
	info, err := os.Stat(req.VideoPath)
	if err != nil {
		return nil, fmt.Errorf("video file not found: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("videoPath is a directory, not a file")
	}

	engine := req.Engine
	if engine == "" {
		engine = EngineMlx
	}

	m.mu.Lock()
	m.nextID++
	id := fmt.Sprintf("job_%d_%d", time.Now().UnixNano(), m.nextID)
	outputDir := filepath.Join(m.baseDir, "jobs", id)
	job := newJob(id, req.VideoPath, req.Model, req.Language, engine, outputDir, time.Now())
	m.jobs[id] = job
	m.order = append(m.order, id)
	m.mu.Unlock()

	m.persist()
	m.queueCh <- id
	return job, nil
}

func (m *Manager) GetJob(id string) (*Job, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	j, ok := m.jobs[id]
	return j, ok
}

func (m *Manager) ListJobs() []Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Snapshot, 0, len(m.order))
	for i := len(m.order) - 1; i >= 0; i-- {
		out = append(out, m.jobs[m.order[i]].Snapshot())
	}
	return out
}

func (m *Manager) CancelJob(id string) error {
	job, ok := m.GetJob(id)
	if !ok {
		return ErrNotFound
	}
	switch job.State() {
	case Queued:
		if err := job.markCancelled(); err != nil {
			return err
		}
		m.publishState(job)
		m.persist()
		return nil
	case Running:
		job.requestCancel()
		if r := job.runnerRef(); r != nil {
			r.Cancel()
		}
		// The actual Cancelled transition happens in runJob once Run()
		// returns -- Cancel() here only requests termination.
		return nil
	default:
		return ErrAlreadyTerminal
	}
}

func (m *Manager) workerLoop() {
	for {
		select {
		case id, ok := <-m.queueCh:
			if !ok {
				return
			}
			m.runJobSafely(id)
		case <-m.closed:
			return
		}
	}
}

// runJobSafely wraps runJob in a recover() so a panic while running one job
// can never take down the worker loop and silently stop all future jobs.
func (m *Manager) runJobSafely(id string) {
	job, ok := m.GetJob(id)
	if !ok {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			_ = job.markFailed(fmt.Sprintf("Internal error: %v", r))
			m.publishState(job)
			m.persist()
		}
	}()

	if job.State() != Queued {
		return // e.g. cancelled while still queued
	}
	m.runJob(job)
}

func (m *Manager) runJob(job *Job) {
	eng := engineFor(job.Engine)

	var bin whisperbin.Resolution
	switch job.Engine {
	case EngineWhisperCpp:
		bin = m.resolveWhisperCli()
	default:
		bin = m.resolveMlx()
	}
	if bin.ResolvedVia == whisperbin.ViaNone {
		m.failJob(job, eng.Name()+" not found, please configure its path in Settings")
		return
	}
	ffmpeg := m.resolveFFmpeg() // required by both: mlx uses it internally, whispercpp invokes it directly below
	if ffmpeg.ResolvedVia == whisperbin.ViaNone {
		if job.Engine == EngineWhisperCpp {
			m.failJob(job, "ffmpeg not found -- whisper-cli needs it to convert the video into 16kHz mono WAV first. Please configure ffmpeg's path in Settings.")
		} else {
			m.failJob(job, "ffmpeg not found, please run: brew install ffmpeg")
		}
		return
	}

	if err := os.MkdirAll(job.OutputDir, 0o755); err != nil {
		m.failJob(job, "Failed to create output directory: "+err.Error())
		return
	}

	logFile, err := os.OpenFile(filepath.Join(job.OutputDir, "output.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		m.failJob(job, "Failed to create log file: "+err.Error())
		return
	}
	defer logFile.Close()

	inputPath := job.VideoPath
	if prepArgs, prepInput, ok := eng.PrepareCommand(job); ok {
		// Register the cleanup as soon as the path is known, before running
		// anything -- this way a cancel or an error mid-conversion still
		// removes the (possibly partial) temp WAV, not just a clean finish.
		defer os.Remove(prepInput)

		prepRunner := NewRunner(ffmpeg.Path, prepArgs, job.OutputDir)
		if err := job.markRunning(prepRunner); err != nil {
			return
		}
		m.publishState(job)
		m.persist()

		prepErr := prepRunner.Run(func(e RawEvent) {
			m.dispatchEvent(job, eng, logFile, e)
		})

		if job.wasCancelRequested() {
			_ = job.markCancelled()
			m.publishState(job)
			m.persist()
			return
		}
		if prepErr != nil {
			m.failJob(job, "Audio preprocessing (ffmpeg) failed: "+prepErr.Error())
			return
		}
		inputPath = prepInput
	}

	args := eng.MainArgs(job, inputPath)
	runner := NewRunner(bin.Path, args, job.OutputDir)
	if job.State() == Queued {
		if err := job.markRunning(runner); err != nil {
			return
		}
	} else {
		job.replaceRunner(runner)
	}
	m.publishState(job)
	m.persist()

	runErr := runner.Run(func(e RawEvent) {
		m.dispatchEvent(job, eng, logFile, e)
	})

	if job.wasCancelRequested() {
		_ = job.markCancelled()
		m.publishState(job)
		m.persist()
		return
	}

	if runErr != nil {
		tail := job.tailOfLog(20)
		m.failJob(job, fmt.Sprintf("%s process exited with an error: %v\n%s", eng.Name(), runErr, tail))
		return
	}

	expected := filepath.Join(job.OutputDir, job.ID+".srt")
	if _, statErr := os.Stat(expected); statErr != nil {
		m.failJob(job, "Process exited successfully but the expected .srt file was not found (possibly out of disk space)")
		return
	}

	dest, copyErr := copySrtNextToVideo(job.VideoPath, expected)
	sidecarErr := ""
	if copyErr != nil {
		sidecarErr = copyErr.Error()
		dest = ""
	}
	_ = job.markDone(expected, dest, sidecarErr)
	m.publishState(job)
	m.persist()
}

func (m *Manager) failJob(job *Job, msg string) {
	_ = job.markFailed(msg)
	m.publishState(job)
	m.persist()
}

func (m *Manager) publishState(job *Job) {
	snap := job.Snapshot()
	job.publish(SSEEvent{Name: "state", Data: StateEvent{
		State:           snap.State,
		Error:           snap.Error,
		SRTAvailable:    snap.SRTPath != "",
		SRTSidecarPath:  snap.SRTSidecarPath,
		SRTSidecarError: snap.SRTSidecarError,
	}})
}

// copySrtNextToVideo writes a copy of the generated .srt beside the source
// video (same directory, named after the video) so the user finds the
// subtitle file right where the video lives, with no extra click needed.
// This is a best-effort side effect on top of an already-successful
// transcription -- a failure here (read-only volume, permission denied,
// disk full) must never turn a Done job into a Failed one; it's surfaced
// separately via Job.srtSidecarError instead.
func copySrtNextToVideo(videoPath, srtPath string) (string, error) {
	base := strings.TrimSuffix(filepath.Base(videoPath), filepath.Ext(videoPath))
	dest := filepath.Join(filepath.Dir(videoPath), base+".srt")

	src, err := os.Open(srtPath)
	if err != nil {
		return "", err
	}
	defer src.Close()

	out, err := os.Create(dest)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err := io.Copy(out, src); err != nil {
		out.Close()
		os.Remove(dest)
		return "", err
	}
	return dest, nil
}

// dispatchEvent implements the core log/progress split: a line matching the
// engine's own ProgressPattern (a bare-\r tqdm redraw for mlx_whisper, or a
// \n-terminated "progress = NN%" line for whisper-cli) is throttled to
// ~1/sec and published as a "progress" event without ever touching the
// capped replay buffer (otherwise a few seconds of progress-bar spam would
// evict genuinely important early log lines before any client connects);
// anything else is buffered, persisted to output.log, and published as a
// "log" event.
func (m *Manager) dispatchEvent(job *Job, eng Engine, logFile *os.File, e RawEvent) {
	match := eng.ProgressPattern().FindStringSubmatch(e.Text)
	if match != nil {
		e.Kind = "progress"
	}

	if e.Kind == "progress" {
		job.mu.Lock()
		skip := !job.lastProgress.IsZero() && time.Since(job.lastProgress) < time.Second
		if !skip {
			job.lastProgress = time.Now()
		}
		job.mu.Unlock()
		if skip {
			return
		}

		var percent *int
		if match != nil {
			if n, err := strconv.Atoi(match[1]); err == nil && n >= 0 && n <= 100 {
				percent = &n
			}
		}
		// percent stays nil (not dropped) when the regex doesn't match, or
		// matched an out-of-range value -- e.g. a Hugging Face Hub download
		// progress bar uses a different tqdm format than mlx_whisper's own
		// transcription progress.
		job.publish(SSEEvent{Name: "progress", Data: ProgressEvent{Percent: percent, Raw: e.Text}})
		return
	}

	seq := job.appendLog(e.Stream, e.Text)
	fmt.Fprintf(logFile, "[%s] %s\n", e.Stream, e.Text)
	job.publish(SSEEvent{Name: "log", Data: LogEvent{Seq: seq, Stream: e.Stream, Text: e.Text}})
}

// Close performs a graceful shutdown: any currently running job's process
// group is force-killed immediately (the server itself is exiting right
// now, so there's no reason to wait through Cancel's grace period), leaving
// no orphaned mlx_whisper process holding onto GPU unified memory.
func (m *Manager) Close() {
	m.closeOnce.Do(func() {
		m.mu.Lock()
		var running *Job
		for _, j := range m.jobs {
			if j.State() == Running {
				running = j
				break
			}
		}
		m.mu.Unlock()
		if running != nil {
			if r := running.runnerRef(); r != nil {
				r.ForceKill()
			}
		}
		close(m.closed)
	})
}
