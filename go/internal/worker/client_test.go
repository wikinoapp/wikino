package worker

import (
	"reflect"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/dispatcher"
)

func TestPeriodicJobSpecs(t *testing.T) {
	t.Parallel()

	want := []periodicJobSpec{
		{
			args:     dispatcher.CleanupRateLimitsArgs{RetentionHours: 24},
			interval: time.Hour,
		},
		{
			args:     dispatcher.CleanupEmailConfirmationsArgs{RetentionHours: 24},
			interval: time.Hour,
		},
	}

	got := periodicJobSpecs()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("periodicJobSpecs() = %+v、期待値 = %+v", got, want)
	}
}

func TestToPeriodicJobs(t *testing.T) {
	t.Parallel()

	specs := periodicJobSpecs()
	if got := toPeriodicJobs(specs); len(got) != len(specs) {
		t.Errorf("定期ジョブの数 = %d、期待値 = %d", len(got), len(specs))
	}
}
