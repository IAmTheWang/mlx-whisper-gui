package jobmanager

import (
	"path/filepath"
	"regexp"
	"strings"

	"whisper-gui/internal/config"
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
	args := []string{
		"-m", job.Model,
		"-f", inputPath,
		"-of", filepath.Join(job.OutputDir, job.ID),
		"-osrt",
		"-pp",
		"-nt",
		"-l", lang,
	}
	// whisper-cli has no equivalent to mlx_whisper's --condition-on-previous-text
	// (it doesn't carry decoded text across windows by default in the first
	// place), but a silent stretch can still make the model hallucinate a
	// plausible-sounding phrase and repeat it -- VAD sidesteps this by
	// skipping non-speech audio before it ever reaches the decoder, rather
	// than transcribing silence and hoping the result is sane. Opt-in only:
	// enabling it requires a separately-downloaded VAD model file, so a job
	// with none configured runs exactly as before.
	cfg, _ := config.Load()
	if cfg.WhisperVadModelPath != "" {
		args = append(args, "--vad", "-vm", cfg.WhisperVadModelPath)
	}
	return args
}

func (whisperCppEngine) ProgressPattern() *regexp.Regexp { return whisperCppPercentRe }
