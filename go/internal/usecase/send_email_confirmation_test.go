package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
)

// mockEmailConfirmationSender はテスト用のモック EmailConfirmationSender
type mockEmailConfirmationSender struct {
	called bool
	to     string
	code   string
	appURL string
	locale string
	err    error
}

func (m *mockEmailConfirmationSender) Send(_ context.Context, to, code, appURL, locale string) error {
	m.called = true
	m.to = to
	m.code = code
	m.appURL = appURL
	m.locale = locale
	return m.err
}

func TestSendEmailConfirmationUsecase_Execute_Japanese(t *testing.T) {
	t.Parallel()

	sender := &mockEmailConfirmationSender{}
	uc := NewSendEmailConfirmationUsecase(sender)

	ctx := i18n.SetLocale(context.Background(), "ja")
	input := SendEmailConfirmationInput{
		Email:  "test@example.com",
		Code:   "ABC123",
		AppURL: "https://example.dev",
		Locale: "ja",
	}

	err := uc.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !sender.called {
		t.Fatal("Send() が呼ばれていません")
	}
	if sender.to != "test@example.com" {
		t.Errorf("to = %s, want test@example.com", sender.to)
	}
	if sender.code != "ABC123" {
		t.Errorf("code = %s, want ABC123", sender.code)
	}
	if sender.appURL != "https://example.dev" {
		t.Errorf("appURL = %s, want https://example.dev", sender.appURL)
	}
	if sender.locale != "ja" {
		t.Errorf("locale = %s, want ja", sender.locale)
	}
}

func TestSendEmailConfirmationUsecase_Execute_English(t *testing.T) {
	t.Parallel()

	sender := &mockEmailConfirmationSender{}
	uc := NewSendEmailConfirmationUsecase(sender)

	ctx := i18n.SetLocale(context.Background(), "en")
	input := SendEmailConfirmationInput{
		Email:  "test@example.com",
		Code:   "ABC123",
		AppURL: "https://example.dev",
		Locale: "en",
	}

	err := uc.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !sender.called {
		t.Fatal("Send() が呼ばれていません")
	}
	if sender.locale != "en" {
		t.Errorf("locale = %s, want en", sender.locale)
	}
}

func TestSendEmailConfirmationUsecase_Execute_EmptyEmail(t *testing.T) {
	t.Parallel()

	sender := &mockEmailConfirmationSender{}
	uc := NewSendEmailConfirmationUsecase(sender)

	ctx := context.Background()
	input := SendEmailConfirmationInput{
		Email:  "",
		Code:   "ABC123",
		AppURL: "https://example.dev",
		Locale: "ja",
	}

	err := uc.Execute(ctx, input)
	if err == nil {
		t.Fatal("Execute() error = nil, want error")
	}
	if sender.called {
		t.Error("Send() が呼ばれるべきではありません")
	}
}

func TestSendEmailConfirmationUsecase_Execute_SendError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("メール送信エラー")
	sender := &mockEmailConfirmationSender{err: expectedErr}
	uc := NewSendEmailConfirmationUsecase(sender)

	ctx := i18n.SetLocale(context.Background(), "ja")
	input := SendEmailConfirmationInput{
		Email:  "test@example.com",
		Code:   "ABC123",
		AppURL: "https://example.dev",
		Locale: "ja",
	}

	err := uc.Execute(ctx, input)
	if err == nil {
		t.Fatal("Execute() error = nil, want error")
	}
	if !sender.called {
		t.Error("Send() が呼ばれていません")
	}
}
