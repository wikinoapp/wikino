package usecase

import (
	"context"
	"database/sql"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
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
