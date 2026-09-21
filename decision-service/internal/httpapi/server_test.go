package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"example.com/policy-snapshot/decision-service/internal/policy"
	"example.com/policy-snapshot/decision-service/internal/telemetry"
)

func newTestHandler(metrics telemetry.Metrics) http.Handler {
	store := policy.NewStore(policy.BootstrapSnapshot())
	return NewServer(store, metrics, 4096).Handler()
}

func TestDecisionContractUsesBootstrapSnapshot(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/v1/decisions", strings.NewReader(
		`{"subject_id":"subject-1","operation":"export","region":"standard"}`,
	))
	response := httptest.NewRecorder()
	newTestHandler(telemetry.NoopMetrics{}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["decision"] != "allow" || body["policy_revision"] != float64(1) {
		t.Fatalf("unexpected body: %v", body)
	}
	if got := response.Header().Get("X-Policy-Revision"); got != "1" {
		t.Fatalf("X-Policy-Revision = %q", got)
	}
}

func TestDecisionRejectsUnknownFields(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/v1/decisions", strings.NewReader(
		`{"subject_id":"subject-1","operation":"read","region":"standard","extra":true}`,
	))
	response := httptest.NewRecorder()
	newTestHandler(telemetry.NoopMetrics{}).ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.Code)
	}
}

func TestMetricsUseBoundedRoute(t *testing.T) {
	metrics := &telemetry.MemoryMetrics{}
	request := httptest.NewRequest(http.MethodPost, "/v1/decisions", strings.NewReader(
		`{"subject_id":"private-subject","operation":"read","region":"standard"}`,
	))
	response := httptest.NewRecorder()
	newTestHandler(metrics).ServeHTTP(response, request)

	observations := metrics.Requests()
	if len(observations) != 1 || observations[0].Route != "/v1/decisions" || observations[0].StatusClass != "2xx" {
		t.Fatalf("unexpected observations: %+v", observations)
	}
}
