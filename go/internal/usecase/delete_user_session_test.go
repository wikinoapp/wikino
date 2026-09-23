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
			t.Fatalf("FindByToken()のエラー = %v", err)
		}
		if session == nil {
			t.Fatal("削除前にセッションが存在しない")
		}

		// UseCaseを実行
		err = uc.Execute(context.Background(), DeleteUserSessionInput{
			Token: sessionToken,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		// セッションが削除されていることを確認
		session, err = userSessionRepo.FindByToken(context.Background(), sessionToken)
		if err != nil {
			t.Fatalf("削除後のFindByToken()のエラー = %v", err)
		}
		if session != nil {
			t.Error("データベースからセッションが削除されていない")
		}
	})

	t.Run("存在しないトークンでもエラーにならない", func(t *testing.T) {
		err := uc.Execute(context.Background(), DeleteUserSessionInput{
			Token: "non-existent-token",
		})
		if err != nil {
			t.Fatalf("存在しないトークンでのExecute()のエラー = %v、期待値 = nil", err)
		}
	})
}
