package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type ModelInfo struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Cached bool   `json:"cached"`
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
	{"mlx-community/whisper-large-v3", "Large v3 (most accurate, slow)"},
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

func isKnownModel(id string) bool {
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

func handleModels(w http.ResponseWriter, r *http.Request) {
	models := make([]ModelInfo, 0, len(curatedModels))
	for _, m := range curatedModels {
		models = append(models, ModelInfo{ID: m.ID, Label: m.Label, Cached: isModelCached(m.ID)})
	}
	writeJSON(w, http.StatusOK, ModelsResponse{Models: models, Languages: curatedLanguages})
}
