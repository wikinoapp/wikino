package usecase

import (
	"context"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/worker"
)

// mockEnqueuer はテスト用のモック enqueuer
type mockEnqueuer struct {
	called bool
	input  worker.EnqueueEmailConfirmationInput
}

func (m *mockEnqueuer) EnqueueEmailConfirmation(_ context.Context, input worker.EnqueueEmailConfirmationInput) error {
	m.called = true
	m.input = input
	return nil
}

func TestSendEmailConfirmationUsecase_Execute_Japanese(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	q := testutil.QueriesWithTx(tx)

	cfg := &config.Config{
		Env:    "test",
		Domain: "wikino.app",
	}

	emailConfirmationRepo := repository.NewEmailConfirmationRepository(q)
	enqueuer := &mockEnqueuer{}
	uc := NewSendEmailConfirmationUsecase(cfg, emailConfirmationRepo, enqueuer)

	ctx := i18n.SetLocale(context.Background(), "ja")
	input := SendEmailConfirmationInput{
		Email:  "test@example.com",
		Event:  model.EmailConfirmationEventSignUp,
		Locale: "ja",
	}

	output, err := uc.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if output.EmailConfirmationID == "" {
		t.Error("EmailConfirmationID が空です")
	}

	// DBに保存されたことを確認
	confirmation, err := emailConfirmationRepo.FindByID(ctx, output.EmailConfirmationID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if confirmation == nil {
		t.Fatal("メール確認情報がDBに保存されていません")
	}
	if confirmation.Email != "test@example.com" {
		t.Errorf("Email = %s, want test@example.com", confirmation.Email)
	}
	if confirmation.Event != model.EmailConfirmationEventSignUp {
		t.Errorf("Event = %d, want %d", confirmation.Event, model.EmailConfirmationEventSignUp)
	}
	if len(confirmation.Code) != 6 {
		t.Errorf("Code length = %d, want 6", len(confirmation.Code))
	}

	// エンキューが呼ばれたことを確認
	if !enqueuer.called {
		t.Error("EnqueueEmailConfirmation が呼ばれていません")
	}
	if enqueuer.input.Email != "test@example.com" {
		t.Errorf("enqueued Email = %s, want test@example.com", enqueuer.input.Email)
	}
	if enqueuer.input.Locale != "ja" {
		t.Errorf("enqueued Locale = %s, want ja", enqueuer.input.Locale)
	}
}

func TestSendEmailConfirmationUsecase_Execute_English(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	q := testutil.QueriesWithTx(tx)

	cfg := &config.Config{
		Env:    "test",
		Domain: "wikino.app",
	}

	emailConfirmationRepo := repository.NewEmailConfirmationRepository(q)
	enqueuer := &mockEnqueuer{}
	uc := NewSendEmailConfirmationUsecase(cfg, emailConfirmationRepo, enqueuer)

	ctx := i18n.SetLocale(context.Background(), "en")
	input := SendEmailConfirmationInput{
		Email:  "english@example.com",
		Event:  model.EmailConfirmationEventSignUp,
		Locale: "en",
	}

	output, err := uc.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if output.EmailConfirmationID == "" {
		t.Error("EmailConfirmationID が空です")
	}

	// DBに保存されたことを確認
	confirmation, err := emailConfirmationRepo.FindByID(ctx, output.EmailConfirmationID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if confirmation == nil {
		t.Fatal("メール確認情報がDBに保存されていません")
	}
	if confirmation.Email != "english@example.com" {
		t.Errorf("Email = %s, want english@example.com", confirmation.Email)
	}

	// エンキューが呼ばれたことを確認
	if !enqueuer.called {
		t.Error("EnqueueEmailConfirmation が呼ばれていません")
	}
	if enqueuer.input.Locale != "en" {
		t.Errorf("enqueued Locale = %s, want en", enqueuer.input.Locale)
	}
}

func TestSendEmailConfirmationUsecase_Execute_PasswordReset(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTestDB(t)
	q := testutil.QueriesWithTx(tx)

	cfg := &config.Config{
		Env:    "test",
		Domain: "wikino.app",
	}

	emailConfirmationRepo := repository.NewEmailConfirmationRepository(q)
	enqueuer := &mockEnqueuer{}
	uc := NewSendEmailConfirmationUsecase(cfg, emailConfirmationRepo, enqueuer)

	ctx := i18n.SetLocale(context.Background(), "ja")
	input := SendEmailConfirmationInput{
		Email:  "reset@example.com",
		Event:  model.EmailConfirmationEventPasswordReset,
		Locale: "ja",
	}

	output, err := uc.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// DBに保存されたイベント種別を確認
	confirmation, err := emailConfirmationRepo.FindByID(ctx, output.EmailConfirmationID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if confirmation.Event != model.EmailConfirmationEventPasswordReset {
		t.Errorf("Event = %d, want %d", confirmation.Event, model.EmailConfirmationEventPasswordReset)
	}

	// エンキューが呼ばれたことを確認
	if !enqueuer.called {
		t.Error("EnqueueEmailConfirmation が呼ばれていません")
	}
}

func TestGenerateConfirmationCode(t *testing.T) {
	t.Parallel()

	t.Run("6文字の大文字英数字が生成される", func(t *testing.T) {
		t.Parallel()

		code, err := generateConfirmationCode()
		if err != nil {
			t.Fatalf("generateConfirmationCode() error = %v", err)
		}

		if len(code) != 6 {
			t.Errorf("コード長 = %d, want 6", len(code))
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
				t.Fatalf("generateConfirmationCode() error = %v", err)
			}
			codes[code] = true
		}

		// 100回生成して、少なくとも90種類以上の異なるコードが生成されることを確認
		if len(codes) < 90 {
			t.Errorf("ユニークなコード数 = %d, want >= 90", len(codes))
		}
	})
}

// contains は文字列に指定した部分文字列が含まれているかを返す
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
