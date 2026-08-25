package server

import "net/http"

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/browse", handleBrowse)
	s.mux.HandleFunc("GET /api/models", handleModels)
	s.mux.HandleFunc("GET /api/settings", handleGetSettings)
	s.mux.HandleFunc("PUT /api/settings", handlePutSettings)

	s.mux.HandleFunc("POST /api/jobs", s.handleCreateJob)
	s.mux.HandleFunc("GET /api/jobs", s.handleListJobs)
	s.mux.HandleFunc("GET /api/jobs/{id}", s.handleGetJob)
	s.mux.HandleFunc("POST /api/jobs/{id}/cancel", s.handleCancelJob)
	s.mux.HandleFunc("GET /api/jobs/{id}/events", s.handleJobEvents)
	s.mux.HandleFunc("GET /api/jobs/{id}/srt", s.handleDownloadSRT)
	s.mux.HandleFunc("POST /api/jobs/{id}/move-srt", s.handleMoveSRT)

	// Subtree pattern "/api/" is more specific than "/" for anything under
	// /api/, so any API path not matched by one of the routes above lands
	// here (JSON 404) rather than falling through to the SPA fallback below
	// -- a mistyped endpoint must never silently return 200 + HTML.
	s.mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeJSONError(w, http.StatusNotFound, "not found")
	})

	s.mux.Handle("/", spaHandler())
}
