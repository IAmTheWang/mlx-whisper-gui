package server

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"whisper-gui/internal/jobmanager"
)

type moveSRTRequest struct {
	// DestDir is an absolute path to an existing directory. Empty means
	// "the source video's folder" -- the same destination the automatic
	// sidecar copy already targets.
	DestDir string `json:"destDir"`
}

type moveSRTResponse struct {
	Path string `json:"path"`
}

// handleMoveSRT copies the finished .srt to a folder of the caller's
// choosing. Like the automatic sidecar copy in jobmanager, this overwrites
// an existing file at the destination without asking -- consistent
// behavior, and the destination is always something the user just picked.
func (s *Server) handleMoveSRT(w http.ResponseWriter, r *http.Request) {
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
	if _, err := os.Stat(snap.SRTPath); err != nil {
		writeJSONError(w, http.StatusNotFound, "srt file not found")
		return
	}

	var req moveSRTRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	destDir := req.DestDir
	if destDir == "" {
		destDir = filepath.Dir(snap.VideoPath)
	}
	if !filepath.IsAbs(destDir) {
		writeJSONError(w, http.StatusBadRequest, "destDir must be an absolute path")
		return
	}
	if info, err := os.Stat(destDir); err != nil || !info.IsDir() {
		writeJSONError(w, http.StatusBadRequest, "destDir does not exist or is not a directory")
		return
	}

	dest := filepath.Join(destDir, srtFilenameFor(snap.VideoPath))
	if err := copyFile(snap.SRTPath, dest); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to copy srt: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, moveSRTResponse{Path: dest})
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		os.Remove(dest)
		return err
	}
	return nil
}
