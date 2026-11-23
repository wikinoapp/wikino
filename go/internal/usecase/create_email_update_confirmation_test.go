package usecase

import (
	"context"
	"database/sql"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func newCreateEmailUpdateConfirmationUsecaseForTest(tx *sql.Tx, inserter dispatcher.JobInserter) (*CreateEmailUpdateConfirmationUsecase, *repository.EmailConfirmationRepository) {
	q := testutil.QueriesWithTx(tx)
	userRepo := repository.NewUserRepository(q)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(q)
	createEmailConfirmationUC := NewCreateEmailConfirmationUsecase(
		&config.Config{Env: "test", Domain: "wikino.app"},
		emailConfirmationRepo,
		dispatcher.NewDispatcher(inserter, nil),
		validator.NewEmailConfirmationCreateValidator(userRepo),
	)
	return NewCreateEmailUpdateConfirmationUsecase(
		validator.NewSettingsEmailUpdateValidator(userRepo, repository.NewUserPasswordRepository(q)),
		createEmailConfirmationUC,
	), emailConfirmationRepo
}

func emailUpdateUser(t *testing.T, tx *sql.Tx, atname string) model.UserID {
	t.Helper()

	digest, err := auth.HashPassword("password123")
	if err != nil {
		t.Fatalf("パスワードのハッシュ化に失敗: %v", err)
	}
	return testutil.NewUserBuilder(t, tx).
		WithEmail(atname + "@example.com").
		WithAtname(atname).
		BuildWithPassword(digest)
}

func TestCreateEmailUpdateConfirmationUsecase_Execute_確認コードを送る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	userID := emailUpdateUser(t, tx, "email_update_send")
	mock := &mockJobInserter{}
	uc, emailConfirmationRepo := newCreateEmailUpdateConfirmationUsecaseForTest(tx, mock)

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	output, err := uc.Execute(ctx, CreateEmailUpdateConfirmationInput{
		UserID:   userID,
		NewEmail: "email-update-send-new@example.com",
		Password: "password123",
		Locale:   "ja",
	})
	if err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}

	confirmation, err := emailConfirmationRepo.FindByID(ctx, output.EmailConfirmationID)
	if err != nil || confirmation == nil {
		t.Fatalf("確認の取得に失敗: 確認 = %v、エラー = %v", confirmation, err)
	}
	if confirmation.Email != "email-update-send-new@example.com" {
		t.Errorf("Email = %q、期待値 = %q", confirmation.Email, "email-update-send-new@example.com")
	}
	if confirmation.Event != model.EmailConfirmationEventEmailUpdate {
		t.Errorf("Event = %d、期待値 = %d", confirmation.Event, model.EmailConfirmationEventEmailUpdate)
	}
	if confirmation.UserID == nil || *confirmation.UserID != userID {
		t.Errorf("開始ユーザー = %v、期待値 = %s", confirmation.UserID, userID)
	}
	if !mock.called {
		t.Error("メールの送信のジョブが投入されていない")
	}
	args, ok := mock.args.(dispatcher.SendEmailConfirmationArgs)
	if !ok || args.Email != "email-update-send-new@example.com" {
		t.Errorf("ジョブの引数 = %+v、新しいメールアドレス宛てを期待", mock.args)
	}
}

func TestCreateEmailUpdateConfirmationUsecase_Execute_パスワードが違うと送らない(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	userID := emailUpdateUser(t, tx, "email_update_wrong")
	mock := &mockJobInserter{}
	uc, _ := newCreateEmailUpdateConfirmationUsecaseForTest(tx, mock)

	ctx := i18n.SetLocale(context.Background(), i18n.LangJa)
	output, err := uc.Execute(ctx, CreateEmailUpdateConfirmationInput{
		UserID:   userID,
		NewEmail: "email-update-wrong-new@example.com",
		Password: "wrong-password",
		Locale:   "ja",
	})
	if output != nil {
		t.Errorf("output = %v、期待値 = nil", output)
	}
	ve := model.AsValidationError(err)
	if ve == nil || !ve.HasFieldError("password") {
		t.Fatalf("passwordのValidationErrorを期待したが、%vだった", err)
	}
	if mock.called {
		t.Error("パスワードが違うのにメールの送信のジョブが投入された")
	}

	var count int
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM email_confirmations WHERE email = $1", "email-update-wrong-new@example.com").Scan(&count); err != nil {
		t.Fatalf("確認の件数の取得に失敗: %v", err)
	}
	if count != 0 {
		t.Errorf("確認の件数 = %d、期待値 = 0", count)
	}
}
