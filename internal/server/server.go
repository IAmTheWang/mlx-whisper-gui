package server

import (
	"encoding/json"
	"log"
	"net/http"

	"whisper-gui/internal/jobmanager"
)

type Server struct {
	jobs *jobmanager.Manager
	mux  *http.ServeMux
}

func New(jobs *jobmanager.Manager) *Server {
	s := &Server{jobs: jobs, mux: http.NewServeMux()}
	s.routes()
	return s
}

// Handler returns the top-level http.Handler, wrapped so a panic in any one
// request can never take down the whole server -- it's logged and answered
// with a 500 for that request only.
func (s *Server) Handler() http.Handler {
	return recoverMiddleware(s.mux)
}

func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic handling %s %s: %v", r.Method, r.URL.Path, rec)
				writeJSONError(w, http.StatusInternalServerError, "Internal error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// NewHTTPServer builds the http.Server with WriteTimeout left at its zero
// value (unlimited). A global WriteTimeout would apply to the *entire*
// request lifetime, including long-lived SSE connections that can span the
// tens of minutes a transcription job takes -- setting one here would
// silently sever those connections with no error that points back at tqdm
// or mlx_whisper, making it very hard to diagnose. Any timeout protection
// for ordinary short API calls belongs on those specific handlers, not here.
func NewHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:    addr,
		Handler: handler,
	}
}
