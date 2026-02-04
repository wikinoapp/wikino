package worker

import (
	"testing"

	"github.com/riverqueue/river"
)

func TestCleanupRateLimitsArgs_Kind(t *testing.T) {
	t.Parallel()

	args := CleanupRateLimitsArgs{}
	if args.Kind() != "cleanup_rate_limits" {
		t.Errorf("Kind() = %s, want cleanup_rate_limits", args.Kind())
	}
}

func TestCleanupRateLimitsArgs_InsertOpts(t *testing.T) {
	t.Parallel()

	args := CleanupRateLimitsArgs{}
	opts := args.InsertOpts()

	if opts.Queue != river.QueueDefault {
		t.Errorf("Queue = %s, want %s", opts.Queue, river.QueueDefault)
	}
	if opts.MaxAttempts != 3 {
		t.Errorf("MaxAttempts = %d, want 3", opts.MaxAttempts)
	}
}
