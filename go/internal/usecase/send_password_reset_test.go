package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
)

// mockPasswordResetSenderはテスト用のモックPasswordResetSender
type mockPasswordResetSender struct {
	called   bool
	to       string
	resetURL string
	appURL   string
	locale   string
	err      error
}

func (m *mockPasswordResetSender) Send(_ context.Context, to, resetURL, appURL, locale string) error {
	m.called = true
	m.to = to
	m.resetURL = resetURL
	m.appURL = appURL
	m.locale = locale
	return m.err
}

func TestSendPasswordResetUsecase_Execute_Japanese(t *testing.T) {
	t.Parallel()

	sender := &mockPasswordResetSender{}
	uc := NewSendPasswordResetUsecase(sender)

	ctx := i18n.SetLocale(context.Background(), "ja")
	input := SendPasswordResetInput{
		Email:    "test@example.com",
		ResetURL: "https://example.dev/password/edit?token=abc123",
		AppURL:   "https://example.dev",
		Locale:   "ja",
	}

	err := uc.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}

	if !sender.called {
		t.Fatal("Send() が呼ばれていません")
	}
	if sender.to != "test@example.com" {
		t.Errorf("to = %s、期待値 = test@example.com", sender.to)
	}
	if sender.resetURL != "https://example.dev/password/edit?token=abc123" {
		t.Errorf("resetURL = %s、期待値 = https://example.dev/password/edit?token=abc123", sender.resetURL)
	}
	if sender.appURL != "https://example.dev" {
		t.Errorf("appURL = %s、期待値 = https://example.dev", sender.appURL)
	}
	if sender.locale != "ja" {
		t.Errorf("locale = %s、期待値 = ja", sender.locale)
	}
}

func TestSendPasswordResetUsecase_Execute_English(t *testing.T) {
	t.Parallel()

	sender := &mockPasswordResetSender{}
	uc := NewSendPasswordResetUsecase(sender)

	ctx := i18n.SetLocale(context.Background(), "en")
	input := SendPasswordResetInput{
		Email:    "test@example.com",
		ResetURL: "https://example.dev/password/edit?token=abc123",
		AppURL:   "https://example.dev",
		Locale:   "en",
	}

	err := uc.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}

	if !sender.called {
		t.Fatal("Send() が呼ばれていません")
	}
	if sender.locale != "en" {
		t.Errorf("locale = %s、期待値 = en", sender.locale)
	}
}

func TestSendPasswordResetUsecase_Execute_EmptyEmail(t *testing.T) {
	t.Parallel()

	sender := &mockPasswordResetSender{}
	uc := NewSendPasswordResetUsecase(sender)

	ctx := context.Background()
	input := SendPasswordResetInput{
		Email:    "",
		ResetURL: "https://example.dev/password/edit?token=abc123",
		AppURL:   "https://example.dev",
		Locale:   "ja",
	}

	err := uc.Execute(ctx, input)
	if err == nil {
		t.Fatal("Execute()のエラー = nil、期待値 = エラー")
	}
	if sender.called {
		t.Error("Send()が呼ばれた")
	}
}

func TestSendPasswordResetUsecase_Execute_SendError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("メール送信エラー")
	sender := &mockPasswordResetSender{err: expectedErr}
	uc := NewSendPasswordResetUsecase(sender)

	ctx := i18n.SetLocale(context.Background(), "ja")
	input := SendPasswordResetInput{
		Email:    "test@example.com",
		ResetURL: "https://example.dev/password/edit?token=abc123",
		AppURL:   "https://example.dev",
		Locale:   "ja",
	}

	err := uc.Execute(ctx, input)
	if err == nil {
		t.Fatal("Execute()のエラー = nil、期待値 = エラー")
	}
	if !sender.called {
		t.Error("Send() が呼ばれていません")
	}
}
