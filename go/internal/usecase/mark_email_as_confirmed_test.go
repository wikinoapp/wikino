package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func TestMarkEmailAsConfirmedUsecase_Execute_Success(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := repository.NewEmailConfirmationRepository(q)
	updateValidator := validator.NewEmailConfirmationUpdateValidator(repo)
	uc := NewMarkEmailAsConfirmedUsecase(repo, updateValidator)

	// テストデータを作成（有効な確認コード）
	ecID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("mark-success@example.com").
		WithEvent(model.EmailConfirmationEventSignUp).
		WithCode("ABC123").
		WithStartedAt(time.Now()).
		Build()

	ctx := i18n.SetLocale(context.Background(), "ja")

	// メール確認を完了状態に更新
	err := uc.Execute(ctx, MarkEmailAsConfirmedInput{
		EmailConfirmationID: ecID,
		Code:                "ABC123",
	})
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

func TestMarkEmailAsConfirmedUsecase_Execute_ValidationError(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := repository.NewEmailConfirmationRepository(q)
	updateValidator := validator.NewEmailConfirmationUpdateValidator(repo)
	uc := NewMarkEmailAsConfirmedUsecase(repo, updateValidator)

	// テストデータを作成
	ecID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("mark-validation@example.com").
		WithEvent(model.EmailConfirmationEventSignUp).
		WithCode("XYZ789").
		WithStartedAt(time.Now()).
		Build()

	ctx := i18n.SetLocale(context.Background(), "ja")

	// 間違ったコードで実行
	err := uc.Execute(ctx, MarkEmailAsConfirmedInput{
		EmailConfirmationID: ecID,
		Code:                "WRONG1",
	})

	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatal("expected ValidationError, got nil or different error type")
	}
	if !ve.HasFieldError("code") {
		t.Error("expected field error for 'code'")
	}
}

func TestMarkEmailAsConfirmedUsecase_Execute_AlreadySucceeded(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	repo := repository.NewEmailConfirmationRepository(q)
	updateValidator := validator.NewEmailConfirmationUpdateValidator(repo)
	uc := NewMarkEmailAsConfirmedUsecase(repo, updateValidator)

	// 既に確認済みのテストデータを作成
	ecID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("mark-already@example.com").
		WithEvent(model.EmailConfirmationEventSignUp).
		WithCode("DEF456").
		WithStartedAt(time.Now()).
		BuildSucceeded()

	ctx := i18n.SetLocale(context.Background(), "ja")

	// 既に確認済みのコードで実行
	err := uc.Execute(ctx, MarkEmailAsConfirmedInput{
		EmailConfirmationID: ecID,
		Code:                "DEF456",
	})

	ae := model.AsAppError(err)
	if ae == nil {
		t.Fatal("expected AppError, got nil or different error type")
	}
	if ae.Code != model.AppErrCodeConflict {
		t.Errorf("AppError.Code = %d, want %d (AppErrCodeConflict)", ae.Code, model.AppErrCodeConflict)
	}
}
