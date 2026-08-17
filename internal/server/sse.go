package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"whisper-gui/internal/jobmanager"
)

func (s *Server) handleJobEvents(w http.ResponseWriter, r *http.Request) {
	job, ok := s.jobs.GetJob(r.PathValue("id"))
	if !ok {
		writeJSONError(w, http.StatusNotFound, "Job not found")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSONError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	// This connection can live for as long as the transcription job runs
	// (tens of minutes for a long video) -- explicitly disable its write
	// deadline. Belt-and-suspenders alongside leaving the server's global
	// WriteTimeout at 0 in server.go.
	_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	writeSSE := func(name string, data interface{}) bool {
		payload, err := json.Marshal(data)
		if err != nil {
			return true
		}
		if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", name, payload); err != nil {
			return false
		}
		flusher.Flush()
		return true
	}

	ch, replay, unsubscribe := job.Subscribe()
	defer unsubscribe()

	// A fresh Job loaded from jobs.json after a server restart has an empty
	// in-memory replay buffer even though it's a terminal job with real
	// history -- fall back to the on-disk output.log so a historical job's
	// full log is still viewable, then close (there's nothing live to tail).
	if len(replay) == 0 {
		if snap := job.Snapshot(); snap.State.Terminal() {
			if data, err := job.ReadLogFile(); err == nil {
				lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
				for i, line := range lines {
					if line == "" {
						continue
					}
					stream, text := parseLogLine(line)
					if !writeSSE("log", jobmanager.LogEvent{Seq: int64(i + 1), Stream: stream, Text: text}) {
						return
					}
				}
			}
			writeSSE("state", jobmanager.StateEvent{State: snap.State, Error: snap.Error, SRTAvailable: snap.SRTPath != ""})
			return
		}
	}

	for _, e := range replay {
		if !writeSSE("log", e) {
			return
		}
	}

	if snap := job.Snapshot(); snap.State.Terminal() {
		writeSSE("state", jobmanager.StateEvent{State: snap.State, Error: snap.Error, SRTAvailable: snap.SRTPath != ""})
		return
	}

	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case evt := <-ch:
			if !writeSSE(evt.Name, evt.Data) {
				return
			}
			if se, isState := evt.Data.(jobmanager.StateEvent); isState && se.State.Terminal() {
				return
			}
		case <-heartbeat.C:
			if _, err := fmt.Fprint(w, ": keepalive\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

// parseLogLine reverses the "[stream] text" format runJob writes to
// output.log, so historical replay can report the original stream instead
// of a generic placeholder.
func parseLogLine(raw string) (stream, text string) {
	if strings.HasPrefix(raw, "[") {
		if idx := strings.Index(raw, "] "); idx > 0 {
			return raw[1:idx], raw[idx+2:]
		}
	}
	return "unknown", raw
}
