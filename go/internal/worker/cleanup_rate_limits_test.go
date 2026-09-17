package worker

import (
	"testing"

	"github.com/riverqueue/river"

	"github.com/wikinoapp/wikino/go/internal/dispatcher"
)

func TestCleanupRateLimitsArgs_Kind(t *testing.T) {
	t.Parallel()

	args := dispatcher.CleanupRateLimitsArgs{}
	if args.Kind() != "cleanup_rate_limits" {
		t.Errorf("Kind() = %s、期待値 = cleanup_rate_limits", args.Kind())
	}
}

func TestCleanupRateLimitsArgs_InsertOpts(t *testing.T) {
	t.Parallel()

	args := dispatcher.CleanupRateLimitsArgs{}
	opts := args.InsertOpts()

	if opts.Queue != river.QueueDefault {
		t.Errorf("Queue = %s、期待値 = %s", opts.Queue, river.QueueDefault)
	}
	if opts.MaxAttempts != 3 {
		t.Errorf("MaxAttempts = %d、期待値 = 3", opts.MaxAttempts)
	}
}
