package worker

import "testing"

// TestDeleteSpaceWorker_Timeoutは、1回の試行の上限が、報告の途絶えたジョブをRiverが取り戻すまでの
// 時間より短いことを確かめる。Riverはこれを満たさない設定では起動しない
func TestDeleteSpaceWorker_Timeout(t *testing.T) {
	t.Parallel()

	worker := NewDeleteSpaceWorker(nil)
	if got := worker.Timeout(nil); got != deleteSpaceAttemptTimeout {
		t.Errorf("Timeout() = %v、期待値 = %v", got, deleteSpaceAttemptTimeout)
	}
	if rescueStuckJobsAfter <= deleteSpaceAttemptTimeout {
		t.Errorf("rescueStuckJobsAfter = %v、期待値 = %vより大きい", rescueStuckJobsAfter, deleteSpaceAttemptTimeout)
	}
}
