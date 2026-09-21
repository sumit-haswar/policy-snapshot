package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"example.com/policy-snapshot/decision-service/internal/domain"
	"example.com/policy-snapshot/decision-service/internal/policy"
	"example.com/policy-snapshot/decision-service/internal/telemetry"
)

type Server struct {
	store           *policy.Store
	metrics         telemetry.Metrics
	maxRequestBytes int64
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorResponse struct {
	Error errorDetail `json:"error"`
}

func NewServer(store *policy.Store, metrics telemetry.Metrics, maxRequestBytes int64) *Server {
	return &Server{store: store, metrics: metrics, maxRequestBytes: maxRequestBytes}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/live", s.live)
	mux.HandleFunc("GET /health/ready", s.ready)
	mux.HandleFunc("POST /v1/decisions", s.decide)
	return s.observe(mux)
}

func (s *Server) live(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) ready(w http.ResponseWriter, _ *http.Request) {
	if s.store.Current() == nil {
		writeError(w, http.StatusServiceUnavailable, "NOT_READY", "no active policy snapshot")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) decide(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, s.maxRequestBytes))
	decoder.DisallowUnknownFields()
	var request domain.DecisionRequest
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "request body must match the decision contract")
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "request body must contain one JSON value")
		return
	}
	if request.SubjectID == "" || request.Operation == "" || request.Region == "" {
		writeError(w, http.StatusBadRequest, "INVALID_ARGUMENT", "subject_id, operation, and region are required")
		return
	}

	snapshot := s.store.Current()
	response := snapshot.Evaluate(request)
	w.Header().Set("X-Policy-Revision", stringInt(response.PolicyRevision))
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) observe(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		wrapped := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(wrapped, r)
		route := routeTemplate(r.URL.Path)
		statusClass := stringInt(wrapped.status/100) + "xx"
		s.metrics.RequestCompleted(route, statusClass, time.Since(started))
		slog.Info("request completed", "method", r.Method, "route", route, "status", wrapped.status)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func routeTemplate(path string) string {
	switch {
	case path == "/v1/decisions":
		return "/v1/decisions"
	case strings.HasPrefix(path, "/internal/policies/"):
		return "/internal/policies/{operation}"
	default:
		return path
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorResponse{Error: errorDetail{Code: code, Message: message}})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil && !errors.Is(err, http.ErrHandlerTimeout) {
		slog.Error("write response", "error", err)
	}
}

func stringInt(value int) string {
	if value == 0 {
		return "0"
	}
	digits := make([]byte, 0, 12)
	for value > 0 {
		digits = append(digits, byte('0'+value%10))
		value /= 10
	}
	for left, right := 0, len(digits)-1; left < right; left, right = left+1, right-1 {
		digits[left], digits[right] = digits[right], digits[left]
	}
	return string(digits)
}
