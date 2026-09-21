package policy

import (
	"context"
	"time"

	"example.com/policy-snapshot/decision-service/internal/registry"
	"example.com/policy-snapshot/decision-service/internal/telemetry"
)

const OutcomeDisabled = "disabled"

type RegistryClient interface {
	FetchManifest(ctx context.Context, ifNoneMatch string) (registry.ManifestResult, error)
	FetchPage(ctx context.Context, revision, pageIndex int) (registry.PageResult, error)
}

type RefreshConfig struct {
	Timeout  time.Duration
	MaxPages int
	MaxBytes int64
}

type RefreshResult struct {
	Outcome        string `json:"outcome"`
	ActiveRevision int    `json:"active_revision"`
}

type Refresher struct {
	client  RegistryClient
	store   *Store
	metrics telemetry.Metrics
	config  RefreshConfig
}

func NewRefresher(client RegistryClient, store *Store, metrics telemetry.Metrics, config RefreshConfig) *Refresher {
	return &Refresher{
		client:  client,
		store:   store,
		metrics: metrics,
		config:  config,
	}
}

func (r *Refresher) RefreshOnce(_ context.Context) RefreshResult {
	started := time.Now()
	result := RefreshResult{
		Outcome:        OutcomeDisabled,
		ActiveRevision: r.store.Current().Revision(),
	}
	r.metrics.PolicyRefreshCompleted(result.Outcome, time.Since(started))
	return result
}
