package usecase

import (
	"context"
	"testing"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// mockJobInserterはテスト用のモックinserter
type mockJobInserter struct {
	called bool
	args   river.JobArgs
}

func (m *mockJobInserter) Insert(_ context.Context, args river.JobArgs, _ *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	m.called = true
	m.args = args
	return &rivertype.JobInsertResult{}, nil
}

func TestCreateEmailConfirmationUsecase_Execute_Japanese(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)

	cfg := &config.Config{
		Env:    "test",
		Domain: "wikino.app",
	}

	emailConfirmationRepo := repository.NewEmailConfirmationRepository(q)
	mock := &mockJobInserter{}
	d := dispatcher.NewDispatcher(mock)
	userRepo := repository.NewUserRepository(q)
	createValidator := validator.NewEmailConfirmationCreateValidator(userRepo)
	uc := NewCreateEmailConfirmationUsecase(cfg, emailConfirmationRepo, d, createValidator)

	ctx := i18n.SetLocale(context.Background(), "ja")
	input := CreateEmailConfirmationInput{
		Email:  "test@example.com",
		Event:  model.EmailConfirmationEventSignUp,
		Locale: "ja",
	}

	output, err := uc.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}

	if output.EmailConfirmationID == "" {
		t.Error("EmailConfirmationIDが空です")
	}

	// DBに保存されたことを確認
	confirmation, err := emailConfirmationRepo.FindByID(ctx, output.EmailConfirmationID)
	if err != nil {
		t.Fatalf("FindByID()のエラー = %v", err)
	}
	if confirmation == nil {
		t.Fatal("メール確認情報がDBに保存されていません")
	}
	if confirmation.Email != "test@example.com" {
		t.Errorf("Email = %s、期待値 = test@example.com", confirmation.Email)
	}
	if confirmation.Event != model.EmailConfirmationEventSignUp {
		t.Errorf("Event = %d、期待値 = %d", confirmation.Event, model.EmailConfirmationEventSignUp)
	}
	if len(confirmation.Code) != 6 {
		t.Errorf("コードの長さ = %d、期待値 = 6", len(confirmation.Code))
	}

	// エンキューが呼ばれたことを確認
	if !mock.called {
		t.Error("Insertが呼ばれていません")
	}

	// SendEmailConfirmationArgsの検証
	emailArgs, ok := mock.args.(dispatcher.SendEmailConfirmationArgs)
	if !ok {
		t.Fatalf("argsの型がSendEmailConfirmationArgsではありません: %T", mock.args)
	}
	if emailArgs.Email != "test@example.com" {
		t.Errorf("Email = %s、期待値 = test@example.com", emailArgs.Email)
	}
	if emailArgs.Code != confirmation.Code {
		t.Errorf("Code = %s、期待値 = %s", emailArgs.Code, confirmation.Code)
	}
	if emailArgs.Locale != "ja" {
		t.Errorf("Locale = %s、期待値 = ja", emailArgs.Locale)
	}
	if emailArgs.AppURL == "" {
		t.Error("AppURLが空です")
	}
}

func TestCreateEmailConfirmationUsecase_Execute_English(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)

	cfg := &config.Config{
		Env:    "test",
		Domain: "wikino.app",
	}

	emailConfirmationRepo := repository.NewEmailConfirmationRepository(q)
	mock := &mockJobInserter{}
	d := dispatcher.NewDispatcher(mock)
	userRepo := repository.NewUserRepository(q)
	createValidator := validator.NewEmailConfirmationCreateValidator(userRepo)
	uc := NewCreateEmailConfirmationUsecase(cfg, emailConfirmationRepo, d, createValidator)

	ctx := i18n.SetLocale(context.Background(), "en")
	input := CreateEmailConfirmationInput{
		Email:  "english@example.com",
		Event:  model.EmailConfirmationEventSignUp,
		Locale: "en",
	}

	output, err := uc.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}

	if output.EmailConfirmationID == "" {
		t.Error("EmailConfirmationIDが空です")
	}

	// DBに保存されたことを確認
	confirmation, err := emailConfirmationRepo.FindByID(ctx, output.EmailConfirmationID)
	if err != nil {
		t.Fatalf("FindByID()のエラー = %v", err)
	}
	if confirmation == nil {
		t.Fatal("メール確認情報がDBに保存されていません")
	}
	if confirmation.Email != "english@example.com" {
		t.Errorf("Email = %s、期待値 = english@example.com", confirmation.Email)
	}

	// エンキューが呼ばれたことを確認
	if !mock.called {
		t.Error("Insertが呼ばれていません")
	}

	// SendEmailConfirmationArgsの検証 (英語)
	emailArgs, ok := mock.args.(dispatcher.SendEmailConfirmationArgs)
	if !ok {
		t.Fatalf("argsの型がSendEmailConfirmationArgsではありません: %T", mock.args)
	}
	if emailArgs.Email != "english@example.com" {
		t.Errorf("Email = %s、期待値 = english@example.com", emailArgs.Email)
	}
	if emailArgs.Code != confirmation.Code {
		t.Errorf("Code = %s、期待値 = %s", emailArgs.Code, confirmation.Code)
	}
	if emailArgs.Locale != "en" {
		t.Errorf("Locale = %s、期待値 = en", emailArgs.Locale)
	}
}

func TestCreateEmailConfirmationUsecase_Execute_PasswordReset(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)

	cfg := &config.Config{
		Env:    "test",
		Domain: "wikino.app",
	}

	emailConfirmationRepo := repository.NewEmailConfirmationRepository(q)
	mock := &mockJobInserter{}
	d := dispatcher.NewDispatcher(mock)
	userRepo := repository.NewUserRepository(q)
	createValidator := validator.NewEmailConfirmationCreateValidator(userRepo)
	uc := NewCreateEmailConfirmationUsecase(cfg, emailConfirmationRepo, d, createValidator)

	ctx := i18n.SetLocale(context.Background(), "ja")
	input := CreateEmailConfirmationInput{
		Email:  "reset@example.com",
		Event:  model.EmailConfirmationEventPasswordReset,
		Locale: "ja",
	}

	output, err := uc.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}

	// DBに保存されたイベント種別を確認
	confirmation, err := emailConfirmationRepo.FindByID(ctx, output.EmailConfirmationID)
	if err != nil {
		t.Fatalf("FindByID()のエラー = %v", err)
	}
	if confirmation.Event != model.EmailConfirmationEventPasswordReset {
		t.Errorf("Event = %d、期待値 = %d", confirmation.Event, model.EmailConfirmationEventPasswordReset)
	}

	// エンキューが呼ばれたことを確認
	if !mock.called {
		t.Error("Insertが呼ばれていません")
	}
}

func TestGenerateConfirmationCode(t *testing.T) {
	t.Parallel()

	t.Run("6文字の大文字英数字が生成される", func(t *testing.T) {
		t.Parallel()

		code, err := generateConfirmationCode()
		if err != nil {
			t.Fatalf("generateConfirmationCode()のエラー = %v", err)
		}

		if len(code) != 6 {
			t.Errorf("コード長 = %d、期待値 = 6", len(code))
		}

		// すべての文字が有効な文字セットに含まれていることを確認
		const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		for _, c := range code {
			if !contains(charset, string(c)) {
				t.Errorf("無効な文字: %c", c)
			}
		}
	})

	t.Run("生成されるコードはランダムである", func(t *testing.T) {
		t.Parallel()

		codes := make(map[string]bool)
		for i := 0; i < 100; i++ {
			code, err := generateConfirmationCode()
			if err != nil {
				t.Fatalf("generateConfirmationCode()のエラー = %v", err)
			}
			codes[code] = true
		}

		// 100回生成して、少なくとも90種類以上の異なるコードが生成されることを確認
		if len(codes) < 90 {
			t.Errorf("ユニークなコード数 = %d、期待値 = 90以上", len(codes))
		}
	})
}

// containsは文字列に指定した部分文字列が含まれているかを返す
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
