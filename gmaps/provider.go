package gmaps

import (
	"context"
	"github.com/gosom/scrapemate"
)

// Provider defines the interface for job queue operations
type Provider interface {
	// Push adds a new job to the queue
	Push(ctx context.Context, job scrapemate.IJob) error
} 