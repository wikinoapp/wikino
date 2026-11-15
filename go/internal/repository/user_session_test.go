package repository

import (
	"context"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestUserSessionRepository_Create(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
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
			t.Fatalf("Create()のエラー = %v", err)
		}
		if session == nil {
			t.Fatal("Create()がnilを返した、期待値 = セッション")
		}
		if session.UserID != userID {
			t.Errorf("session.UserID = %v、期待値 = %v", session.UserID, userID)
		}
		if session.Token != "test-token-12345" {
			t.Errorf("session.Token = %v、期待値 = test-token-12345", session.Token)
		}
		if session.IPAddress != "192.168.1.1" {
			t.Errorf("session.IPAddress = %v、期待値 = 192.168.1.1", session.IPAddress)
		}
		if session.UserAgent != "Mozilla/5.0" {
			t.Errorf("session.UserAgent = %v、期待値 = Mozilla/5.0", session.UserAgent)
		}
	})
}

func TestUserSessionRepository_FindByToken(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
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
			t.Fatalf("FindByToken()のエラー = %v", err)
		}
		if session == nil {
			t.Fatal("FindByToken()がnilを返した、期待値 = セッション")
		}
		if session.Token != "find-by-token-test" {
			t.Errorf("session.Token = %v、期待値 = find-by-token-test", session.Token)
		}
	})

	t.Run("存在しないトークンはnilを返す", func(t *testing.T) {
		session, err := repo.FindByToken(context.Background(), "nonexistent-token")
		if err != nil {
			t.Fatalf("FindByToken()のエラー = %v", err)
		}
		if session != nil {
			t.Errorf("FindByToken() = %v、期待値 = nil", session)
		}
	})
}

func TestUserSessionRepository_FindByID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
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
			t.Fatalf("FindByID()のエラー = %v", err)
		}
		if session == nil {
			t.Fatal("FindByID()がnilを返した、期待値 = セッション")
		}
		if session.ID != created.ID {
			t.Errorf("session.ID = %v、期待値 = %v", session.ID, created.ID)
		}
	})

	t.Run("存在しないIDはnilを返す", func(t *testing.T) {
		session, err := repo.FindByID(context.Background(), "00000000-0000-0000-0000-000000000000")
		if err != nil {
			t.Fatalf("FindByID()のエラー = %v", err)
		}
		if session != nil {
			t.Errorf("FindByID() = %v、期待値 = nil", session)
		}
	})
}

func TestUserSessionRepository_Delete(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
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
			t.Fatalf("Delete()のエラー = %v", err)
		}

		// 削除後は取得できない
		session, err := repo.FindByID(context.Background(), created.ID)
		if err != nil {
			t.Fatalf("FindByID()のエラー = %v", err)
		}
		if session != nil {
			t.Errorf("Delete()でセッションが削除されていない: FindByID() = %v、期待値 = nil", session)
		}
	})
}

func TestUserSessionRepository_DeleteByToken(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
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
			t.Fatalf("DeleteByToken()のエラー = %v", err)
		}

		// 削除後は取得できない
		session, err := repo.FindByToken(context.Background(), "delete-by-token-test")
		if err != nil {
			t.Fatalf("FindByToken()のエラー = %v", err)
		}
		if session != nil {
			t.Errorf("DeleteByToken()でセッションが削除されていない: FindByToken() = %v、期待値 = nil", session)
		}
	})
}

func TestUserSessionRepository_DeleteByUserID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewUserSessionRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("repo-sessions-owner@example.com").
		WithAtname("repo_sessions_owner").
		Build()
	otherID := testutil.NewUserBuilder(t, tx).
		WithEmail("repo-sessions-other@example.com").
		WithAtname("repo_sessions_other").
		Build()
	token1 := testutil.NewSessionBuilder(t, tx).WithUserID(userID).WithToken("repo-sessions-owner-1").BuildAndGetToken()
	token2 := testutil.NewSessionBuilder(t, tx).WithUserID(userID).WithToken("repo-sessions-owner-2").BuildAndGetToken()
	otherToken := testutil.NewSessionBuilder(t, tx).WithUserID(otherID).WithToken("repo-sessions-other").BuildAndGetToken()

	if err := repo.DeleteByUserID(ctx, userID); err != nil {
		t.Fatalf("DeleteByUserID()のエラー = %v", err)
	}

	for _, token := range []string{token1, token2} {
		if s, err := repo.FindByToken(ctx, token); err != nil || s != nil {
			t.Errorf("FindByToken(%q) = %v, %v、期待値 = nil, nil", token, s, err)
		}
	}
	if s, err := repo.FindByToken(ctx, otherToken); err != nil || s == nil {
		t.Errorf("ほかのユーザーのFindByToken() = %v, %v、セッションが残ることを期待", s, err)
	}
}
