package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"example.com/policy-snapshot/decision-gateway/internal/config"
	"example.com/policy-snapshot/decision-gateway/internal/httpapi"
	"example.com/policy-snapshot/decision-gateway/internal/telemetry"
	"example.com/policy-snapshot/decision-gateway/internal/upstream"
)

func main() {
	cfg, err := config.FromEnvironment()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(2)
	}
	client, err := upstream.NewClient(cfg.UpstreamURL, &http.Client{}, cfg.MaxResponseBytes)
	if err != nil {
		slog.Error("create upstream client", "error", err)
		os.Exit(2)
	}
	handler := httpapi.NewServer(client, telemetry.NoopMetrics{}, cfg.UpstreamTimeout, cfg.MaxRequestBytes).Handler()
	server := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	rootContext, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-rootContext.Done()
		shutdownContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownContext); err != nil {
			slog.Error("shutdown server", "error", err)
		}
	}()

	slog.Info("decision gateway listening", "address", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("serve", "error", err)
		os.Exit(1)
	}
}
