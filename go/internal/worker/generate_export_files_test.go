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
		t.Errorf("Kind() = %s、期待値 = generate_export_files", args.Kind())
	}
}

func TestGenerateExportFilesArgs_InsertOpts(t *testing.T) {
	t.Parallel()

	args := dispatcher.GenerateExportFilesArgs{}
	opts := args.InsertOpts()

	if opts.Queue != river.QueueDefault {
		t.Errorf("Queue = %s、期待値 = %s", opts.Queue, river.QueueDefault)
	}
	if opts.MaxAttempts != 3 {
		t.Errorf("MaxAttempts = %d、期待値 = 3", opts.MaxAttempts)
	}
}

// TestGenerateExportFilesWorker_Timeoutは1回の試行の上限と、その上限を過ぎてからRiverが
// ジョブを回収する余地をクライアントが残していることを確認する。両者が逆になっているとRiverは
// 起動を拒否する。
func TestGenerateExportFilesWorker_Timeout(t *testing.T) {
	t.Parallel()

	worker := NewGenerateExportFilesWorker(nil)

	if got := worker.Timeout(nil); got != model.ExportAttemptTimeout {
		t.Errorf("Timeout() = %v、期待値 = %v", got, model.ExportAttemptTimeout)
	}
	if rescueStuckJobsAfter <= model.ExportAttemptTimeout {
		t.Errorf("rescueStuckJobsAfter = %v、期待値 = %vより大きい", rescueStuckJobsAfter, model.ExportAttemptTimeout)
	}
}
