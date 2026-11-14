package repository

import (
	"context"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestUserRepository_FindByID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
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
			t.Fatalf("FindByID()のエラー = %v", err)
		}
		if user == nil {
			t.Fatal("FindByID()がnilを返した、期待値 = ユーザー")
		}
		if user.ID != userID {
			t.Errorf("user.ID = %v、期待値 = %v", user.ID, userID)
		}
		if user.Email != "findbyid@example.com" {
			t.Errorf("user.Email = %v、期待値 = findbyid@example.com", user.Email)
		}
		if user.Atname != "findbyid" {
			t.Errorf("user.Atname = %v、期待値 = findbyid", user.Atname)
		}
	})

	t.Run("存在しないユーザーはnilを返す", func(t *testing.T) {
		user, err := repo.FindByID(context.Background(), "00000000-0000-0000-0000-000000000000")
		if err != nil {
			t.Fatalf("FindByID()のエラー = %v", err)
		}
		if user != nil {
			t.Errorf("FindByID() = %v、期待値 = nil", user)
		}
	})
}

func TestUserRepository_FindByEmail(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
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
			t.Fatalf("FindByEmail()のエラー = %v", err)
		}
		if user == nil {
			t.Fatal("FindByEmail()がnilを返した、期待値 = ユーザー")
		}
		if user.Email != "findbyemail@example.com" {
			t.Errorf("user.Email = %v、期待値 = findbyemail@example.com", user.Email)
		}
	})

	t.Run("存在しないメールアドレスはnilを返す", func(t *testing.T) {
		user, err := repo.FindByEmail(context.Background(), "notexist@example.com")
		if err != nil {
			t.Fatalf("FindByEmail()のエラー = %v", err)
		}
		if user != nil {
			t.Errorf("FindByEmail() = %v、期待値 = nil", user)
		}
	})
}

func TestUserRepository_FindByAtname(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
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
			t.Fatalf("FindByAtname()のエラー = %v", err)
		}
		if user == nil {
			t.Fatal("FindByAtname()がnilを返した、期待値 = ユーザー")
		}
		if user.Atname != "findbyatname" {
			t.Errorf("user.Atname = %v、期待値 = findbyatname", user.Atname)
		}
	})

	t.Run("存在しないアットネームはnilを返す", func(t *testing.T) {
		user, err := repo.FindByAtname(context.Background(), "notexist")
		if err != nil {
			t.Fatalf("FindByAtname()のエラー = %v", err)
		}
		if user != nil {
			t.Errorf("FindByAtname() = %v、期待値 = nil", user)
		}
	})
}

func TestUserRepository_toModel(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
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
		t.Fatalf("FindByID()のエラー = %v", err)
	}

	t.Run("LocaleがLocaleJaとして変換される", func(t *testing.T) {
		if user.Locale != model.LocaleJa {
			t.Errorf("user.Locale = %v、期待値 = LocaleJa", user.Locale)
		}
	})

	t.Run("DiscardedAtがnilとして変換される", func(t *testing.T) {
		if user.DiscardedAt != nil {
			t.Errorf("user.DiscardedAt = %v、期待値 = nil", user.DiscardedAt)
		}
	})
}

func TestUserRepository_ExistsByAtname(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewUserRepository(testutil.QueriesWithTx(tx))

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("exists-by-atname@example.com").
		WithAtname("exists_by_atname").
		Build()
	discardedID := testutil.NewUserBuilder(t, tx).
		WithEmail("exists-by-atname-discarded@example.com").
		WithAtname("exists_discarded").
		Build()
	if _, err := tx.ExecContext(context.Background(), "UPDATE users SET discarded_at = NOW() WHERE id = $1", string(discardedID)); err != nil {
		t.Fatalf("ユーザーの削除に失敗: %v", err)
	}

	tests := []struct {
		name      string
		atname    string
		excludeID *model.UserID
		want      bool
	}{
		{name: "使われているアットネーム", atname: "exists_by_atname", want: true},
		{name: "削除済みのユーザーのアットネームも使われているとみなす", atname: "exists_discarded", want: true},
		{name: "使われていないアットネーム", atname: "exists_unused", want: false},
		{name: "除外したユーザー自身のアットネームは数えない", atname: "exists_by_atname", excludeID: &userID, want: false},
		{name: "除外したユーザー以外が使うアットネームは数える", atname: "exists_discarded", excludeID: &userID, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.ExistsByAtname(context.Background(), tt.atname, tt.excludeID)
			if err != nil {
				t.Fatalf("ExistsByAtname()のエラー = %v", err)
			}
			if got != tt.want {
				t.Errorf("ExistsByAtname(%q) = %v、期待値 = %v", tt.atname, got, tt.want)
			}
		})
	}
}

func TestUserRepository_UpdateProfile(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewUserRepository(testutil.QueriesWithTx(tx))

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("update-profile@example.com").
		WithAtname("update_profile").
		Build()

	t.Run("アットネーム・名前・自己紹介を更新する", func(t *testing.T) {
		user, err := repo.UpdateProfile(context.Background(), UpdateProfileInput{
			ID:          userID,
			Atname:      "updated_profile",
			Name:        "新しい名前",
			Description: "新しい自己紹介",
		})
		if err != nil {
			t.Fatalf("UpdateProfile()のエラー = %v", err)
		}
		if user == nil {
			t.Fatal("UpdateProfile()がnilを返した、期待値 = ユーザー")
		}
		if user.Atname != "updated_profile" || user.Name != "新しい名前" || user.Description != "新しい自己紹介" {
			t.Errorf("更新後のユーザー = %+v", user)
		}
	})

	t.Run("削除済みのユーザーは更新せずnilを返す", func(t *testing.T) {
		if _, err := tx.ExecContext(context.Background(), "UPDATE users SET discarded_at = NOW() WHERE id = $1", string(userID)); err != nil {
			t.Fatalf("ユーザーの削除に失敗: %v", err)
		}
		user, err := repo.UpdateProfile(context.Background(), UpdateProfileInput{ID: userID, Atname: "after_discard"})
		if err != nil {
			t.Fatalf("UpdateProfile()のエラー = %v", err)
		}
		if user != nil {
			t.Errorf("UpdateProfile() = %+v、期待値 = nil", user)
		}
	})
}

func TestUserRepository_ExistsByEmail(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewUserRepository(testutil.QueriesWithTx(tx))

	testutil.NewUserBuilder(t, tx).
		WithEmail("exists-by-email@example.com").
		WithAtname("exists_by_email").
		Build()
	discardedID := testutil.NewUserBuilder(t, tx).
		WithEmail("exists-by-email-discarded@example.com").
		WithAtname("exists_email_disc").
		Build()
	if _, err := tx.ExecContext(context.Background(), "UPDATE users SET discarded_at = NOW() WHERE id = $1", string(discardedID)); err != nil {
		t.Fatalf("ユーザーの削除に失敗: %v", err)
	}

	tests := []struct {
		name  string
		email string
		want  bool
	}{
		{name: "使われているメールアドレス", email: "exists-by-email@example.com", want: true},
		{name: "削除済みのユーザーのメールアドレスも使われているとみなす", email: "exists-by-email-discarded@example.com", want: true},
		{name: "使われていないメールアドレス", email: "exists-by-email-unused@example.com", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.ExistsByEmail(context.Background(), tt.email)
			if err != nil {
				t.Fatalf("ExistsByEmail()のエラー = %v", err)
			}
			if got != tt.want {
				t.Errorf("ExistsByEmail(%q) = %v、期待値 = %v", tt.email, got, tt.want)
			}
		})
	}
}

func TestUserRepository_UpdateEmail(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewUserRepository(testutil.QueriesWithTx(tx))

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("update-email@example.com").
		WithAtname("update_email").
		Build()

	t.Run("メールアドレスを更新する", func(t *testing.T) {
		user, err := repo.UpdateEmail(context.Background(), userID, "updated-email@example.com")
		if err != nil {
			t.Fatalf("UpdateEmail()のエラー = %v", err)
		}
		if user == nil {
			t.Fatal("UpdateEmail()がnilを返した、期待値 = ユーザー")
		}
		if user.Email != "updated-email@example.com" {
			t.Errorf("Email = %q、期待値 = %q", user.Email, "updated-email@example.com")
		}
	})

	t.Run("削除済みのユーザーは更新せずnilを返す", func(t *testing.T) {
		if _, err := tx.ExecContext(context.Background(), "UPDATE users SET discarded_at = NOW() WHERE id = $1", string(userID)); err != nil {
			t.Fatalf("ユーザーの削除に失敗: %v", err)
		}
		user, err := repo.UpdateEmail(context.Background(), userID, "after-discard@example.com")
		if err != nil {
			t.Fatalf("UpdateEmail()のエラー = %v", err)
		}
		if user != nil {
			t.Errorf("UpdateEmail() = %+v、期待値 = nil", user)
		}
	})
}

func TestUserRepository_DiscardとUndiscard(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewUserRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("repo-discard-user@example.com").
		WithAtname("repo_discard_user").
		Build()
	discardedAt := time.Now().Truncate(time.Microsecond)

	discarded, err := repo.Discard(ctx, userID, discardedAt)
	if err != nil {
		t.Fatalf("Discard()のエラー = %v", err)
	}
	if !discarded {
		t.Fatal("Discard() = false、期待値 = true")
	}
	// 論理削除したユーザーはメールアドレスで引けなくなる (ログインできない)
	if u, err := repo.FindByEmail(ctx, "repo-discard-user@example.com"); err != nil || u != nil {
		t.Errorf("FindByEmail() = %v, %v、期待値 = nil, nil", u, err)
	}

	again, err := repo.Discard(ctx, userID, discardedAt.Add(time.Second))
	if err != nil {
		t.Fatalf("2回目のDiscard()のエラー = %v", err)
	}
	if again {
		t.Error("論理削除したユーザーへのDiscard() = true、期待値 = false")
	}

	// 別の時刻を渡しても、その時刻の論理削除ではないため取り消さない
	if err := repo.Undiscard(ctx, userID, discardedAt.Add(time.Second), time.Now()); err != nil {
		t.Fatalf("Undiscard()のエラー = %v", err)
	}
	if u, _ := repo.FindByEmail(ctx, "repo-discard-user@example.com"); u != nil {
		t.Error("別の時刻を渡したUndiscard()で論理削除が取り消された")
	}

	if err := repo.Undiscard(ctx, userID, discardedAt, time.Now()); err != nil {
		t.Fatalf("Undiscard()のエラー = %v", err)
	}
	if u, err := repo.FindByEmail(ctx, "repo-discard-user@example.com"); err != nil || u == nil {
		t.Errorf("Undiscard()のあとのFindByEmail() = %v, %v、ユーザーを期待", u, err)
	}
}

func TestUserRepository_LockDiscardedとDeleteDiscarded(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	repo := NewUserRepository(testutil.QueriesWithTx(tx))
	ctx := context.Background()
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("repo-delete-user@example.com").
		WithAtname("repo_delete_user").
		Build()

	// 論理削除していないユーザーはロックも削除もしない
	locked, err := repo.LockDiscarded(ctx, userID)
	if err != nil {
		t.Fatalf("LockDiscarded()のエラー = %v", err)
	}
	if locked {
		t.Error("論理削除していないユーザーのLockDiscarded() = true、期待値 = false")
	}
	deleted, err := repo.DeleteDiscarded(ctx, userID)
	if err != nil {
		t.Fatalf("DeleteDiscarded()のエラー = %v", err)
	}
	if deleted {
		t.Error("論理削除していないユーザーのDeleteDiscarded() = true、期待値 = false")
	}

	if _, err := repo.Discard(ctx, userID, time.Now()); err != nil {
		t.Fatalf("Discard()のエラー = %v", err)
	}
	locked, err = repo.LockDiscarded(ctx, userID)
	if err != nil {
		t.Fatalf("LockDiscarded()のエラー = %v", err)
	}
	if !locked {
		t.Error("論理削除したユーザーのLockDiscarded() = false、期待値 = true")
	}
	deleted, err = repo.DeleteDiscarded(ctx, userID)
	if err != nil {
		t.Fatalf("DeleteDiscarded()のエラー = %v", err)
	}
	if !deleted {
		t.Error("論理削除したユーザーのDeleteDiscarded() = false、期待値 = true")
	}
	if u, err := repo.FindByID(ctx, userID); err != nil || u != nil {
		t.Errorf("物理削除したユーザーのFindByID() = %v, %v、期待値 = nil, nil", u, err)
	}
}
