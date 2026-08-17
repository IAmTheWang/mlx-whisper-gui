package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"whisper-gui/internal/jobmanager"
)

type createJobRequest struct {
	VideoPath string `json:"videoPath"`
	Model     string `json:"model"`
	Language  string `json:"language"`
}

func (s *Server) handleCreateJob(w http.ResponseWriter, r *http.Request) {
	var req createJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if !isKnownModel(req.Model) {
		writeJSONError(w, http.StatusBadRequest, "Unknown model: "+req.Model)
		return
	}
	if req.Language != "" && !isKnownLanguage(req.Language) {
		writeJSONError(w, http.StatusBadRequest, "Unknown language code: "+req.Language)
		return
	}

	job, err := s.jobs.CreateJob(jobmanager.CreateRequest{
		VideoPath: req.VideoPath,
		Model:     req.Model,
		Language:  req.Language,
	})
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, job.Snapshot())
}

func (s *Server) handleListJobs(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.jobs.ListJobs())
}

func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request) {
	job, ok := s.jobs.GetJob(r.PathValue("id"))
	if !ok {
		writeJSONError(w, http.StatusNotFound, "Job not found")
		return
	}
	writeJSON(w, http.StatusOK, job.Snapshot())
}

func (s *Server) handleCancelJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	switch err := s.jobs.CancelJob(id); {
	case err == nil:
		job, _ := s.jobs.GetJob(id)
		writeJSON(w, http.StatusOK, job.Snapshot())
	case errors.Is(err, jobmanager.ErrNotFound):
		writeJSONError(w, http.StatusNotFound, "Job not found")
	case errors.Is(err, jobmanager.ErrAlreadyTerminal):
		writeJSONError(w, http.StatusConflict, "Job already finished, cannot cancel")
	default:
		writeJSONError(w, http.StatusInternalServerError, err.Error())
	}
}
