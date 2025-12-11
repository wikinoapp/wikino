package repository

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestUserRepository_FindByID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewUserRepository(q)

	// テストユーザーを作成
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("findbyid@example.com").
		WithAtname("findbyid").
		Build()

	t.Run("存在するユーザーを取得できる", func(t *testing.T) {
		user, err := repo.FindByID(context.Background(), userID)
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}
		if user == nil {
			t.Fatal("FindByID() returned nil, want user")
		}
		if user.ID != userID {
			t.Errorf("user.ID = %v, want %v", user.ID, userID)
		}
		if user.Email != "findbyid@example.com" {
			t.Errorf("user.Email = %v, want findbyid@example.com", user.Email)
		}
		if user.Atname != "findbyid" {
			t.Errorf("user.Atname = %v, want findbyid", user.Atname)
		}
	})

	t.Run("存在しないユーザーはnilを返す", func(t *testing.T) {
		user, err := repo.FindByID(context.Background(), "00000000-0000-0000-0000-000000000000")
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}
		if user != nil {
			t.Errorf("FindByID() = %v, want nil", user)
		}
	})
}

func TestUserRepository_FindByEmail(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewUserRepository(q)

	// テストユーザーを作成
	testutil.NewUserBuilder(t, tx).
		WithEmail("findbyemail@example.com").
		WithAtname("findbyemail").
		Build()

	t.Run("存在するユーザーをメールアドレスで取得できる", func(t *testing.T) {
		user, err := repo.FindByEmail(context.Background(), "findbyemail@example.com")
		if err != nil {
			t.Fatalf("FindByEmail() error = %v", err)
		}
		if user == nil {
			t.Fatal("FindByEmail() returned nil, want user")
		}
		if user.Email != "findbyemail@example.com" {
			t.Errorf("user.Email = %v, want findbyemail@example.com", user.Email)
		}
	})

	t.Run("存在しないメールアドレスはnilを返す", func(t *testing.T) {
		user, err := repo.FindByEmail(context.Background(), "notexist@example.com")
		if err != nil {
			t.Fatalf("FindByEmail() error = %v", err)
		}
		if user != nil {
			t.Errorf("FindByEmail() = %v, want nil", user)
		}
	})
}

func TestUserRepository_FindByAtname(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewUserRepository(q)

	// テストユーザーを作成
	testutil.NewUserBuilder(t, tx).
		WithEmail("findbyatname@example.com").
		WithAtname("findbyatname").
		Build()

	t.Run("存在するユーザーをアットネームで取得できる", func(t *testing.T) {
		user, err := repo.FindByAtname(context.Background(), "findbyatname")
		if err != nil {
			t.Fatalf("FindByAtname() error = %v", err)
		}
		if user == nil {
			t.Fatal("FindByAtname() returned nil, want user")
		}
		if user.Atname != "findbyatname" {
			t.Errorf("user.Atname = %v, want findbyatname", user.Atname)
		}
	})

	t.Run("存在しないアットネームはnilを返す", func(t *testing.T) {
		user, err := repo.FindByAtname(context.Background(), "notexist")
		if err != nil {
			t.Fatalf("FindByAtname() error = %v", err)
		}
		if user != nil {
			t.Errorf("FindByAtname() = %v, want nil", user)
		}
	})
}

func TestUserRepository_toModel(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	q := testutil.QueriesWithTx(tx)
	repo := NewUserRepository(q)

	// テストユーザーを作成
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("tomodel@example.com").
		WithAtname("tomodel").
		WithName("To Model User").
		Build()

	user, err := repo.FindByID(context.Background(), userID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	t.Run("LocaleがLocaleJaとして変換される", func(t *testing.T) {
		if user.Locale != model.LocaleJa {
			t.Errorf("user.Locale = %v, want LocaleJa", user.Locale)
		}
	})

	t.Run("DiscardedAtがnilとして変換される", func(t *testing.T) {
		if user.DiscardedAt != nil {
			t.Errorf("user.DiscardedAt = %v, want nil", user.DiscardedAt)
		}
	})
}
