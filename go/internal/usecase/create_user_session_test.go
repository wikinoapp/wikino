package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestCreateUserSessionUsecase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("正常系: セッションを作成できる", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		userSessionRepo := repository.NewUserSessionRepository(q)
		uc := NewCreateUserSessionUsecase(userSessionRepo)

		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("create-session@example.com").
			WithAtname("createsession").
			Build()

		output, err := uc.Execute(context.Background(), CreateUserSessionInput{
			UserID:    userID,
			IPAddress: "192.168.1.1",
			UserAgent: "Mozilla/5.0",
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("Execute() returned nil, want output")
		}
		if output.Token == "" {
			t.Error("Execute() returned empty token")
		}

		// DBに保存されていることを確認
		session, err := userSessionRepo.FindByToken(context.Background(), output.Token)
		if err != nil {
			t.Fatalf("FindByToken() error = %v", err)
		}
		if session == nil {
			t.Fatal("FindByToken() returned nil, want session")
		}
		if session.UserID != userID {
			t.Errorf("session.UserID = %v, want %v", session.UserID, userID)
		}
		if session.IPAddress != "192.168.1.1" {
			t.Errorf("session.IPAddress = %v, want 192.168.1.1", session.IPAddress)
		}
		if session.UserAgent != "Mozilla/5.0" {
			t.Errorf("session.UserAgent = %v, want Mozilla/5.0", session.UserAgent)
		}
	})

	t.Run("正常系: 空のIPアドレスとUserAgentでもセッションを作成できる", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		userSessionRepo := repository.NewUserSessionRepository(q)
		uc := NewCreateUserSessionUsecase(userSessionRepo)

		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("empty-fields@example.com").
			WithAtname("emptyfields").
			Build()

		output, err := uc.Execute(context.Background(), CreateUserSessionInput{
			UserID:    userID,
			IPAddress: "",
			UserAgent: "",
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if output == nil {
			t.Fatal("Execute() returned nil, want output")
		}
		if output.Token == "" {
			t.Error("Execute() returned empty token")
		}

		session, err := userSessionRepo.FindByToken(context.Background(), output.Token)
		if err != nil {
			t.Fatalf("FindByToken() error = %v", err)
		}
		if session == nil {
			t.Fatal("FindByToken() returned nil, want session")
		}
		if session.IPAddress != "" {
			t.Errorf("session.IPAddress = %v, want empty string", session.IPAddress)
		}
		if session.UserAgent != "" {
			t.Errorf("session.UserAgent = %v, want empty string", session.UserAgent)
		}
	})

	t.Run("正常系: 各呼び出しで異なるトークンが生成される", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		userSessionRepo := repository.NewUserSessionRepository(q)
		uc := NewCreateUserSessionUsecase(userSessionRepo)

		userID := testutil.NewUserBuilder(t, tx).
			WithEmail("unique-token@example.com").
			WithAtname("uniquetoken").
			Build()

		input := CreateUserSessionInput{
			UserID:    userID,
			IPAddress: "192.168.1.2",
			UserAgent: "TestAgent",
		}

		output1, err := uc.Execute(context.Background(), input)
		if err != nil {
			t.Fatalf("Execute() first call error = %v", err)
		}

		output2, err := uc.Execute(context.Background(), input)
		if err != nil {
			t.Fatalf("Execute() second call error = %v", err)
		}

		if output1.Token == output2.Token {
			t.Error("Execute() returned same token for different calls")
		}
	})

	t.Run("異常系: 存在しないユーザーIDの場合はエラーを返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		userSessionRepo := repository.NewUserSessionRepository(q)
		uc := NewCreateUserSessionUsecase(userSessionRepo)

		_, err := uc.Execute(context.Background(), CreateUserSessionInput{
			UserID:    "00000000-0000-0000-0000-000000000000",
			IPAddress: "192.168.1.1",
			UserAgent: "Mozilla/5.0",
		})
		if err == nil {
			t.Error("Execute() expected error for non-existent user, got nil")
		}
	})
}
