package usecase

import (
	"context"
	"database/sql"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func newUpdateProfileUsecaseForTest(tx *sql.Tx) (*UpdateProfileUsecase, *repository.UserRepository) {
	userRepo := repository.NewUserRepository(testutil.QueriesWithTx(tx))
	return NewUpdateProfileUsecase(validator.NewProfileUpdateValidator(userRepo), userRepo), userRepo
}

func TestUpdateProfileUsecase_Execute_プロフィールを保存する(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("update-profile-uc@example.com").
		WithAtname("update_profile_uc").
		Build()
	uc, userRepo := newUpdateProfileUsecaseForTest(tx)

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	output, err := uc.Execute(ctx, UpdateProfileInput{
		UserID:      userID,
		Atname:      "update_profile_new",
		Name:        "新しい名前",
		Description: "新しい自己紹介",
	})
	if err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}
	if output.User.Atname != "update_profile_new" {
		t.Errorf("User.Atname = %q、期待値 = %q", output.User.Atname, "update_profile_new")
	}

	saved, err := userRepo.FindByID(ctx, userID)
	if err != nil {
		t.Fatalf("FindByID()のエラー = %v", err)
	}
	if saved.Atname != "update_profile_new" || saved.Name != "新しい名前" || saved.Description != "新しい自己紹介" {
		t.Errorf("保存されたユーザー = %+v", saved)
	}
}

func TestUpdateProfileUsecase_Execute_検証に失敗したら保存しない(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("update-profile-invalid@example.com").
		WithAtname("update_profile_invalid").
		WithName("元の名前").
		Build()
	uc, userRepo := newUpdateProfileUsecaseForTest(tx)

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	_, err := uc.Execute(ctx, UpdateProfileInput{
		UserID: userID,
		Atname: "admin",
		Name:   "変えた名前",
	})
	if ve := model.AsValidationError(err); ve == nil || !ve.HasFieldError("atname") {
		t.Fatalf("atnameのValidationErrorを期待したが、%vだった", err)
	}

	saved, err := userRepo.FindByID(ctx, userID)
	if err != nil {
		t.Fatalf("FindByID()のエラー = %v", err)
	}
	if saved.Atname != "update_profile_invalid" || saved.Name != "元の名前" {
		t.Errorf("保存されたユーザー = %+v、変わっていないことを期待", saved)
	}
}

func TestUpdateProfileUsecase_Execute_削除済みのユーザーは見つからない(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("update-profile-discarded@example.com").
		WithAtname("update_profile_gone").
		Build()
	if _, err := tx.ExecContext(context.Background(), "UPDATE users SET discarded_at = NOW() WHERE id = $1", string(userID)); err != nil {
		t.Fatalf("ユーザーの削除に失敗: %v", err)
	}
	uc, _ := newUpdateProfileUsecaseForTest(tx)

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	_, err := uc.Execute(ctx, UpdateProfileInput{UserID: userID, Atname: "update_profile_gone"})

	ae := model.AsAppError(err)
	if ae == nil || ae.Code != model.AppErrCodeResourceNotFound {
		t.Fatalf("ResourceNotFoundのAppErrorを期待したが、%vだった", err)
	}
}

func TestUpdateProfileUsecase_Execute_アットネームの競合を検証エラーにする(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	db := testutil.GetTestDB()
	userRepo := repository.NewUserRepository(query.New(db))
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("update-profile-conflict-owner@example.com").
		WithAtname("profile_race_self").
		WithName("元の名前").
		Build()

	snapshot, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		t.Fatalf("検証用トランザクションの開始に失敗: %v", err)
	}
	defer func() { _ = snapshot.Rollback() }()
	validationRepo := userRepo.WithTx(snapshot)
	const atname = "profile_conflict_new"
	exists, err := validationRepo.ExistsByAtname(ctx, atname, &userID)
	if err != nil || exists {
		t.Fatalf("競合前の存在確認: exists = %v、エラー = %v", exists, err)
	}

	// 検証用のスナップショットからは見えないユーザーを別接続で作り、一意制約でだけ競合させる。
	competingID := testutil.NewUserBuilderDB(t, db).
		WithEmail("update-profile-conflict-other@example.com").
		WithAtname(atname).
		WithName("競合相手の名前").
		Build()
	t.Cleanup(func() {
		if _, err := db.ExecContext(context.Background(), "DELETE FROM users WHERE id IN ($1, $2)", string(userID), string(competingID)); err != nil {
			t.Errorf("競合テストのユーザーの削除に失敗: %v", err)
		}
	})
	before := make([]*model.User, 0, 2)
	for _, id := range []model.UserID{userID, competingID} {
		user, err := userRepo.FindByID(ctx, id)
		if err != nil || user == nil {
			t.Fatalf("更新前のユーザーの取得に失敗: ユーザー = %v、エラー = %v", user, err)
		}
		before = append(before, user)
	}

	uc := NewUpdateProfileUsecase(validator.NewProfileUpdateValidator(validationRepo), userRepo)
	output, err := uc.Execute(ctx, UpdateProfileInput{
		UserID:      userID,
		Atname:      atname,
		Name:        "変更した名前",
		Description: "変更した自己紹介",
	})
	if output != nil {
		t.Errorf("output = %v、期待値 = nil", output)
	}
	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatalf("ValidationErrorを期待したが、%vだった", err)
	}
	if got := ve.GetFieldErrors("atname"); len(got) != 1 || got[0] != "このアットネームは既に使用されています" {
		t.Errorf("atnameのエラー = %q、期待した重複エラーと一致しない", got)
	}

	for _, original := range before {
		user, err := userRepo.FindByID(ctx, original.ID)
		if err != nil || user == nil {
			t.Fatalf("更新後のユーザーの取得に失敗: ユーザー = %v、エラー = %v", user, err)
		}
		if user.Atname != original.Atname || user.Name != original.Name || user.Description != original.Description {
			t.Errorf("ユーザー%sの情報が変わっている: %+v", original.ID, user)
		}
	}
}
