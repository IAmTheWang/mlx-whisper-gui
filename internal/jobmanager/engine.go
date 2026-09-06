package jobmanager

import "regexp"

const (
	EngineMlx        = "mlx"
	EngineWhisperCpp = "whispercpp"
)

// Engine describes everything that differs between transcription backends:
// how to build the (optional) input-preparation command, how to build the
// main transcription command, and how to recognize a progress line in that
// engine's own output format. Binary resolution and subprocess execution are
// deliberately NOT part of this interface -- Manager keeps its existing
// resolveMlx/resolveFFmpeg-style injected closures (see manager.go) so tests
// can keep substituting fake binaries, and Runner (runner.go) stays the one
// place that actually spawns processes.
type Engine interface {
	Name() string

	// PrepareCommand describes a pre-processing subprocess (run via ffmpeg)
	// needed to produce this engine's input file. ok=false means no
	// preparation is needed and job.VideoPath should be used directly.
	// args excludes the ffmpeg binary path itself, matching NewRunner's
	// convention.
	PrepareCommand(job *Job) (args []string, inputPath string, ok bool)

	// MainArgs returns the transcription command's args (excluding the
	// engine's own binary path), given the resolved input path -- either
	// job.VideoPath directly, or PrepareCommand's inputPath after conversion.
	MainArgs(job *Job, inputPath string) []string

	// ProgressPattern's capture group 1 is a 0-100 percent. It is applied
	// against every RawEvent, not just ones already classified as
	// Kind=="progress", so an engine whose progress lines are '\n'-terminated
	// (whisper-cli) can still be reclassified out of the ordinary log stream.
	ProgressPattern() *regexp.Regexp
}

// engineFor returns the Engine for a job's persisted/requested engine name.
// Unknown or empty names (including jobs persisted before Engine existed)
// fall back to mlx, the app's original and only engine until now.
func engineFor(name string) Engine {
	switch name {
	case EngineWhisperCpp:
		return whisperCppEngine{}
	default:
		return mlxEngine{}
	}
}
