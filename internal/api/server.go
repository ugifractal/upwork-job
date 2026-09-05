package api

import (
	"crypto/subtle"
	"encoding/json"
	"log"
	"net/http"

	"go-upwork-job/internal/notifier"
	"go-upwork-job/internal/store"
)

type Server struct {
	store    *store.Store
	notifier *notifier.Notifier
	apiKey   string
}

func New(st *store.Store, nf *notifier.Notifier, apiKey string) *Server {
	return &Server{store: st, notifier: nf, apiKey: apiKey}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.Handle("POST /api/jobs", s.requireAuth(http.HandlerFunc(s.handleCreateJob)))
	mux.Handle("GET /api/jobs", s.requireAuth(http.HandlerFunc(s.handleListJobs)))
	return logRequests(mux)
}

type createJobRequest struct {
	UpworkJobID string `json:"upwork_job_id"`
	Title       string `json:"title"`
	Summary     string `json:"summary"`
	Link        string `json:"link"`
}

func (s *Server) handleCreateJob(w http.ResponseWriter, r *http.Request) {
	var req createJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.UpworkJobID == "" || req.Title == "" {
		writeError(w, http.StatusBadRequest, "upwork_job_id and title are required")
		return
	}

	job, created, err := s.store.CreateJob(store.Job{
		UpworkJobID: req.UpworkJobID,
		Title:       req.Title,
		Summary:     req.Summary,
		Link:        req.Link,
	})
	if err != nil {
		log.Printf("create job: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to save job")
		return
	}
	if !created {
		writeJSON(w, http.StatusOK, map[string]any{"status": "skipped", "job": nil})
		return
	}

	if err := s.notifier.SendNewJob(job); err != nil {
		log.Printf("send email notification: %v", err)
	}
	writeJSON(w, http.StatusCreated, map[string]any{"status": "created", "job": job})
}

func (s *Server) handleListJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := s.store.ListJobs()
	if err != nil {
		log.Printf("list jobs: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list jobs")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"jobs": jobs})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("X-API-Key")
		if subtle.ConstantTimeCompare([]byte(key), []byte(s.apiKey)) != 1 {
			writeError(w, http.StatusUnauthorized, "invalid api key")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"error": msg})
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}