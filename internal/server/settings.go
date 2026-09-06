package server

import (
	"encoding/json"
	"net/http"
	"os"

	"whisper-gui/internal/config"
	"whisper-gui/internal/jobmanager"
	"whisper-gui/internal/whisperbin"
)

type SettingsResponse struct {
	MlxWhisperPath        string `json:"mlxWhisperPath"`
	MlxResolvedVia        string `json:"mlxResolvedVia"`
	FFmpegPath            string `json:"ffmpegPath"`
	FFmpegResolvedVia     string `json:"ffmpegResolvedVia"`
	WhisperCliPath        string `json:"whisperCliPath"`
	WhisperCliResolvedVia string `json:"whisperCliResolvedVia"`
	WhisperCppModelDir    string `json:"whisperCppModelDir"`
	DefaultEngine         string `json:"defaultEngine"`
}

func currentSettings() SettingsResponse {
	mlx := whisperbin.LocateMlxWhisper()
	ff := whisperbin.LocateFFmpeg()
	wc := whisperbin.LocateWhisperCli()
	cfg, _ := config.Load()
	return SettingsResponse{
		MlxWhisperPath:        mlx.Path,
		MlxResolvedVia:        string(mlx.ResolvedVia),
		FFmpegPath:            ff.Path,
		FFmpegResolvedVia:     string(ff.ResolvedVia),
		WhisperCliPath:        wc.Path,
		WhisperCliResolvedVia: string(wc.ResolvedVia),
		WhisperCppModelDir:    cfg.WhisperCppModelDir,
		DefaultEngine:         cfg.DefaultEngine,
	}
}

func handleGetSettings(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, currentSettings())
}

type settingsUpdateRequest struct {
	MlxWhisperPath     *string `json:"mlxWhisperPath,omitempty"`
	FFmpegPath         *string `json:"ffmpegPath,omitempty"`
	WhisperCliPath     *string `json:"whisperCliPath,omitempty"`
	WhisperCppModelDir *string `json:"whisperCppModelDir,omitempty"`
	DefaultEngine      *string `json:"defaultEngine,omitempty"`
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
	// An empty string clears the override back to auto-detection (config -> PATH
	// -> fallback) rather than being validated as a path -- the Settings UI always
	// submits every field, so a field left blank must mean "unset this", not
	// "reject the whole save because '' isn't executable".
	if req.MlxWhisperPath != nil {
		if *req.MlxWhisperPath != "" && !whisperbin.IsExecutable(*req.MlxWhisperPath) {
			writeJSONError(w, http.StatusBadRequest, "The specified mlx_whisper path does not exist or is not executable")
			return
		}
		cfg.MlxWhisperPath = *req.MlxWhisperPath
	}
	if req.FFmpegPath != nil {
		if *req.FFmpegPath != "" && !whisperbin.IsExecutable(*req.FFmpegPath) {
			writeJSONError(w, http.StatusBadRequest, "The specified ffmpeg path does not exist or is not executable")
			return
		}
		cfg.FFmpegPath = *req.FFmpegPath
	}
	if req.WhisperCliPath != nil {
		if *req.WhisperCliPath != "" && !whisperbin.IsExecutable(*req.WhisperCliPath) {
			writeJSONError(w, http.StatusBadRequest, "The specified whisper-cli path does not exist or is not executable")
			return
		}
		cfg.WhisperCliPath = *req.WhisperCliPath
	}
	if req.WhisperCppModelDir != nil {
		if *req.WhisperCppModelDir != "" {
			info, statErr := os.Stat(*req.WhisperCppModelDir)
			if statErr != nil || !info.IsDir() {
				writeJSONError(w, http.StatusBadRequest, "The specified whisper.cpp model directory does not exist or is not a directory")
				return
			}
		}
		cfg.WhisperCppModelDir = *req.WhisperCppModelDir
	}
	if req.DefaultEngine != nil {
		if *req.DefaultEngine != "" && *req.DefaultEngine != jobmanager.EngineMlx && *req.DefaultEngine != jobmanager.EngineWhisperCpp {
			writeJSONError(w, http.StatusBadRequest, "Unknown engine: "+*req.DefaultEngine)
			return
		}
		cfg.DefaultEngine = *req.DefaultEngine
	}
	if err := config.Save(cfg); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	whisperbin.Refresh()
	writeJSON(w, http.StatusOK, currentSettings())
}
