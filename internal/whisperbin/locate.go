package whisperbin

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"whisper-gui/internal/config"
)

type ResolvedVia string

const (
	ViaConfig   ResolvedVia = "config"
	ViaPath     ResolvedVia = "path"
	ViaFallback ResolvedVia = "fallback"
	ViaNone     ResolvedVia = "none"
)

type Resolution struct {
	Path        string      `json:"path"`
	ResolvedVia ResolvedVia `json:"resolvedVia"`
}

var (
	mu               sync.Mutex
	cachedMlx        *Resolution
	cachedFFmpeg     *Resolution
	cachedWhisperCli *Resolution
)

// IsExecutable reports whether p exists, is a regular file, and has at
// least one executable bit set. Exported so the settings API can validate a
// user-supplied override path with the exact same check used internally.
func IsExecutable(p string) bool {
	if p == "" {
		return false
	}
	info, err := os.Stat(p)
	if err != nil || info.IsDir() {
		return false
	}
	return info.Mode()&0o111 != 0
}

// LocateMlxWhisper resolves the mlx_whisper binary: config override -> PATH -> known
// pip user-install fallback. Never errors; callers check ResolvedVia == ViaNone.
func LocateMlxWhisper() Resolution {
	mu.Lock()
	defer mu.Unlock()
	if cachedMlx != nil {
		return *cachedMlx
	}
	r := resolveMlxWhisper()
	cachedMlx = &r
	return r
}

func resolveMlxWhisper() Resolution {
	cfg, _ := config.Load()
	if IsExecutable(cfg.MlxWhisperPath) {
		return Resolution{Path: cfg.MlxWhisperPath, ResolvedVia: ViaConfig}
	}
	if p, err := exec.LookPath("mlx_whisper"); err == nil {
		return Resolution{Path: p, ResolvedVia: ViaPath}
	}
	if home, err := os.UserHomeDir(); err == nil {
		fallback := filepath.Join(home, "Library", "Python", "3.9", "bin", "mlx_whisper")
		if IsExecutable(fallback) {
			return Resolution{Path: fallback, ResolvedVia: ViaFallback}
		}
	}
	return Resolution{ResolvedVia: ViaNone}
}

// LocateFFmpeg resolves ffmpeg via config override or PATH only -- it's normally
// installed via Homebrew onto a standard PATH location, so no pip-style fallback path
// is needed the way mlx_whisper has one.
func LocateFFmpeg() Resolution {
	mu.Lock()
	defer mu.Unlock()
	if cachedFFmpeg != nil {
		return *cachedFFmpeg
	}
	r := resolveFFmpeg()
	cachedFFmpeg = &r
	return r
}

func resolveFFmpeg() Resolution {
	cfg, _ := config.Load()
	if IsExecutable(cfg.FFmpegPath) {
		return Resolution{Path: cfg.FFmpegPath, ResolvedVia: ViaConfig}
	}
	if p, err := exec.LookPath("ffmpeg"); err == nil {
		return Resolution{Path: p, ResolvedVia: ViaPath}
	}
	return Resolution{ResolvedVia: ViaNone}
}

// LocateWhisperCli resolves whisper.cpp's whisper-cli binary: config override -> PATH
// only, same as LocateFFmpeg -- `brew install whisper-cpp` puts it on a standard
// Homebrew PATH, so no pip-style fallback path is needed the way mlx_whisper has one.
func LocateWhisperCli() Resolution {
	mu.Lock()
	defer mu.Unlock()
	if cachedWhisperCli != nil {
		return *cachedWhisperCli
	}
	r := resolveWhisperCli()
	cachedWhisperCli = &r
	return r
}

func resolveWhisperCli() Resolution {
	cfg, _ := config.Load()
	if IsExecutable(cfg.WhisperCliPath) {
		return Resolution{Path: cfg.WhisperCliPath, ResolvedVia: ViaConfig}
	}
	if p, err := exec.LookPath("whisper-cli"); err == nil {
		return Resolution{Path: p, ResolvedVia: ViaPath}
	}
	return Resolution{ResolvedVia: ViaNone}
}

// Refresh clears the cached resolutions so the next Locate* call re-checks the
// filesystem/PATH -- used after the user saves a new path in Settings.
func Refresh() {
	mu.Lock()
	defer mu.Unlock()
	cachedMlx = nil
	cachedFFmpeg = nil
	cachedWhisperCli = nil
}
