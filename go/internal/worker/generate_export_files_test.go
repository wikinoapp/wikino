package worker

import (
	"testing"

	"github.com/riverqueue/river"

	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	"github.com/wikinoapp/wikino/go/internal/model"
)

func TestGenerateExportFilesArgs_Kind(t *testing.T) {
	t.Parallel()

	args := dispatcher.GenerateExportFilesArgs{}
	if args.Kind() != "generate_export_files" {
		t.Errorf("Kind() = %s, want generate_export_files", args.Kind())
	}
}

func TestGenerateExportFilesArgs_InsertOpts(t *testing.T) {
	t.Parallel()

	args := dispatcher.GenerateExportFilesArgs{}
	opts := args.InsertOpts()

	if opts.Queue != river.QueueDefault {
		t.Errorf("Queue = %s, want %s", opts.Queue, river.QueueDefault)
	}
	if opts.MaxAttempts != 3 {
		t.Errorf("MaxAttempts = %d, want 3", opts.MaxAttempts)
	}
}

// TestGenerateExportFilesWorker_Timeout checks the bound on one attempt, and that the client leaves
// River room to rescue a job only after that bound has passed. River refuses to start when the two
// are the other way around.
//
// [Ja] TestGenerateExportFilesWorker_Timeout は 1 回の試行の上限と、その上限を過ぎてから River が
// ジョブを回収する余地をクライアントが残していることを確認する。両者が逆になっていると River は
// 起動を拒否する。
func TestGenerateExportFilesWorker_Timeout(t *testing.T) {
	t.Parallel()

	worker := NewGenerateExportFilesWorker(nil)

	if got := worker.Timeout(nil); got != model.ExportAttemptTimeout {
		t.Errorf("Timeout() = %v, want %v", got, model.ExportAttemptTimeout)
	}
	if rescueStuckJobsAfter <= model.ExportAttemptTimeout {
		t.Errorf("rescueStuckJobsAfter = %v, want greater than %v", rescueStuckJobsAfter, model.ExportAttemptTimeout)
	}
}
