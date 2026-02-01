package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestVerifyEmailConfirmationUsecase_Execute_Success(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	q := testutil.QueriesWithTx(tx)
	repo := repository.NewEmailConfirmationRepository(q)
	uc := NewVerifyEmailConfirmationUsecase(repo)

	// テストデータを作成（有効な確認コード）
	ecID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("verify-success@example.com").
		WithEvent(model.EmailConfirmationEventSignUp).
		WithCode("ABC123").
		WithStartedAt(time.Now()).
		Build()

	// 確認コードを検証
	err := uc.Execute(context.Background(), VerifyEmailConfirmationInput{
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

func TestVerifyEmailConfirmationUsecase_Execute_CaseInsensitive(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	q := testutil.QueriesWithTx(tx)
	repo := repository.NewEmailConfirmationRepository(q)
	uc := NewVerifyEmailConfirmationUsecase(repo)

	// テストデータを作成（大文字のコード）
	ecID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("case-insensitive@example.com").
		WithEvent(model.EmailConfirmationEventSignUp).
		WithCode("XYZ789").
		WithStartedAt(time.Now()).
		Build()

	// 小文字で入力しても検証が成功することを確認
	err := uc.Execute(context.Background(), VerifyEmailConfirmationInput{
		EmailConfirmationID: ecID,
		Code:                "xyz789",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil（小文字でも成功すべき）", err)
	}
}

func TestVerifyEmailConfirmationUsecase_Execute_NotFound(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	q := testutil.QueriesWithTx(tx)
	repo := repository.NewEmailConfirmationRepository(q)
	uc := NewVerifyEmailConfirmationUsecase(repo)

	// 存在しないIDで検証
	err := uc.Execute(context.Background(), VerifyEmailConfirmationInput{
		EmailConfirmationID: "00000000-0000-0000-0000-000000000000",
		Code:                "ABC123",
	})
	if !errors.Is(err, ErrEmailConfirmationNotFound) {
		t.Errorf("Execute() error = %v, want %v", err, ErrEmailConfirmationNotFound)
	}
}

func TestVerifyEmailConfirmationUsecase_Execute_AlreadySucceeded(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	q := testutil.QueriesWithTx(tx)
	repo := repository.NewEmailConfirmationRepository(q)
	uc := NewVerifyEmailConfirmationUsecase(repo)

	// 既に確認済みのテストデータを作成
	ecID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("already-succeeded@example.com").
		WithEvent(model.EmailConfirmationEventSignUp).
		WithCode("DEF456").
		WithStartedAt(time.Now()).
		BuildSucceeded()

	// 検証しようとするとエラーになる
	err := uc.Execute(context.Background(), VerifyEmailConfirmationInput{
		EmailConfirmationID: ecID,
		Code:                "DEF456",
	})
	if !errors.Is(err, ErrEmailConfirmationAlreadySucceeded) {
		t.Errorf("Execute() error = %v, want %v", err, ErrEmailConfirmationAlreadySucceeded)
	}
}

func TestVerifyEmailConfirmationUsecase_Execute_Expired(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	q := testutil.QueriesWithTx(tx)
	repo := repository.NewEmailConfirmationRepository(q)
	uc := NewVerifyEmailConfirmationUsecase(repo)

	// 16分前のテストデータを作成（15分で有効期限切れ）
	ecID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("expired@example.com").
		WithEvent(model.EmailConfirmationEventSignUp).
		WithCode("GHI789").
		WithStartedAt(time.Now().Add(-16 * time.Minute)).
		Build()

	// 検証しようとするとエラーになる
	err := uc.Execute(context.Background(), VerifyEmailConfirmationInput{
		EmailConfirmationID: ecID,
		Code:                "GHI789",
	})
	if !errors.Is(err, ErrEmailConfirmationExpired) {
		t.Errorf("Execute() error = %v, want %v", err, ErrEmailConfirmationExpired)
	}
}

func TestVerifyEmailConfirmationUsecase_Execute_CodeMismatch(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	q := testutil.QueriesWithTx(tx)
	repo := repository.NewEmailConfirmationRepository(q)
	uc := NewVerifyEmailConfirmationUsecase(repo)

	// テストデータを作成
	ecID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("code-mismatch@example.com").
		WithEvent(model.EmailConfirmationEventSignUp).
		WithCode("JKL012").
		WithStartedAt(time.Now()).
		Build()

	// 間違ったコードで検証
	err := uc.Execute(context.Background(), VerifyEmailConfirmationInput{
		EmailConfirmationID: ecID,
		Code:                "WRONG1",
	})
	if !errors.Is(err, ErrEmailConfirmationCodeMismatch) {
		t.Errorf("Execute() error = %v, want %v", err, ErrEmailConfirmationCodeMismatch)
	}

	// 確認が完了状態になっていないことを確認
	ec, err := repo.FindByID(context.Background(), ecID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if ec.SucceededAt != nil {
		t.Error("間違ったコードで確認が完了状態に更新されてしまいました")
	}
}

func TestVerifyEmailConfirmationUsecase_Execute_PasswordResetEvent(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	q := testutil.QueriesWithTx(tx)
	repo := repository.NewEmailConfirmationRepository(q)
	uc := NewVerifyEmailConfirmationUsecase(repo)

	// パスワードリセットイベントのテストデータを作成
	ecID := testutil.NewEmailConfirmationBuilder(t, tx).
		WithEmail("password-reset@example.com").
		WithEvent(model.EmailConfirmationEventPasswordReset).
		WithCode("MNO345").
		WithStartedAt(time.Now()).
		Build()

	// 確認コードを検証（イベント種別に関係なく検証できる）
	err := uc.Execute(context.Background(), VerifyEmailConfirmationInput{
		EmailConfirmationID: ecID,
		Code:                "MNO345",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
}
