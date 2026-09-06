package jobmanager

import (
	"path/filepath"
	"regexp"
	"strings"
)

// whisperCppPercentRe matches whisper.cpp's own progress line, e.g.
// "whisper_print_progress_callback: progress = 42%" -- printed to stderr,
// '\n'-terminated, and only emitted when whisper-cli is passed -pp.
var whisperCppPercentRe = regexp.MustCompile(`progress\s*=\s*(\d{1,3})%`)

// whisperCppEngine wraps whisper.cpp's whisper-cli binary. Unlike mlx_whisper,
// whisper-cli's plain build (as distributed by `brew install whisper-cpp`)
// cannot decode video containers itself, so a video source needs an explicit
// ffmpeg pre-conversion to 16kHz mono WAV before whisper-cli ever runs.
type whisperCppEngine struct{}

func (whisperCppEngine) Name() string { return "whisper.cpp" }

func (whisperCppEngine) PrepareCommand(job *Job) (args []string, inputPath string, ok bool) {
	// Skip the conversion entirely if the source is already WAV -- a simple
	// extension check, not a real header sniff, is enough here (YAGNI).
	if strings.EqualFold(filepath.Ext(job.VideoPath), ".wav") {
		return nil, "", false
	}
	wav := filepath.Join(job.OutputDir, job.ID+".wav")
	return []string{"-y", "-i", job.VideoPath, "-ar", "16000", "-ac", "1", "-c:a", "pcm_s16le", wav}, wav, true
}

func (whisperCppEngine) MainArgs(job *Job, inputPath string) []string {
	// whisper-cli defaults -l to "en", not autodetect -- unlike mlx_whisper,
	// omitting the flag does NOT mean autodetect, so "" must become "auto".
	lang := job.Language
	if lang == "" {
		lang = "auto"
	}
	return []string{
		"-m", job.Model,
		"-f", inputPath,
		"-of", filepath.Join(job.OutputDir, job.ID),
		"-osrt",
		"-pp",
		"-nt",
		"-l", lang,
	}
}

func (whisperCppEngine) ProgressPattern() *regexp.Regexp { return whisperCppPercentRe }
