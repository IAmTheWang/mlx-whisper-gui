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
