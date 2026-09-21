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

	"example.com/policy-snapshot/decision-service/internal/config"
	"example.com/policy-snapshot/decision-service/internal/httpapi"
	"example.com/policy-snapshot/decision-service/internal/policy"
	"example.com/policy-snapshot/decision-service/internal/registry"
	"example.com/policy-snapshot/decision-service/internal/telemetry"
)

func main() {
	cfg, err := config.FromEnvironment()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(2)
	}

	store := policy.NewStore(policy.BootstrapSnapshot())
	metrics := telemetry.NoopMetrics{}
	registryClient, err := registry.NewClient(cfg.RegistryURL, &http.Client{}, cfg.MaxBytes)
	if err != nil {
		slog.Error("create registry client", "error", err)
		os.Exit(2)
	}
	refresher := policy.NewRefresher(registryClient, store, metrics, policy.RefreshConfig{
		Timeout: cfg.RefreshTimeout, MaxPages: cfg.MaxPages, MaxBytes: cfg.MaxBytes,
	})

	rootContext, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if cfg.RefreshEnabled {
		go policy.RunPoller(rootContext, cfg.RefreshInterval, refresher)
	}

	handler := httpapi.NewServer(store, metrics, cfg.MaxDecisionRequestBytes).Handler()
	server := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		<-rootContext.Done()
		shutdownContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownContext); err != nil {
			slog.Error("shutdown server", "error", err)
		}
	}()

	slog.Info("decision service listening", "address", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("serve", "error", err)
		os.Exit(1)
	}
}
