package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"whisper-gui/internal/config"
	"whisper-gui/internal/jobmanager"
)

type ModelInfo struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Cached bool   `json:"cached"`
	Engine string `json:"engine"`
}

type LanguageInfo struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

type ModelsResponse struct {
	Models    []ModelInfo    `json:"models"`
	Languages []LanguageInfo `json:"languages"`
}

var curatedModels = []struct{ ID, Label string }{
	{"mlx-community/whisper-large-v3-turbo", "Large v3 Turbo (fast)"},
	{"mlx-community/whisper-large-v3-mlx", "Large v3 (most accurate, slow)"},
	{"mlx-community/whisper-medium", "Medium (balanced)"},
}

var curatedLanguages = []LanguageInfo{
	{"auto", "Auto-detect"},
	{"ja", "Japanese"},
	{"en", "English"},
	{"zh", "Chinese"},
	{"ko", "Korean"},
	{"fr", "French"},
	{"de", "German"},
	{"es", "Spanish"},
	{"it", "Italian"},
	{"pt", "Portuguese"},
	{"ru", "Russian"},
}

func isKnownEngine(engine string) bool {
	return engine == jobmanager.EngineMlx || engine == jobmanager.EngineWhisperCpp
}

// isKnownModel validates a model id for the given engine: mlx checks against
// the static curatedModels list, whispercpp re-scans the configured model
// directory so only a file actually present there can be passed to -m.
func isKnownModel(id, engine string) bool {
	if engine == jobmanager.EngineWhisperCpp {
		for _, m := range scanWhisperCppModels() {
			if m.ID == id {
				return true
			}
		}
		return false
	}
	for _, m := range curatedModels {
		if m.ID == id {
			return true
		}
	}
	return false
}

func isKnownLanguage(code string) bool {
	for _, l := range curatedLanguages {
		if l.Code == code {
			return true
		}
	}
	return false
}

// isModelCached checks the Hugging Face Hub cache layout directly
// (~/.cache/huggingface/hub/models--<org>--<name>/snapshots/*/<weights>)
// rather than shelling out to anything -- cheap and exactly what
// mlx_whisper itself reads from.
func isModelCached(id string) bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	cacheName := "models--" + strings.ReplaceAll(id, "/", "--")
	snapshotsDir := filepath.Join(home, ".cache", "huggingface", "hub", cacheName, "snapshots")
	snapshots, err := os.ReadDir(snapshotsDir)
	if err != nil {
		return false
	}
	for _, snap := range snapshots {
		if !snap.IsDir() {
			continue
		}
		files, err := os.ReadDir(filepath.Join(snapshotsDir, snap.Name()))
		if err == nil && len(files) > 0 {
			return true
		}
	}
	return false
}

// scanWhisperCppModels lists the *.bin files directly inside the configured
// whisper.cpp model directory (top-level only, not recursive). There is no
// download step for this engine -- the user places ggml-*.bin files there
// themselves -- so presence on disk is the only "cached" signal needed.
//
// A VAD model (config.WhisperVadModelPath) is a *.bin file too, and nothing
// stops the user from downloading it into the same directory as their
// transcription models -- so it's explicitly excluded here by path. It's a
// VAD model, not a transcription model, and would fail or produce garbage if
// passed to whisper-cli's -m.
func scanWhisperCppModels() []ModelInfo {
	models := []ModelInfo{} // never nil -- must serialize as JSON [] for the frontend, not null
	cfg, err := config.Load()
	if err != nil || cfg.WhisperCppModelDir == "" {
		return models
	}
	entries, err := os.ReadDir(cfg.WhisperCppModelDir)
	if err != nil {
		return models
	}
	for _, e := range entries {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".bin") {
			continue
		}
		path := filepath.Join(cfg.WhisperCppModelDir, e.Name())
		if cfg.WhisperVadModelPath != "" && path == cfg.WhisperVadModelPath {
			continue
		}
		label := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		models = append(models, ModelInfo{ID: path, Label: label, Cached: true, Engine: jobmanager.EngineWhisperCpp})
	}
	return models
}

func handleModels(w http.ResponseWriter, r *http.Request) {
	engine := r.URL.Query().Get("engine")
	if engine == "" {
		engine = jobmanager.EngineMlx
	}
	if !isKnownEngine(engine) {
		writeJSONError(w, http.StatusBadRequest, "Unknown engine: "+engine)
		return
	}

	var models []ModelInfo
	if engine == jobmanager.EngineWhisperCpp {
		models = scanWhisperCppModels()
	} else {
		models = make([]ModelInfo, 0, len(curatedModels))
		for _, m := range curatedModels {
			models = append(models, ModelInfo{ID: m.ID, Label: m.Label, Cached: isModelCached(m.ID), Engine: jobmanager.EngineMlx})
		}
	}
	writeJSON(w, http.StatusOK, ModelsResponse{Models: models, Languages: curatedLanguages})
}
