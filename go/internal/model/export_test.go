package model

import (
	"testing"
	"time"
)

func TestExport_InProgress(t *testing.T) {
	t.Parallel()

	now := time.Now()
	fresh := now.Add(-ExportHeartbeatInterval)
	stale := now.Add(-ExportHeartbeatStaleAfter - time.Second)
	waiting := now.Add(-ExportQueuedStaleAfter + time.Second)
	unclaimed := now.Add(-ExportQueuedStaleAfter - time.Second)

	tests := []struct {
		name            string
		status          ExportStatus
		statusChangedAt time.Time
		heartbeatAt     *time.Time
		want            bool
	}{
		{name: "ワーカーを待っているqueuedは実行中とみなす", status: ExportStatusQueued, statusChangedAt: waiting, want: true},
		{name: "拾われないまま閾値を過ぎたqueuedは実行中とみなさない", status: ExportStatusQueued, statusChangedAt: unclaimed, want: false},
		{name: "heartbeatが新しいstartedは実行中とみなす", status: ExportStatusStarted, statusChangedAt: unclaimed, heartbeatAt: &fresh, want: true},
		{name: "heartbeatが古いstartedは実行中とみなさない", status: ExportStatusStarted, statusChangedAt: fresh, heartbeatAt: &stale, want: false},
		{name: "heartbeatを持たないstartedは実行中とみなさない", status: ExportStatusStarted, statusChangedAt: fresh, want: false},
		{name: "succeededは実行中とみなさない", status: ExportStatusSucceeded, statusChangedAt: fresh, heartbeatAt: &fresh, want: false},
		{name: "failedは実行中とみなさない", status: ExportStatusFailed, statusChangedAt: fresh, heartbeatAt: &fresh, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			export := &Export{Status: tt.status, StatusChangedAt: tt.statusChangedAt, HeartbeatAt: tt.heartbeatAt}
			if got := export.InProgress(now); got != tt.want {
				t.Errorf("InProgress() = %v, want %v", got, tt.want)
			}
		})
	}
}
