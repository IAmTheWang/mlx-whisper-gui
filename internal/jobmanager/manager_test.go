package jobmanager

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"whisper-gui/internal/whisperbin"
)

func newTestManager(t *testing.T, mlxScriptPath string) *Manager {
	t.Helper()
	return newTestManagerFull(t, mlxScriptPath, "/usr/bin/true", "")
}

// newTestManagerFull is the general form: an empty whisperCliScriptPath
// resolves whisper-cli as ViaNone (not found), matching a machine that never
// installed it -- most tests only care about the mlx_whisper path and use
// newTestManager instead.
func newTestManagerFull(t *testing.T, mlxScriptPath, ffmpegScriptPath, whisperCliScriptPath string) *Manager {
	t.Helper()
	m, err := newManagerWithBaseDir(t.TempDir(),
		func() whisperbin.Resolution {
			return whisperbin.Resolution{Path: mlxScriptPath, ResolvedVia: whisperbin.ViaPath}
		},
		func() whisperbin.Resolution {
			return whisperbin.Resolution{Path: ffmpegScriptPath, ResolvedVia: whisperbin.ViaPath}
		},
		func() whisperbin.Resolution {
			if whisperCliScriptPath == "" {
				return whisperbin.Resolution{ResolvedVia: whisperbin.ViaNone}
			}
			return whisperbin.Resolution{Path: whisperCliScriptPath, ResolvedVia: whisperbin.ViaPath}
		},
	)
	if err != nil {
		t.Fatalf("newManagerWithBaseDir: %v", err)
	}
	return m
}

// writeFakeMlxWhisper generates a stand-in for the real mlx_whisper CLI: it
// parses --output-dir/--output-name the same way the real tool does (so
// runJob's "does the expected .srt exist?" check can be exercised), sleeps
// for a controllable duration (to simulate a job that's still running), and
// exits with a controllable code.
func writeFakeMlxWhisper(t *testing.T, sleepSeconds float64, exitCode int, createSRT bool) string {
	t.Helper()
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "fake_mlx_whisper.sh")
	createLine := `[ -n "$outdir" ] && [ -n "$outname" ] && echo "fake transcript" > "$outdir/$outname.srt"`
	if !createSRT {
		createLine = ":"
	}
	content := fmt.Sprintf(`#!/bin/sh
outdir=""
outname=""
while [ $# -gt 0 ]; do
  case "$1" in
    --output-dir) outdir="$2"; shift 2 ;;
    --output-name) outname="$2"; shift 2 ;;
    *) shift ;;
  esac
done
echo "loading model"
sleep %v
%s
exit %d
`, sleepSeconds, createLine, exitCode)
	if err := os.WriteFile(scriptPath, []byte(content), 0o755); err != nil {
		t.Fatalf("write fake script: %v", err)
	}
	return scriptPath
}

// writeFakeWhisperCli generates a stand-in for whisper-cli: it parses -of the
// same way the real tool does (each -o* flag appends its own extension, so
// -osrt + -of <path> writes <path>.srt), sleeps for a controllable duration,
// and exits with a controllable code. It deliberately ignores -f/-m -- these
// tests exercise Manager's orchestration, not whisper-cli's actual behavior.
func writeFakeWhisperCli(t *testing.T, sleepSeconds float64, exitCode int, createSRT bool) string {
	t.Helper()
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "fake_whisper_cli.sh")
	createLine := `[ -n "$of" ] && echo "fake transcript" > "$of.srt"`
	if !createSRT {
		createLine = ":"
	}
	content := fmt.Sprintf(`#!/bin/sh
of=""
while [ $# -gt 0 ]; do
  case "$1" in
    -of) of="$2"; shift 2 ;;
    *) shift ;;
  esac
done
sleep %v
%s
exit %d
`, sleepSeconds, createLine, exitCode)
	if err := os.WriteFile(scriptPath, []byte(content), 0o755); err != nil {
		t.Fatalf("write fake script: %v", err)
	}
	return scriptPath
}

// writeFakeFFmpeg generates a stand-in for the ffmpeg preprocessing step in
// whisperCppEngine.PrepareCommand: it ignores its args entirely and just
// sleeps for a controllable duration, so tests can exercise cancellation
// while that phase is "running".
func writeFakeFFmpeg(t *testing.T, sleepSeconds float64) string {
	t.Helper()
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "fake_ffmpeg.sh")
	content := fmt.Sprintf("#!/bin/sh\nsleep %v\nexit 0\n", sleepSeconds)
	if err := os.WriteFile(scriptPath, []byte(content), 0o755); err != nil {
		t.Fatalf("write fake script: %v", err)
	}
	return scriptPath
}

func waitForState(t *testing.T, job *Job, timeout time.Duration) State {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if s := job.State(); s.Terminal() {
			return s
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("job %s did not reach a terminal state within %v (stuck at %s)", job.ID, timeout, job.State())
	return ""
}

func TestManager_HappyPath_QueuedRunningDone(t *testing.T) {
	script := writeFakeMlxWhisper(t, 0, 0, true)
	m := newTestManager(t, script)

	job, err := m.CreateJob(CreateRequest{VideoPath: mustTempVideo(t), Model: "whisper-large-v3-turbo", Language: "ja"})
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}

	final := waitForState(t, job, 5*time.Second)
	if final != Done {
		t.Fatalf("expected job to end Done, got %s (error: %s)", final, job.Snapshot().Error)
	}
	snap := job.Snapshot()
	if snap.SRTPath == "" {
		t.Fatal("expected SRTPath to be set on a Done job")
	}
	if _, err := os.Stat(snap.SRTPath); err != nil {
		t.Fatalf("expected srt file to exist at %s: %v", snap.SRTPath, err)
	}
}

func TestManager_Done_CopiesSrtNextToVideo(t *testing.T) {
	script := writeFakeMlxWhisper(t, 0, 0, true)
	m := newTestManager(t, script)

	videoPath := mustTempVideo(t) // .../clip.mp4
	job, err := m.CreateJob(CreateRequest{VideoPath: videoPath, Model: "m", Language: "ja"})
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}

	if final := waitForState(t, job, 5*time.Second); final != Done {
		t.Fatalf("expected Done, got %s", final)
	}

	snap := job.Snapshot()
	wantSidecar := filepath.Join(filepath.Dir(videoPath), "clip.srt")
	if snap.SRTSidecarPath != wantSidecar {
		t.Fatalf("expected sidecar path %s, got %q (sidecar error: %q)", wantSidecar, snap.SRTSidecarPath, snap.SRTSidecarError)
	}
	if _, err := os.Stat(wantSidecar); err != nil {
		t.Fatalf("expected sidecar srt to exist next to the video: %v", err)
	}
	if snap.SRTSidecarError != "" {
		t.Fatalf("expected no sidecar error, got %q", snap.SRTSidecarError)
	}
}

func TestManager_Done_SidecarCopyFailureDoesNotFailJob(t *testing.T) {
	script := writeFakeMlxWhisper(t, 0, 0, true)
	m := newTestManager(t, script)

	// Put the video in a directory with no write permission, so the
	// best-effort sidecar copy fails -- this must NOT turn the otherwise
	// successful transcription into a Failed job.
	dir := t.TempDir()
	videoPath := filepath.Join(dir, "clip.mp4")
	if err := os.WriteFile(videoPath, []byte("fake"), 0o644); err != nil {
		t.Fatalf("write fake video: %v", err)
	}
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatalf("chmod dir read-only: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) }) // let t.TempDir() clean up afterwards

	job, err := m.CreateJob(CreateRequest{VideoPath: videoPath, Model: "m", Language: "ja"})
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}

	final := waitForState(t, job, 5*time.Second)
	if final != Done {
		t.Fatalf("expected Done despite sidecar copy failure, got %s", final)
	}
	snap := job.Snapshot()
	if snap.SRTSidecarError == "" {
		t.Fatal("expected a sidecar error to be recorded when the video's directory is read-only")
	}
	if snap.SRTSidecarPath != "" {
		t.Fatalf("expected no sidecar path on failure, got %q", snap.SRTSidecarPath)
	}
}

func TestManager_ExitZeroButNoSRT_IsFailed(t *testing.T) {
	script := writeFakeMlxWhisper(t, 0, 0, false) // exits 0 but never writes the .srt
	m := newTestManager(t, script)

	job, err := m.CreateJob(CreateRequest{VideoPath: mustTempVideo(t), Model: "m", Language: "ja"})
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}

	final := waitForState(t, job, 5*time.Second)
	if final != Failed {
		t.Fatalf("expected exit-0-but-missing-file to be reported as Failed, got %s", final)
	}
}

func TestManager_NonZeroExit_IsFailed(t *testing.T) {
	script := writeFakeMlxWhisper(t, 0, 1, false)
	m := newTestManager(t, script)

	job, err := m.CreateJob(CreateRequest{VideoPath: mustTempVideo(t), Model: "m", Language: "ja"})
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}

	final := waitForState(t, job, 5*time.Second)
	if final != Failed {
		t.Fatalf("expected non-zero exit to be Failed, got %s", final)
	}
}

func TestManager_CancelWhileQueued_NeverRuns(t *testing.T) {
	blocking := writeFakeMlxWhisper(t, 2, 0, true)
	m := newTestManager(t, blocking)

	first, err := m.CreateJob(CreateRequest{VideoPath: mustTempVideo(t), Model: "m", Language: "ja"})
	if err != nil {
		t.Fatalf("CreateJob first: %v", err)
	}
	// give the worker a moment to pick up `first` so it occupies the single slot
	time.Sleep(100 * time.Millisecond)
	if first.State() != Running {
		t.Fatalf("expected first job to be Running by now, got %s", first.State())
	}

	second, err := m.CreateJob(CreateRequest{VideoPath: mustTempVideo(t), Model: "m", Language: "ja"})
	if err != nil {
		t.Fatalf("CreateJob second: %v", err)
	}
	if second.State() != Queued {
		t.Fatalf("expected second job to be Queued while first is Running, got %s", second.State())
	}

	if err := m.CancelJob(second.ID); err != nil {
		t.Fatalf("CancelJob(second): %v", err)
	}
	if second.State() != Cancelled {
		t.Fatalf("expected second job to be Cancelled immediately, got %s", second.State())
	}

	// clean up: let/force the first job to finish so it doesn't outlive the test
	_ = m.CancelJob(first.ID)
	waitForState(t, first, 5*time.Second)

	// second must never have actually run (no srt for it)
	if second.Snapshot().SRTPath != "" {
		t.Fatal("a job cancelled while queued should never have produced an srt")
	}
}

func TestManager_CancelWhileRunning_Terminates(t *testing.T) {
	longRunning := writeFakeMlxWhisper(t, 30, 0, true)
	m := newTestManager(t, longRunning)

	job, err := m.CreateJob(CreateRequest{VideoPath: mustTempVideo(t), Model: "m", Language: "ja"})
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}
	time.Sleep(150 * time.Millisecond)
	if job.State() != Running {
		t.Fatalf("expected job to be Running, got %s", job.State())
	}

	if err := m.CancelJob(job.ID); err != nil {
		t.Fatalf("CancelJob: %v", err)
	}

	final := waitForState(t, job, 5*time.Second) // well before the fake's 30s sleep would finish on its own
	if final != Cancelled {
		t.Fatalf("expected Cancelled, got %s", final)
	}
}

func TestManager_WhisperCpp_HappyPath_QueuedRunningDone(t *testing.T) {
	whisperCli := writeFakeWhisperCli(t, 0, 0, true)
	m := newTestManagerFull(t, "/usr/bin/true", "/usr/bin/true", whisperCli)

	job, err := m.CreateJob(CreateRequest{VideoPath: mustTempVideo(t), Model: "/models/ggml-small.bin", Language: "ja", Engine: EngineWhisperCpp})
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}

	final := waitForState(t, job, 5*time.Second)
	if final != Done {
		t.Fatalf("expected job to end Done, got %s (error: %s)", final, job.Snapshot().Error)
	}
	snap := job.Snapshot()
	if snap.Engine != EngineWhisperCpp {
		t.Fatalf("expected Engine %q on the snapshot, got %q", EngineWhisperCpp, snap.Engine)
	}
	if snap.SRTPath == "" {
		t.Fatal("expected SRTPath to be set on a Done job")
	}
	if _, err := os.Stat(snap.SRTPath); err != nil {
		t.Fatalf("expected srt file to exist at %s: %v", snap.SRTPath, err)
	}
}

// TestManager_WhisperCpp_CancelDuringPrepare proves cancellation works even
// while the job is still in whisper.cpp's ffmpeg-prepare phase, before the
// main whisper-cli runner has been created -- this exercises Job.replaceRunner,
// since markRunning only fires once, on the Queued->Running transition.
func TestManager_WhisperCpp_CancelDuringPrepare(t *testing.T) {
	slowFFmpeg := writeFakeFFmpeg(t, 30)
	whisperCli := writeFakeWhisperCli(t, 0, 0, true)
	m := newTestManagerFull(t, "/usr/bin/true", slowFFmpeg, whisperCli)

	job, err := m.CreateJob(CreateRequest{VideoPath: mustTempVideo(t), Model: "/models/ggml-small.bin", Language: "ja", Engine: EngineWhisperCpp})
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}
	time.Sleep(150 * time.Millisecond)
	if job.State() != Running {
		t.Fatalf("expected job to be Running (in the prepare phase), got %s", job.State())
	}

	if err := m.CancelJob(job.ID); err != nil {
		t.Fatalf("CancelJob: %v", err)
	}

	final := waitForState(t, job, 5*time.Second) // well before the fake ffmpeg's 30s sleep would finish on its own
	if final != Cancelled {
		t.Fatalf("expected Cancelled, got %s", final)
	}
}

func mustTempVideo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "clip.mp4")
	if err := os.WriteFile(p, []byte("not a real video, just needs to exist"), 0o644); err != nil {
		t.Fatalf("write fake video: %v", err)
	}
	return p
}
