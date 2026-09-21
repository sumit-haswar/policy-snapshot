package upstream

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientPreservesResponseBytesAndHeaders(t *testing.T) {
	expected := []byte("{\"future_field\":null,\"decision\":\"allow\"}\n")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Request-ID"); got != "request-1" {
			t.Fatalf("X-Request-ID = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Policy-Revision", "4")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write(expected)
	}))
	defer server.Close()
	client, err := NewClient(server.URL, server.Client(), 1024)
	if err != nil {
		t.Fatal(err)
	}

	response, err := client.Do(context.Background(), http.MethodPost, "/v1/decisions", http.Header{"X-Request-Id": []string{"request-1"}}, []byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusAccepted || string(response.Body) != string(expected) {
		t.Fatalf("unexpected response: %+v", response)
	}
	if got := response.Header.Get("X-Policy-Revision"); got != "4" {
		t.Fatalf("X-Policy-Revision = %q", got)
	}
}
