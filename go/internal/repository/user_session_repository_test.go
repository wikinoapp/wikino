package repository

import (
	"context"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestUserSessionRepository_Create(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewUserSessionRepository(q)

	// テストユーザーを作成
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("session@example.com").
		WithAtname("sessionuser").
		Build()

	t.Run("セッションを作成できる", func(t *testing.T) {
		now := time.Now()
		input := CreateInput{
			UserID:     userID,
			Token:      "test-token-12345",
			IPAddress:  "192.168.1.1",
			UserAgent:  "Mozilla/5.0",
			SignedInAt: now,
		}

		session, err := repo.Create(context.Background(), input)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if session == nil {
			t.Fatal("Create() returned nil, want session")
		}
		if session.UserID != userID {
			t.Errorf("session.UserID = %v, want %v", session.UserID, userID)
		}
		if session.Token != "test-token-12345" {
			t.Errorf("session.Token = %v, want test-token-12345", session.Token)
		}
		if session.IPAddress != "192.168.1.1" {
			t.Errorf("session.IPAddress = %v, want 192.168.1.1", session.IPAddress)
		}
		if session.UserAgent != "Mozilla/5.0" {
			t.Errorf("session.UserAgent = %v, want Mozilla/5.0", session.UserAgent)
		}
	})
}

func TestUserSessionRepository_FindByToken(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewUserSessionRepository(q)

	// テストユーザーを作成
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("findbytoken@example.com").
		WithAtname("findbytokenuser").
		Build()

	// セッションを作成
	input := CreateInput{
		UserID:     userID,
		Token:      "find-by-token-test",
		IPAddress:  "192.168.1.2",
		UserAgent:  "TestAgent",
		SignedInAt: time.Now(),
	}
	_, err := repo.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("セッション作成に失敗: %v", err)
	}

	t.Run("トークンでセッションを取得できる", func(t *testing.T) {
		session, err := repo.FindByToken(context.Background(), "find-by-token-test")
		if err != nil {
			t.Fatalf("FindByToken() error = %v", err)
		}
		if session == nil {
			t.Fatal("FindByToken() returned nil, want session")
		}
		if session.Token != "find-by-token-test" {
			t.Errorf("session.Token = %v, want find-by-token-test", session.Token)
		}
	})

	t.Run("存在しないトークンはnilを返す", func(t *testing.T) {
		session, err := repo.FindByToken(context.Background(), "nonexistent-token")
		if err != nil {
			t.Fatalf("FindByToken() error = %v", err)
		}
		if session != nil {
			t.Errorf("FindByToken() = %v, want nil", session)
		}
	})
}

func TestUserSessionRepository_FindByID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewUserSessionRepository(q)

	// テストユーザーを作成
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("findbyid@example.com").
		WithAtname("findbyiduser").
		Build()

	// セッションを作成
	input := CreateInput{
		UserID:     userID,
		Token:      "find-by-id-test",
		IPAddress:  "192.168.1.3",
		UserAgent:  "TestAgent",
		SignedInAt: time.Now(),
	}
	created, err := repo.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("セッション作成に失敗: %v", err)
	}

	t.Run("IDでセッションを取得できる", func(t *testing.T) {
		session, err := repo.FindByID(context.Background(), created.ID)
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}
		if session == nil {
			t.Fatal("FindByID() returned nil, want session")
		}
		if session.ID != created.ID {
			t.Errorf("session.ID = %v, want %v", session.ID, created.ID)
		}
	})

	t.Run("存在しないIDはnilを返す", func(t *testing.T) {
		session, err := repo.FindByID(context.Background(), "00000000-0000-0000-0000-000000000000")
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}
		if session != nil {
			t.Errorf("FindByID() = %v, want nil", session)
		}
	})
}

func TestUserSessionRepository_Delete(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewUserSessionRepository(q)

	// テストユーザーを作成
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("delete@example.com").
		WithAtname("deleteuser").
		Build()

	// セッションを作成
	input := CreateInput{
		UserID:     userID,
		Token:      "delete-test-token",
		IPAddress:  "192.168.1.4",
		UserAgent:  "TestAgent",
		SignedInAt: time.Now(),
	}
	created, err := repo.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("セッション作成に失敗: %v", err)
	}

	t.Run("セッションを削除できる", func(t *testing.T) {
		err := repo.Delete(context.Background(), created.ID)
		if err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		// 削除後は取得できない
		session, err := repo.FindByID(context.Background(), created.ID)
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}
		if session != nil {
			t.Errorf("Delete() did not delete session, FindByID() = %v, want nil", session)
		}
	})
}

func TestUserSessionRepository_DeleteByToken(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewUserSessionRepository(q)

	// テストユーザーを作成
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("deletebytoken@example.com").
		WithAtname("deletebytokenuser").
		Build()

	// セッションを作成
	input := CreateInput{
		UserID:     userID,
		Token:      "delete-by-token-test",
		IPAddress:  "192.168.1.5",
		UserAgent:  "TestAgent",
		SignedInAt: time.Now(),
	}
	_, err := repo.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("セッション作成に失敗: %v", err)
	}

	t.Run("トークンでセッションを削除できる", func(t *testing.T) {
		err := repo.DeleteByToken(context.Background(), "delete-by-token-test")
		if err != nil {
			t.Fatalf("DeleteByToken() error = %v", err)
		}

		// 削除後は取得できない
		session, err := repo.FindByToken(context.Background(), "delete-by-token-test")
		if err != nil {
			t.Fatalf("FindByToken() error = %v", err)
		}
		if session != nil {
			t.Errorf("DeleteByToken() did not delete session, FindByToken() = %v, want nil", session)
		}
	})
}
