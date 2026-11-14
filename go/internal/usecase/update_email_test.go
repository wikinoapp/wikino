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

// newUpdateEmailUsecaseForTestは、検証に使うリポジトリをvalidationQueriesで、永続化に使う
// リポジトリを共有の接続で組み立てる
func newUpdateEmailUsecaseForTest(db *sql.DB, validationQueries *query.Queries) *UpdateEmailUsecase {
	queries := query.New(db)
	return NewUpdateEmailUsecase(
		db,
		validator.NewSettingsEmailConfirmationUpdateValidator(
			repository.NewEmailConfirmationRepository(validationQueries),
			repository.NewUserRepository(validationQueries),
		),
		repository.NewEmailConfirmationRepository(queries),
		repository.NewUserRepository(queries),
	)
}

func TestUpdateEmailUsecase_Execute_確認を完了してメールアドレスを変更する(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	db := testutil.GetTestDB()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("update-email-uc-old@example.com").
		WithAtname("update_email_uc").
		Build()
	confirmationID := testutil.NewEmailConfirmationBuilderDB(t, db).
		WithEmail("update-email-uc-new@example.com").
		WithEvent(model.EmailConfirmationEventEmailUpdate).
		WithCode("UE0001").
		Build()

	output, err := newUpdateEmailUsecaseForTest(db, query.New(db)).Execute(ctx, UpdateEmailInput{
		UserID:              userID,
		EmailConfirmationID: confirmationID,
		Code:                "UE0001",
	})
	if err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}
	if output.User.Email != "update-email-uc-new@example.com" {
		t.Errorf("User.Email = %q、期待値 = %q", output.User.Email, "update-email-uc-new@example.com")
	}

	confirmation, err := repository.NewEmailConfirmationRepository(query.New(db)).FindByID(ctx, confirmationID)
	if err != nil || confirmation == nil {
		t.Fatalf("確認の取得に失敗: 確認 = %v、エラー = %v", confirmation, err)
	}
	if !confirmation.IsSucceeded() {
		t.Error("確認が完了状態になっていない")
	}
}

func TestUpdateEmailUsecase_Execute_検証に失敗したら変更しない(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	db := testutil.GetTestDB()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("update-email-uc-invalid@example.com").
		WithAtname("update_email_inval").
		Build()
	confirmationID := testutil.NewEmailConfirmationBuilderDB(t, db).
		WithEmail("update-email-uc-invalid-new@example.com").
		WithEvent(model.EmailConfirmationEventEmailUpdate).
		WithCode("UE0002").
		Build()

	_, err := newUpdateEmailUsecaseForTest(db, query.New(db)).Execute(ctx, UpdateEmailInput{
		UserID:              userID,
		EmailConfirmationID: confirmationID,
		Code:                "XYZ999",
	})
	if ve := model.AsValidationError(err); ve == nil {
		t.Fatalf("ValidationErrorを期待したが、%vだった", err)
	}

	user, err := repository.NewUserRepository(query.New(db)).FindByID(ctx, userID)
	if err != nil || user == nil {
		t.Fatalf("ユーザーの取得に失敗: ユーザー = %v、エラー = %v", user, err)
	}
	if user.Email != "update-email-uc-invalid@example.com" {
		t.Errorf("Email = %q、変わっていないことを期待", user.Email)
	}
	confirmation, err := repository.NewEmailConfirmationRepository(query.New(db)).FindByID(ctx, confirmationID)
	if err != nil || confirmation == nil {
		t.Fatalf("確認の取得に失敗: 確認 = %v、エラー = %v", confirmation, err)
	}
	if confirmation.IsSucceeded() {
		t.Error("検証に失敗したのに確認が完了状態になっている")
	}
}

func TestUpdateEmailUsecase_Execute_削除済みのユーザーは見つからない(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	db := testutil.GetTestDB()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("update-email-uc-discarded@example.com").
		WithAtname("update_email_disc").
		Build()
	if _, err := db.ExecContext(ctx, "UPDATE users SET discarded_at = NOW() WHERE id = $1", string(userID)); err != nil {
		t.Fatalf("ユーザーの削除に失敗: %v", err)
	}
	confirmationID := testutil.NewEmailConfirmationBuilderDB(t, db).
		WithEmail("update-email-uc-discarded-new@example.com").
		WithEvent(model.EmailConfirmationEventEmailUpdate).
		WithCode("UE0003").
		Build()

	_, err := newUpdateEmailUsecaseForTest(db, query.New(db)).Execute(ctx, UpdateEmailInput{
		UserID:              userID,
		EmailConfirmationID: confirmationID,
		Code:                "UE0003",
	})
	ae := model.AsAppError(err)
	if ae == nil || ae.Code != model.AppErrCodeResourceNotFound {
		t.Fatalf("ResourceNotFoundのAppErrorを期待したが、%vだった", err)
	}

	// 確認の完了もロールバックされ、同じ確認コードを使い回せない状態にもならない
	confirmation, err := repository.NewEmailConfirmationRepository(query.New(db)).FindByID(ctx, confirmationID)
	if err != nil || confirmation == nil {
		t.Fatalf("確認の取得に失敗: 確認 = %v、エラー = %v", confirmation, err)
	}
	if confirmation.IsSucceeded() {
		t.Error("ユーザーを更新できなかったのに確認が完了状態になっている")
	}
}

func TestUpdateEmailUsecase_Execute_メールアドレスの競合を検証エラーにする(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	db := testutil.GetTestDB()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("update-email-race-self@example.com").
		WithAtname("email_race_self").
		Build()
	const newEmail = "update-email-race-new@example.com"
	confirmationID := testutil.NewEmailConfirmationBuilderDB(t, db).
		WithEmail(newEmail).
		WithEvent(model.EmailConfirmationEventEmailUpdate).
		WithCode("UE0004").
		Build()

	snapshot, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		t.Fatalf("検証用トランザクションの開始に失敗: %v", err)
	}
	defer func() { _ = snapshot.Rollback() }()
	validationQueries := query.New(db).WithTx(snapshot)
	exists, err := repository.NewUserRepository(validationQueries).ExistsByEmail(ctx, newEmail)
	if err != nil || exists {
		t.Fatalf("競合前の存在確認: exists = %v、エラー = %v", exists, err)
	}

	// 検証用のスナップショットからは見えないユーザーを別接続で作り、一意制約でだけ競合させる。
	competingID := testutil.NewUserBuilderDB(t, db).
		WithEmail(newEmail).
		WithAtname("email_race_other").
		Build()
	t.Cleanup(func() {
		if _, err := db.ExecContext(context.Background(), "DELETE FROM users WHERE id IN ($1, $2)", string(userID), string(competingID)); err != nil {
			t.Errorf("競合テストのユーザーの削除に失敗: %v", err)
		}
	})

	output, err := newUpdateEmailUsecaseForTest(db, validationQueries).Execute(ctx, UpdateEmailInput{
		UserID:              userID,
		EmailConfirmationID: confirmationID,
		Code:                "UE0004",
	})
	if output != nil {
		t.Errorf("output = %v、期待値 = nil", output)
	}
	ve := model.AsValidationError(err)
	if ve == nil {
		t.Fatalf("ValidationErrorを期待したが、%vだった", err)
	}
	if len(ve.Global) != 1 || ve.Global[0] != "このメールアドレスは既に登録されています" {
		t.Errorf("フォーム全体のエラー = %q、期待した重複エラーと一致しない", ve.Global)
	}

	user, err := repository.NewUserRepository(query.New(db)).FindByID(ctx, userID)
	if err != nil || user == nil {
		t.Fatalf("ユーザーの取得に失敗: ユーザー = %v、エラー = %v", user, err)
	}
	if user.Email != "update-email-race-self@example.com" {
		t.Errorf("Email = %q、変わっていないことを期待", user.Email)
	}
	confirmation, err := repository.NewEmailConfirmationRepository(query.New(db)).FindByID(ctx, confirmationID)
	if err != nil || confirmation == nil {
		t.Fatalf("確認の取得に失敗: 確認 = %v、エラー = %v", confirmation, err)
	}
	if confirmation.IsSucceeded() {
		t.Error("メールアドレスを更新できなかったのに確認が完了状態になっている")
	}
}
