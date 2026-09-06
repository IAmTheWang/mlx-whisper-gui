package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	MlxWhisperPath     string `json:"mlxWhisperPath,omitempty"`
	FFmpegPath         string `json:"ffmpegPath,omitempty"`
	WhisperCliPath     string `json:"whisperCliPath,omitempty"`
	WhisperCppModelDir string `json:"whisperCppModelDir,omitempty"`
	DefaultEngine      string `json:"defaultEngine,omitempty"` // "" behaves as "mlx"
	// WhisperVadModelPath, when set, enables whisper.cpp's Voice Activity
	// Detection (a ggml-format Silero VAD model file, e.g. ggml-silero-v5.1.2.bin)
	// so silent stretches are skipped before transcription instead of being
	// guessed at -- empty disables VAD (whisper-cli runs exactly as before).
	WhisperVadModelPath string `json:"whisperVadModelPath,omitempty"`
}

func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "Application Support", "whisper-gui"), nil
}

func path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func Load() (Config, error) {
	p, err := path()
	if err != nil {
		return Config{}, err
	}
	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func Save(cfg Config) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	p, err := path()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}
