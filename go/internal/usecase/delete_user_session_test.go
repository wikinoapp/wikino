package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestDeleteUserSessionUsecase_Execute(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	userSessionRepo := repository.NewUserSessionRepository(q)
	uc := NewDeleteUserSessionUsecase(userSessionRepo)

	// テストユーザーを作成
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("delete-session@example.com").
		WithAtname("deletesessionuser").
		Build()

	t.Run("セッションを削除できる", func(t *testing.T) {
		// セッションを作成
		sessionToken := testutil.NewSessionBuilder(t, tx).
			WithUserID(userID).
			BuildAndGetToken()

		// セッションが存在することを確認
		session, err := userSessionRepo.FindByToken(context.Background(), sessionToken)
		if err != nil {
			t.Fatalf("FindByToken() error = %v", err)
		}
		if session == nil {
			t.Fatal("session should exist before deletion")
		}

		// UseCase を実行
		err = uc.Execute(context.Background(), DeleteUserSessionInput{
			Token: sessionToken,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		// セッションが削除されていることを確認
		session, err = userSessionRepo.FindByToken(context.Background(), sessionToken)
		if err != nil {
			t.Fatalf("FindByToken() after delete error = %v", err)
		}
		if session != nil {
			t.Error("session should be deleted from database")
		}
	})

	t.Run("存在しないトークンでもエラーにならない", func(t *testing.T) {
		err := uc.Execute(context.Background(), DeleteUserSessionInput{
			Token: "non-existent-token",
		})
		if err != nil {
			t.Fatalf("Execute() error = %v, want nil for non-existent token", err)
		}
	})
}
