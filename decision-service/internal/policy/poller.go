package policy

import (
	"context"
	"time"
)

type RefreshRunner interface {
	RefreshOnce(context.Context) RefreshResult
}

func RunPoller(ctx context.Context, interval time.Duration, refresher RefreshRunner) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			refresher.RefreshOnce(ctx)
		}
	}
}
