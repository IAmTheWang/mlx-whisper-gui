package jobmanager

import "regexp"

// mlxPercentRe matches tqdm's own progress-bar format, e.g. "42%|####  |".
var mlxPercentRe = regexp.MustCompile(`(\d{1,3})%\|`)

// mlxEngine wraps mlx_whisper, the app's original engine. It decodes the
// source video itself (via an internal ffmpeg call Go never sees), so it
// needs no separate preparation step.
type mlxEngine struct{}

func (mlxEngine) Name() string { return "mlx_whisper" }

func (mlxEngine) PrepareCommand(job *Job) (args []string, inputPath string, ok bool) {
	return nil, "", false
}

func (mlxEngine) MainArgs(job *Job, inputPath string) []string {
	args := []string{
		"--model", job.Model,
		"--output-dir", job.OutputDir,
		"--output-name", job.ID,
		"--output-format", "srt",
		// mlx_whisper defaults to True, which feeds each window's output back in
		// as the next window's prompt -- great for consistency, but once a
		// window mistranscribes (e.g. into silence or hold audio) the model
		// conditions on its own bad output and gets stuck repeating it verbatim
		// for the rest of the file. False trades a bit of cross-window
		// consistency for immunity to that failure loop.
		"--condition-on-previous-text", "False",
	}
	if job.Language != "" && job.Language != "auto" {
		args = append(args, "--language", job.Language)
	}
	args = append(args, inputPath)
	return args
}

func (mlxEngine) ProgressPattern() *regexp.Regexp { return mlxPercentRe }
