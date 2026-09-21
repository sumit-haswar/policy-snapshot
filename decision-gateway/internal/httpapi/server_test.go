package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"example.com/policy-snapshot/decision-gateway/internal/telemetry"
	"example.com/policy-snapshot/decision-gateway/internal/upstream"
)

type fakeUpstream struct {
	response upstream.Response
	error    error
	request  []byte
}

func (f *fakeUpstream) Do(_ context.Context, _, _ string, _ http.Header, body []byte) (upstream.Response, error) {
	f.request = append([]byte(nil), body...)
	return f.response, f.error
}

func (f *fakeUpstream) Ready(context.Context) error { return f.error }

func TestGatewayPreservesDecisionResponse(t *testing.T) {
	expected := []byte("{\"decision\":\"allow\",\"future\":null}\n")
	client := &fakeUpstream{response: upstream.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}, "X-Policy-Revision": []string{"2"}},
		Body:       expected,
	}}
	server := NewServer(client, telemetry.NoopMetrics{}, time.Second, 4096).Handler()
	request := httptest.NewRequest(http.MethodPost, "/v1/decisions", strings.NewReader(`{"subject_id":"one"}`))
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)

	if response.Code != http.StatusOK || string(response.Body.Bytes()) != string(expected) {
		t.Fatalf("unexpected response: status=%d body=%q", response.Code, response.Body.String())
	}
	if got := response.Header().Get("X-Policy-Revision"); got != "2" {
		t.Fatalf("X-Policy-Revision = %q", got)
	}
	if string(client.request) != `{"subject_id":"one"}` {
		t.Fatalf("request body changed: %q", client.request)
	}
}
