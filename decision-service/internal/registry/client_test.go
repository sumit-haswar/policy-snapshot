package registry

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientFetchesManifestAndSendsConditionalHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("If-None-Match"); got != `"previous"` {
			t.Fatalf("If-None-Match = %q", got)
		}
		w.Header().Set("ETag", `"bundle-2"`)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"revision":2,"page_count":1,"rule_count":0,"digest":"sha256:value"}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, server.Client(), 1024)
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.FetchManifest(context.Background(), `"previous"`)
	if err != nil {
		t.Fatal(err)
	}
	if result.Manifest.Revision != 2 || result.ETag != `"bundle-2"` || result.BytesRead == 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestClientReportsNotModified(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotModified)
	}))
	defer server.Close()
	client, _ := NewClient(server.URL, server.Client(), 1024)

	_, err := client.FetchManifest(context.Background(), `"bundle-2"`)
	if !errors.Is(err, ErrNotModified) {
		t.Fatalf("error = %v, want ErrNotModified", err)
	}
}

func TestClientRejectsUnknownFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"revision":2,"page_count":1,"rule_count":0,"digest":"x","extra":true}`))
	}))
	defer server.Close()
	client, _ := NewClient(server.URL, server.Client(), 1024)

	if _, err := client.FetchManifest(context.Background(), ""); err == nil {
		t.Fatal("expected decode error")
	}
}
