package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestMarkEmailAsConfirmedUsecase_Execute_Success(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	q := testutil.QueriesWithTx(tx)
	repo := repository.NewEmailConfirmationRepository(q)
	uc := NewMarkEmailAsConfirmedUsecase(repo)

	// テストデータを作成（有効な確認コード）
	ecID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("mark-success@example.com").
		WithEvent(model.EmailConfirmationEventSignUp).
		WithCode("ABC123").
		WithStartedAt(time.Now()).
		Build()

	// メール確認を完了状態に更新
	err := uc.Execute(context.Background(), ecID)
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	// 確認完了状態に更新されたことを確認
	ec, err := repo.FindByID(context.Background(), ecID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if ec.SucceededAt == nil {
		t.Error("確認が完了状態に更新されていません（SucceededAt = nil）")
	}
}
