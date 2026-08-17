package server

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"whisper-gui/internal/jobmanager"
)

func (s *Server) handleDownloadSRT(w http.ResponseWriter, r *http.Request) {
	job, ok := s.jobs.GetJob(r.PathValue("id"))
	if !ok {
		writeJSONError(w, http.StatusNotFound, "Job not found")
		return
	}
	snap := job.Snapshot()
	if snap.State != jobmanager.Done || snap.SRTPath == "" {
		writeJSONError(w, http.StatusConflict, "Job not finished yet")
		return
	}
	// SRTPath always comes from the job record (server-resolved when the
	// job completed) -- never from anything client-supplied.
	if _, err := os.Stat(snap.SRTPath); err != nil {
		writeJSONError(w, http.StatusNotFound, "srt file not found")
		return
	}

	base := filepath.Base(snap.VideoPath)
	filename := strings.TrimSuffix(base, filepath.Ext(base)) + ".srt"
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	http.ServeFile(w, r, snap.SRTPath)
}
