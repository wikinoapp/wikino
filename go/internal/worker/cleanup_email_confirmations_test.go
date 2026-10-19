package worker

import (
	"testing"

	"github.com/riverqueue/river"

	"github.com/wikinoapp/wikino/go/internal/dispatcher"
)

func TestCleanupEmailConfirmationsArgs_Kind(t *testing.T) {
	t.Parallel()

	args := dispatcher.CleanupEmailConfirmationsArgs{}
	if args.Kind() != "cleanup_email_confirmations" {
		t.Errorf("Kind() = %s、期待値 = cleanup_email_confirmations", args.Kind())
	}
}

func TestCleanupEmailConfirmationsArgs_InsertOpts(t *testing.T) {
	t.Parallel()

	args := dispatcher.CleanupEmailConfirmationsArgs{}
	opts := args.InsertOpts()

	if opts.Queue != river.QueueDefault {
		t.Errorf("Queue = %s、期待値 = %s", opts.Queue, river.QueueDefault)
	}
	if opts.MaxAttempts != 3 {
		t.Errorf("MaxAttempts = %d、期待値 = 3", opts.MaxAttempts)
	}
}
