package usecase

import (
	"context"
	"sync"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/email"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// mockEmailSender はテスト用のメール送信モック
type mockEmailSender struct {
	mu       sync.Mutex
	sentMail []email.SendInput
	err      error
}

func (m *mockEmailSender) Send(_ context.Context, input email.SendInput) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.sentMail = append(m.sentMail, input)
	return nil
}

func (m *mockEmailSender) getSentMail() []email.SendInput {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sentMail
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
	mockSender := &mockEmailSender{}
	uc := NewSendEmailConfirmationUsecase(cfg, emailConfirmationRepo, mockSender)

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

	// メールが送信されたことを確認
	sentMail := mockSender.getSentMail()
	if len(sentMail) != 1 {
		t.Errorf("送信されたメール数 = %d, want 1", len(sentMail))
	}

	if sentMail[0].To != "test@example.com" {
		t.Errorf("送信先 = %s, want test@example.com", sentMail[0].To)
	}

	if sentMail[0].Subject != "[Wikino] 確認用コード" {
		t.Errorf("件名 = %s, want [Wikino] 確認用コード", sentMail[0].Subject)
	}

	// メール本文に日本語のテキストが含まれていることを確認
	if !contains(sentMail[0].HTML, "こんにちは") {
		t.Error("メール本文に日本語の挨拶が含まれていません")
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
	mockSender := &mockEmailSender{}
	uc := NewSendEmailConfirmationUsecase(cfg, emailConfirmationRepo, mockSender)

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

	// メールが送信されたことを確認
	sentMail := mockSender.getSentMail()
	if len(sentMail) != 1 {
		t.Errorf("送信されたメール数 = %d, want 1", len(sentMail))
	}

	if sentMail[0].Subject != "[Wikino] Confirmation Code" {
		t.Errorf("件名 = %s, want [Wikino] Confirmation Code", sentMail[0].Subject)
	}

	// メール本文に英語のテキストが含まれていることを確認
	if !contains(sentMail[0].HTML, "Hello") {
		t.Error("メール本文に英語の挨拶が含まれていません")
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
	mockSender := &mockEmailSender{}
	uc := NewSendEmailConfirmationUsecase(cfg, emailConfirmationRepo, mockSender)

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
