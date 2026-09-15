package account_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
)

func createUnconfirmedEmailConfirmation(t *testing.T, emailConfirmationRepo *repository.EmailConfirmationRepository, email string) string {
	t.Helper()

	now := time.Now()
	emailConfirmation, err := emailConfirmationRepo.Create(t.Context(), repository.CreateEmailConfirmationInput{
		Email:     email,
		Event:     model.EmailConfirmationEventSignUp,
		Code:      generateUniqueCode(),
		StartedAt: now,
	})
	if err != nil {
		t.Fatalf("メールアドレス確認の作成に失敗: %v", err)
	}

	return emailConfirmation.ID
}

func TestNew(t *testing.T) {
	t.Parallel()

	handler, _, emailConfirmationRepo := setupHandler(t)

	testID := time.Now().UnixNano()
	testEmail := fmt.Sprintf("new_success_%d@example.com", testID)

	ecID := createConfirmedEmailConfirmation(t, emailConfirmationRepo, testEmail)

	// HTTPリクエストを作成
	req := httptest.NewRequest(http.MethodGet, "/accounts/new", nil)
	req.Header.Set("Accept-Language", "ja")

	// CSRFトークンをコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	// email_confirmation_idをCookieに設定
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: ecID,
	})

	rr := httptest.NewRecorder()
	handler.New(rr, req)

	// ステータスコードを検証
	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	// レスポンスボディを検証
	body := rr.Body.String()

	// フォームアクションが含まれているか確認
	if !strings.Contains(body, `action="/accounts"`) {
		t.Error("レスポンスにアカウント作成フォームの送信先が見つからない")
	}

	// CSRFトークンが含まれているか確認
	if !strings.Contains(body, "test-csrf-token") {
		t.Error("レスポンスにCSRFトークンが見つからない")
	}

	// メールアドレスが表示されているか確認
	if !strings.Contains(body, testEmail) {
		t.Error("レスポンスにメールアドレスが見つからない")
	}

	// アットネーム入力フィールドが含まれているか確認
	if !strings.Contains(body, `name="atname"`) {
		t.Error("レスポンスにatnameの入力フィールドが見つからない")
	}

	// パスワード入力フィールドが含まれているか確認
	if !strings.Contains(body, `name="password"`) {
		t.Error("レスポンスにパスワードの入力フィールドが見つからない")
	}

	// 日本語の見出しが含まれているか確認
	if !strings.Contains(body, "アカウントを作成") {
		t.Error("レスポンスに日本語の見出しが見つからない")
	}

	// 新規登録に戻るリンクが含まれているか確認
	if !strings.Contains(body, `href="/sign_up"`) {
		t.Error("レスポンスにサインアップへ戻るリンクが見つからない")
	}

	for _, notWant := range []string{`<link rel="canonical"`, `property="og:url"`} {
		if strings.Contains(body, notWant) {
			t.Errorf("レスポンスに想定外の%qが含まれている", notWant)
		}
	}
}

func TestNew_NoEmailConfirmationID(t *testing.T) {
	t.Parallel()

	handler, _, _ := setupHandler(t)

	// HTTPリクエストを作成 (email_confirmation_idのCookieなし)
	req := httptest.NewRequest(http.MethodGet, "/accounts/new", nil)
	req.Header.Set("Accept-Language", "ja")

	// CSRFトークンをコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.New(rr, req)

	// リダイレクトのステータスコードを検証
	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusFound)
	}

	// リダイレクト先を検証
	location := rr.Header().Get("Location")
	if location != "/sign_up" {
		t.Errorf("リダイレクト先 = %v、期待値 = %v", location, "/sign_up")
	}
}

func TestNew_EmailConfirmationNotFound(t *testing.T) {
	t.Parallel()

	handler, _, _ := setupHandler(t)

	// HTTPリクエストを作成 (存在しないemail_confirmation_id)
	req := httptest.NewRequest(http.MethodGet, "/accounts/new", nil)
	req.Header.Set("Accept-Language", "ja")

	// CSRFトークンをコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	// 存在しないIDをCookieに設定 (有効なUUID形式だが存在しない)
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: "00000000-0000-0000-0000-000000000000",
	})

	rr := httptest.NewRecorder()
	handler.New(rr, req)

	// リダイレクトのステータスコードを検証
	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusFound)
	}

	// リダイレクト先を検証
	location := rr.Header().Get("Location")
	if location != "/sign_up" {
		t.Errorf("リダイレクト先 = %v、期待値 = %v", location, "/sign_up")
	}
}

func TestNew_EmailNotVerified(t *testing.T) {
	t.Parallel()

	handler, _, emailConfirmationRepo := setupHandler(t)

	testID := time.Now().UnixNano()
	testEmail := fmt.Sprintf("new_unverified_%d@example.com", testID)

	ecID := createUnconfirmedEmailConfirmation(t, emailConfirmationRepo, testEmail)

	// HTTPリクエストを作成
	req := httptest.NewRequest(http.MethodGet, "/accounts/new", nil)
	req.Header.Set("Accept-Language", "ja")

	// CSRFトークンをコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	// email_confirmation_idをCookieに設定
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: ecID,
	})

	rr := httptest.NewRecorder()
	handler.New(rr, req)

	// リダイレクトのステータスコードを検証
	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusFound)
	}

	// リダイレクト先を検証 (確認コード入力ページへ)
	location := rr.Header().Get("Location")
	if location != "/email_confirmation/edit" {
		t.Errorf("リダイレクト先 = %v、期待値 = %v", location, "/email_confirmation/edit")
	}
}

func TestNew_EnglishLocale(t *testing.T) {
	t.Parallel()

	handler, _, emailConfirmationRepo := setupHandler(t)

	testID := time.Now().UnixNano()
	testEmail := fmt.Sprintf("new_english_%d@example.com", testID)

	ecID := createConfirmedEmailConfirmation(t, emailConfirmationRepo, testEmail)

	// HTTPリクエストを作成 (英語ロケール)
	req := httptest.NewRequest(http.MethodGet, "/accounts/new", nil)
	req.Header.Set("Accept-Language", "en")

	// CSRFトークンと言語設定をコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, i18n.LangEn)
	req = req.WithContext(ctx)

	// email_confirmation_idをCookieに設定
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: ecID,
	})

	rr := httptest.NewRecorder()
	handler.New(rr, req)

	// ステータスコードを検証
	if rr.Code != http.StatusOK {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusOK)
	}

	// 英語の見出しが含まれているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "Create your account") {
		t.Error("レスポンスに英語の見出しが見つからない")
	}

	// 英語のボタンテキストが含まれているか確認
	if !strings.Contains(body, "Create account") {
		t.Error("レスポンスに英語の送信ボタンの文言が見つからない")
	}

	// 英語のパスワードヒントが含まれているか確認
	if !strings.Contains(body, "Must be at least 8 characters") {
		t.Error("レスポンスに英語のパスワードのヒントが見つからない")
	}

	// 英語の戻るリンクが含まれているか確認
	if !strings.Contains(body, "Back to sign up") {
		t.Error("レスポンスに英語の戻るリンクが見つからない")
	}
}
