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
		// This IS whisper.cpp's real equivalent of mlx_whisper's
		// --condition-on-previous-text False. --no-context alone does NOT
		// achieve this: it only clears carried-over state once, at the very
		// start of the whisper_full() call; within that same call, each
		// subsequent 30s window's decoded text still gets fed back in as the
		// next window's prompt (src/whisper.cpp's prompt_past1, updated and
		// read unconditionally after every window, gated only by
		// n_max_text_ctx > 0). -mc 0 sets n_max_text_ctx to 0, which skips
		// that injection entirely -- confirmed empirically: the same
		// recording that produced ~22 minutes of "ご視聴ありがとうございました"
		// repeated verbatim (once the model mistranscribed one window)
		// transcribed correctly end-to-end with -mc 0.
		"-mc", "0",
	}
	// VAD is a separate, complementary measure -- it skips genuinely silent
	// stretches before they ever reach the decoder (faster, and avoids the
	// *first* bad guess in the first place), whereas -mc 0 above stops one
	// bad guess from being *carried forward* into every subsequent window
	// regardless of cause. Opt-in only: enabling it requires a
	// separately-downloaded VAD model file, so a job with none configured
	// just skips these two flags.
	cfg, _ := config.Load()
	if cfg.WhisperVadModelPath != "" {
		args = append(args, "--vad", "-vm", cfg.WhisperVadModelPath)
	}
	return args
}

func (whisperCppEngine) ProgressPattern() *regexp.Regexp { return whisperCppPercentRe }
