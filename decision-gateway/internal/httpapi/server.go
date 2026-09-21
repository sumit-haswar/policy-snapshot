package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"example.com/policy-snapshot/decision-gateway/internal/telemetry"
	"example.com/policy-snapshot/decision-gateway/internal/upstream"
)

type Upstream interface {
	Do(ctx context.Context, method, path string, header http.Header, body []byte) (upstream.Response, error)
	Ready(ctx context.Context) error
}

type Server struct {
	upstream        Upstream
	metrics         telemetry.Metrics
	upstreamTimeout time.Duration
	maxRequestBytes int64
}

func NewServer(client Upstream, metrics telemetry.Metrics, upstreamTimeout time.Duration, maxRequestBytes int64) *Server {
	return &Server{upstream: client, metrics: metrics, upstreamTimeout: upstreamTimeout, maxRequestBytes: maxRequestBytes}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/live", s.live)
	mux.HandleFunc("GET /health/ready", s.ready)
	mux.HandleFunc("POST /v1/decisions", s.proxyDecision)
	return s.observe(mux)
}

func (s *Server) live(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), s.upstreamTimeout)
	defer cancel()
	if err := s.upstream.Ready(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": map[string]string{
			"code": "NOT_READY", "message": "decision service is not ready",
		}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) proxyDecision(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, s.maxRequestBytes))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": map[string]string{
			"code": "INVALID_ARGUMENT", "message": "request body is too large",
		}})
		return
	}
	requestID := r.Header.Get("X-Request-ID")
	if requestID == "" {
		requestID = newRequestID()
	}
	headers := r.Header.Clone()
	headers.Set("X-Request-ID", requestID)

	ctx, cancel := context.WithTimeout(r.Context(), s.upstreamTimeout)
	defer cancel()
	response, err := s.upstream.Do(ctx, r.Method, r.URL.RequestURI(), headers, body)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": map[string]string{
			"code": "UPSTREAM_UNAVAILABLE", "message": "decision service is unavailable",
		}})
		return
	}
	copyResponseHeaders(w.Header(), response.Header)
	w.Header().Set("X-Request-ID", requestID)
	w.WriteHeader(response.StatusCode)
	_, _ = w.Write(response.Body)
}

func (s *Server) observe(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		wrapped := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(wrapped, r)
		statusClass := string([]byte{byte('0' + wrapped.status/100), 'x', 'x'})
		s.metrics.RequestCompleted(routeTemplate(r.URL.Path), statusClass, time.Since(started))
		slog.Info("request completed", "method", r.Method, "route", routeTemplate(r.URL.Path), "status", wrapped.status)
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
	if path == "/v1/decisions" {
		return "/v1/decisions"
	}
	return path
}

func copyResponseHeaders(destination, source http.Header) {
	for name, values := range source {
		if isHopByHop(name) || name == "Content-Length" {
			continue
		}
		for _, value := range values {
			destination.Add(name, value)
		}
	}
}

func isHopByHop(name string) bool {
	switch http.CanonicalHeaderKey(name) {
	case "Connection", "Keep-Alive", "Proxy-Authenticate", "Proxy-Authorization", "Te", "Trailer", "Transfer-Encoding", "Upgrade":
		return true
	default:
		return false
	}
}

func newRequestID() string {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "request-unavailable"
	}
	return hex.EncodeToString(buffer)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil && !errors.Is(err, http.ErrHandlerTimeout) {
		slog.Error("write response", "error", err)
	}
}
