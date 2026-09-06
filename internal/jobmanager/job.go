package jobmanager

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type State string

const (
	Queued    State = "queued"
	Running   State = "running"
	Done      State = "done"
	Failed    State = "failed"
	Cancelled State = "cancelled"
)

func (s State) Terminal() bool {
	return s == Done || s == Failed || s == Cancelled
}

// LogEvent is a real (\n-terminated, or EOF-flushed) line of subprocess
// output -- these are the ones kept in the replay buffer and output.log.
type LogEvent struct {
	Seq    int64  `json:"seq"`
	Stream string `json:"stream"`
	Text   string `json:"text"`
}

// ProgressEvent is a bare-\r tqdm redraw. Percent is nil when the "NN%|"
// regex didn't match (e.g. a Hugging Face Hub download progress bar in a
// different format) -- Raw is always populated so the frontend can still
// show something instead of looking stuck.
type ProgressEvent struct {
	Percent *int   `json:"percent"`
	Raw     string `json:"raw"`
}

type StateEvent struct {
	State           State  `json:"state"`
	Error           string `json:"error,omitempty"`
	SRTAvailable    bool   `json:"srtAvailable"`
	SRTSidecarPath  string `json:"srtSidecarPath,omitempty"`
	SRTSidecarError string `json:"srtSidecarError,omitempty"`
}

// SSEEvent is what flows down each subscriber channel; Name maps directly
// onto the SSE "event:" field ("log" | "progress" | "state").
type SSEEvent struct {
	Name string
	Data interface{}
}

const logBufferCap = 500

var validTransitions = map[State][]State{
	Queued:  {Running, Cancelled},
	Running: {Done, Failed, Cancelled},
}

type Job struct {
	ID        string
	VideoPath string
	Model     string
	Language  string
	Engine    string
	OutputDir string
	CreatedAt time.Time

	mu              sync.Mutex
	state           State
	startedAt       *time.Time
	finishedAt      *time.Time
	errMsg          string
	srtPath         string
	srtSidecarPath  string
	srtSidecarError string
	cancelRequested bool

	seq          int64
	logBuf       []LogEvent
	subs         map[chan SSEEvent]struct{}
	lastProgress time.Time
	runner       *Runner
}

func newJob(id, videoPath, model, language, engine, outputDir string, createdAt time.Time) *Job {
	return &Job{
		ID:        id,
		VideoPath: videoPath,
		Model:     model,
		Language:  language,
		Engine:    engine,
		OutputDir: outputDir,
		CreatedAt: createdAt,
		state:     Queued,
		subs:      make(map[chan SSEEvent]struct{}),
	}
}

func (j *Job) State() State {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.state
}

// Snapshot is the JSON-serializable view of a job, used by the API and
// persisted to jobs.json -- deliberately excludes runner/subs/logBuf.
type Snapshot struct {
	ID              string     `json:"id"`
	VideoPath       string     `json:"videoPath"`
	Model           string     `json:"model"`
	Language        string     `json:"language"`
	Engine          string     `json:"engine"`
	OutputDir       string     `json:"outputDir"`
	State           State      `json:"state"`
	CreatedAt       time.Time  `json:"createdAt"`
	StartedAt       *time.Time `json:"startedAt,omitempty"`
	FinishedAt      *time.Time `json:"finishedAt,omitempty"`
	Error           string     `json:"error,omitempty"`
	SRTPath         string     `json:"srtPath,omitempty"`
	SRTSidecarPath  string     `json:"srtSidecarPath,omitempty"`
	SRTSidecarError string     `json:"srtSidecarError,omitempty"`
}

func (j *Job) Snapshot() Snapshot {
	j.mu.Lock()
	defer j.mu.Unlock()
	return Snapshot{
		ID: j.ID, VideoPath: j.VideoPath, Model: j.Model, Language: j.Language, Engine: j.Engine,
		OutputDir: j.OutputDir, State: j.state, CreatedAt: j.CreatedAt,
		StartedAt: j.startedAt, FinishedAt: j.finishedAt, Error: j.errMsg, SRTPath: j.srtPath,
		SRTSidecarPath: j.srtSidecarPath, SRTSidecarError: j.srtSidecarError,
	}
}

func (j *Job) transition(to State, mutate func()) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	allowed := false
	for _, s := range validTransitions[j.state] {
		if s == to {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("invalid job state transition %s -> %s", j.state, to)
	}
	j.state = to
	if mutate != nil {
		mutate()
	}
	return nil
}

func (j *Job) markRunning(r *Runner) error {
	now := time.Now()
	return j.transition(Running, func() {
		j.startedAt = &now
		j.runner = r
	})
}

// markDone also sets the sidecar-copy outcome atomically with the Done
// transition itself (rather than via a separate setSidecar call afterwards)
// so a concurrent reader can never observe a Done job with stale/missing
// sidecar info.
func (j *Job) markDone(srtPath, sidecarPath, sidecarErr string) error {
	now := time.Now()
	return j.transition(Done, func() {
		j.finishedAt = &now
		j.srtPath = srtPath
		j.srtSidecarPath = sidecarPath
		j.srtSidecarError = sidecarErr
	})
}

func (j *Job) markFailed(errMsg string) error {
	now := time.Now()
	return j.transition(Failed, func() {
		j.finishedAt = &now
		j.errMsg = errMsg
	})
}

func (j *Job) markCancelled() error {
	now := time.Now()
	return j.transition(Cancelled, func() {
		j.finishedAt = &now
	})
}

func (j *Job) requestCancel() {
	j.mu.Lock()
	j.cancelRequested = true
	j.mu.Unlock()
}

func (j *Job) wasCancelRequested() bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.cancelRequested
}

func (j *Job) runnerRef() *Runner {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.runner
}

// replaceRunner swaps the active runner reference without a state transition.
// Some engines run more than one subprocess phase while the job stays
// Running (e.g. whisper.cpp's ffmpeg-prepare phase followed by its main
// whisper-cli phase) -- markRunning only fires on the Queued->Running
// transition, so later phases use this instead to keep Cancel() working
// against whichever subprocess is actually active.
func (j *Job) replaceRunner(r *Runner) {
	j.mu.Lock()
	j.runner = r
	j.mu.Unlock()
}

// appendLog records a real log line in the capped in-memory ring buffer and
// returns its sequence number (monotonic within this job, used by the
// frontend to dedupe lines across an SSE reconnect).
func (j *Job) appendLog(stream, text string) int64 {
	j.mu.Lock()
	j.seq++
	seq := j.seq
	j.logBuf = append(j.logBuf, LogEvent{Seq: seq, Stream: stream, Text: text})
	if len(j.logBuf) > logBufferCap {
		j.logBuf = j.logBuf[len(j.logBuf)-logBufferCap:]
	}
	j.mu.Unlock()
	return seq
}

func (j *Job) tailOfLog(n int) string {
	j.mu.Lock()
	defer j.mu.Unlock()
	start := 0
	if len(j.logBuf) > n {
		start = len(j.logBuf) - n
	}
	var sb strings.Builder
	for _, e := range j.logBuf[start:] {
		sb.WriteString(e.Text)
		sb.WriteByte('\n')
	}
	return sb.String()
}

// ReadLogFile reads the persisted output.log for this job -- used to show a
// historical job's full log after a server restart, since the in-memory
// ring buffer above is gone once the process restarts.
func (j *Job) ReadLogFile() ([]byte, error) {
	return os.ReadFile(filepath.Join(j.OutputDir, "output.log"))
}

// Subscribe registers a new SSE subscriber and returns a snapshot of the
// current replay buffer captured atomically with the subscription, so no
// event published between the snapshot and the caller's first receive can
// be missed or duplicated.
func (j *Job) Subscribe() (ch chan SSEEvent, replay []LogEvent, unsubscribe func()) {
	ch = make(chan SSEEvent, 32)
	j.mu.Lock()
	j.subs[ch] = struct{}{}
	replay = append([]LogEvent(nil), j.logBuf...)
	j.mu.Unlock()
	unsubscribe = func() {
		j.mu.Lock()
		delete(j.subs, ch)
		j.mu.Unlock()
	}
	return ch, replay, unsubscribe
}

// publish fans an event out to all current subscribers. Sends are
// non-blocking: a slow SSE client can't stall the goroutine reading the
// subprocess's stdout/stderr.
func (j *Job) publish(evt SSEEvent) {
	j.mu.Lock()
	subs := make([]chan SSEEvent, 0, len(j.subs))
	for ch := range j.subs {
		subs = append(subs, ch)
	}
	j.mu.Unlock()
	for _, ch := range subs {
		select {
		case ch <- evt:
		default:
		}
	}
}
