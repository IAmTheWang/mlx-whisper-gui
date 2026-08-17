package server

import (
	"encoding/json"
	"net/http"

	"whisper-gui/internal/config"
	"whisper-gui/internal/whisperbin"
)

type SettingsResponse struct {
	MlxWhisperPath    string `json:"mlxWhisperPath"`
	MlxResolvedVia    string `json:"mlxResolvedVia"`
	FFmpegPath        string `json:"ffmpegPath"`
	FFmpegResolvedVia string `json:"ffmpegResolvedVia"`
}

func currentSettings() SettingsResponse {
	mlx := whisperbin.LocateMlxWhisper()
	ff := whisperbin.LocateFFmpeg()
	return SettingsResponse{
		MlxWhisperPath:    mlx.Path,
		MlxResolvedVia:    string(mlx.ResolvedVia),
		FFmpegPath:        ff.Path,
		FFmpegResolvedVia: string(ff.ResolvedVia),
	}
}

func handleGetSettings(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, currentSettings())
}

type settingsUpdateRequest struct {
	MlxWhisperPath *string `json:"mlxWhisperPath,omitempty"`
	FFmpegPath     *string `json:"ffmpegPath,omitempty"`
}

func handlePutSettings(w http.ResponseWriter, r *http.Request) {
	var req settingsUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	cfg, err := config.Load()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if req.MlxWhisperPath != nil {
		if !whisperbin.IsExecutable(*req.MlxWhisperPath) {
			writeJSONError(w, http.StatusBadRequest, "The specified mlx_whisper path does not exist or is not executable")
			return
		}
		cfg.MlxWhisperPath = *req.MlxWhisperPath
	}
	if req.FFmpegPath != nil {
		if !whisperbin.IsExecutable(*req.FFmpegPath) {
			writeJSONError(w, http.StatusBadRequest, "The specified ffmpeg path does not exist or is not executable")
			return
		}
		cfg.FFmpegPath = *req.FFmpegPath
	}
	if err := config.Save(cfg); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	whisperbin.Refresh()
	writeJSON(w, http.StatusOK, currentSettings())
}
