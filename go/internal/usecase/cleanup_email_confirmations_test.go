package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestCleanupEmailConfirmationsUsecase_Execute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		retentionHours int
		codePrefix     string
	}{
		{name: "保持期間を過ぎた行だけを削除する", retentionHours: 24, codePrefix: "CEA"},
		{name: "保持期間が0なら24時間とみなす", retentionHours: 0, codePrefix: "CEB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			repo := repository.NewEmailConfirmationRepository(testutil.QueriesWithTx(tx))
			uc := NewCleanupEmailConfirmationsUsecase(repo)

			now := time.Now()
			oldUnsucceededID := testutil.NewEmailConfirmationBuilder(t, tx).
				WithCode(tt.codePrefix + "001").
				WithStartedAt(now.Add(-25 * time.Hour)).
				Build()
			oldSucceededID := testutil.NewEmailConfirmationBuilder(t, tx).
				WithCode(tt.codePrefix + "002").
				WithStartedAt(now.Add(-25 * time.Hour)).
				WithSucceededAt(now.Add(-25*time.Hour + time.Minute)).
				Build()
			recentID := testutil.NewEmailConfirmationBuilder(t, tx).
				WithCode(tt.codePrefix + "003").
				WithStartedAt(now.Add(-23 * time.Hour)).
				Build()

			err := uc.Execute(context.Background(), CleanupEmailConfirmationsInput{
				RetentionHours: tt.retentionHours,
			})
			if err != nil {
				t.Fatalf("Execute()のエラー = %v", err)
			}

			for id, wantDeleted := range map[string]bool{
				oldUnsucceededID: true,
				oldSucceededID:   true,
				recentID:         false,
			} {
				ec, err := repo.FindByID(context.Background(), id)
				if err != nil {
					t.Fatalf("FindByID()のエラー = %v", err)
				}
				if deleted := ec == nil; deleted != wantDeleted {
					t.Errorf("ID %sが削除されたか = %v、期待値 = %v", id, deleted, wantDeleted)
				}
			}
		})
	}
}
