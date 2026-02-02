package account_test

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/account"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// generateUniqueCode はユニークな確認コードを生成します
func generateUniqueCode() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, 6)
	for i := range code {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		code[i] = charset[n.Int64()]
	}
	return string(code)
}

func TestCreate_Success(t *testing.T) {
	t.Parallel()

	// トランザクションなしでテスト用DBをセットアップ（Usecaseが独自のトランザクションを管理するため）
	db := testutil.SetupTestDBWithoutTx(t)
	queries := query.New(db)

	// 設定を作成
	cfg := &config.Config{
		Env:             "test",
		Port:            "8080",
		Domain:          "localhost",
		CookieDomain:    "",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	// リポジトリを初期化
	userRepo := repository.NewUserRepository(queries)
	userPasswordRepo := repository.NewUserPasswordRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)

	// ユースケースを初期化
	createAccountUC := usecase.NewCreateAccountUsecase(
		db,
		emailConfirmationRepo,
		userRepo,
		userPasswordRepo,
	)
	createUserSessionUC := usecase.NewCreateUserSessionUsecase(userSessionRepo)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// ユニークなテストデータを生成
	testID := time.Now().UnixNano()
	testEmail := fmt.Sprintf("create_success_%d@example.com", testID)
	testAtname := fmt.Sprintf("cs%d", testID%1000000000000)

	// 確認済みのメール確認情報を作成
	now := time.Now()
	emailConfirmation, err := emailConfirmationRepo.Create(t.Context(), repository.CreateEmailConfirmationInput{
		Email:     testEmail,
		Event:     model.EmailConfirmationEventSignUp,
		Code:      generateUniqueCode(),
		StartedAt: now,
	})
	if err != nil {
		t.Fatalf("failed to create email confirmation: %v", err)
	}

	// メール確認を完了状態に更新
	err = emailConfirmationRepo.Succeed(t.Context(), emailConfirmation.ID)
	if err != nil {
		t.Fatalf("failed to succeed email confirmation: %v", err)
	}

	handler := account.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		emailConfirmationRepo,
		createAccountUC,
		createUserSessionUC,
	)

	// フォームデータを作成
	form := url.Values{}
	form.Set("atname", testAtname)
	form.Set("password", "password123")

	// HTTPリクエストを作成
	req := httptest.NewRequest(http.MethodPost, "/accounts", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept-Language", "ja")

	// CSRFトークンをコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	// email_confirmation_id を Cookie に設定
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: emailConfirmation.ID,
	})

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	// リダイレクトのステータスコードを検証
	if rr.Code != http.StatusFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}

	// リダイレクト先を検証
	location := rr.Header().Get("Location")
	if location != "/home" {
		t.Errorf("wrong redirect location: got %v want %v", location, "/home")
	}

	// セッションCookieが設定されているか確認
	cookies := rr.Result().Cookies()
	var hasSessionCookie bool
	for _, cookie := range cookies {
		if cookie.Name == session.CookieName {
			hasSessionCookie = true
			if cookie.Value == "" {
				t.Error("session cookie value is empty")
			}
		}
	}
	if !hasSessionCookie {
		t.Error("session cookie not set")
	}

	// ユーザーが作成されているか確認
	user, err := userRepo.FindByAtname(t.Context(), testAtname)
	if err != nil {
		t.Fatalf("failed to find user: %v", err)
	}
	if user == nil {
		t.Error("user not created")
	}
	if user != nil && user.Email != testEmail {
		t.Errorf("wrong email: got %v want %v", user.Email, testEmail)
	}
}

func TestCreate_ValidationError_AtnameRequired(t *testing.T) {
	t.Parallel()

	// トランザクションなしでテスト用DBをセットアップ
	db := testutil.SetupTestDBWithoutTx(t)
	queries := query.New(db)

	// 設定を作成
	cfg := &config.Config{
		Env:             "test",
		Port:            "8080",
		Domain:          "localhost",
		CookieDomain:    "",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	// リポジトリを初期化
	userRepo := repository.NewUserRepository(queries)
	userPasswordRepo := repository.NewUserPasswordRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)

	// ユースケースを初期化
	createAccountUC := usecase.NewCreateAccountUsecase(
		db,
		emailConfirmationRepo,
		userRepo,
		userPasswordRepo,
	)
	createUserSessionUC := usecase.NewCreateUserSessionUsecase(userSessionRepo)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// ユニークなテストデータを生成
	testID := time.Now().UnixNano()
	testEmail := fmt.Sprintf("atname_required_%d@example.com", testID)

	// 確認済みのメール確認情報を作成
	now := time.Now()
	emailConfirmation, err := emailConfirmationRepo.Create(t.Context(), repository.CreateEmailConfirmationInput{
		Email:     testEmail,
		Event:     model.EmailConfirmationEventSignUp,
		Code:      generateUniqueCode(),
		StartedAt: now,
	})
	if err != nil {
		t.Fatalf("failed to create email confirmation: %v", err)
	}

	// メール確認を完了状態に更新
	err = emailConfirmationRepo.Succeed(t.Context(), emailConfirmation.ID)
	if err != nil {
		t.Fatalf("failed to succeed email confirmation: %v", err)
	}

	handler := account.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		emailConfirmationRepo,
		createAccountUC,
		createUserSessionUC,
	)

	// フォームデータを作成（アットネーム空）
	form := url.Values{}
	form.Set("atname", "")
	form.Set("password", "password123")

	// HTTPリクエストを作成
	req := httptest.NewRequest(http.MethodPost, "/accounts", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept-Language", "ja")

	// CSRFトークンと言語設定をコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, i18n.LangJa)
	req = req.WithContext(ctx)

	// email_confirmation_id を Cookie に設定
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: emailConfirmation.ID,
	})

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	// バリデーションエラーのステータスコードを検証
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	// エラーメッセージが含まれているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "アットネームを入力してください") {
		t.Error("atname required error not found in response")
	}
}

func TestCreate_ValidationError_PasswordTooShort(t *testing.T) {
	t.Parallel()

	// トランザクションなしでテスト用DBをセットアップ
	db := testutil.SetupTestDBWithoutTx(t)
	queries := query.New(db)

	// 設定を作成
	cfg := &config.Config{
		Env:             "test",
		Port:            "8080",
		Domain:          "localhost",
		CookieDomain:    "",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	// リポジトリを初期化
	userRepo := repository.NewUserRepository(queries)
	userPasswordRepo := repository.NewUserPasswordRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)

	// ユースケースを初期化
	createAccountUC := usecase.NewCreateAccountUsecase(
		db,
		emailConfirmationRepo,
		userRepo,
		userPasswordRepo,
	)
	createUserSessionUC := usecase.NewCreateUserSessionUsecase(userSessionRepo)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// ユニークなテストデータを生成
	testID := time.Now().UnixNano()
	testEmail := fmt.Sprintf("pw_short_%d@example.com", testID)

	// 確認済みのメール確認情報を作成
	now := time.Now()
	emailConfirmation, err := emailConfirmationRepo.Create(t.Context(), repository.CreateEmailConfirmationInput{
		Email:     testEmail,
		Event:     model.EmailConfirmationEventSignUp,
		Code:      generateUniqueCode(),
		StartedAt: now,
	})
	if err != nil {
		t.Fatalf("failed to create email confirmation: %v", err)
	}

	// メール確認を完了状態に更新
	err = emailConfirmationRepo.Succeed(t.Context(), emailConfirmation.ID)
	if err != nil {
		t.Fatalf("failed to succeed email confirmation: %v", err)
	}

	handler := account.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		emailConfirmationRepo,
		createAccountUC,
		createUserSessionUC,
	)

	// フォームデータを作成（パスワードが短い）
	form := url.Values{}
	form.Set("atname", "testuser")
	form.Set("password", "short")

	// HTTPリクエストを作成
	req := httptest.NewRequest(http.MethodPost, "/accounts", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept-Language", "ja")

	// CSRFトークンと言語設定をコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, i18n.LangJa)
	req = req.WithContext(ctx)

	// email_confirmation_id を Cookie に設定
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: emailConfirmation.ID,
	})

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	// バリデーションエラーのステータスコードを検証
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	// エラーメッセージが含まれているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "パスワードは8文字以上で入力してください") {
		t.Error("password too short error not found in response")
	}
}

func TestCreate_ValidationError_AtnameInvalidFormat(t *testing.T) {
	t.Parallel()

	// トランザクションなしでテスト用DBをセットアップ
	db := testutil.SetupTestDBWithoutTx(t)
	queries := query.New(db)

	// 設定を作成
	cfg := &config.Config{
		Env:             "test",
		Port:            "8080",
		Domain:          "localhost",
		CookieDomain:    "",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	// リポジトリを初期化
	userRepo := repository.NewUserRepository(queries)
	userPasswordRepo := repository.NewUserPasswordRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)

	// ユースケースを初期化
	createAccountUC := usecase.NewCreateAccountUsecase(
		db,
		emailConfirmationRepo,
		userRepo,
		userPasswordRepo,
	)
	createUserSessionUC := usecase.NewCreateUserSessionUsecase(userSessionRepo)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// ユニークなテストデータを生成
	testID := time.Now().UnixNano()
	testEmail := fmt.Sprintf("atname_invalid_%d@example.com", testID)

	// 確認済みのメール確認情報を作成
	now := time.Now()
	emailConfirmation, err := emailConfirmationRepo.Create(t.Context(), repository.CreateEmailConfirmationInput{
		Email:     testEmail,
		Event:     model.EmailConfirmationEventSignUp,
		Code:      generateUniqueCode(),
		StartedAt: now,
	})
	if err != nil {
		t.Fatalf("failed to create email confirmation: %v", err)
	}

	// メール確認を完了状態に更新
	err = emailConfirmationRepo.Succeed(t.Context(), emailConfirmation.ID)
	if err != nil {
		t.Fatalf("failed to succeed email confirmation: %v", err)
	}

	handler := account.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		emailConfirmationRepo,
		createAccountUC,
		createUserSessionUC,
	)

	// フォームデータを作成（アットネームに無効な文字）
	form := url.Values{}
	form.Set("atname", "test-user!@")
	form.Set("password", "password123")

	// HTTPリクエストを作成
	req := httptest.NewRequest(http.MethodPost, "/accounts", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept-Language", "ja")

	// CSRFトークンと言語設定をコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, i18n.LangJa)
	req = req.WithContext(ctx)

	// email_confirmation_id を Cookie に設定
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: emailConfirmation.ID,
	})

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	// バリデーションエラーのステータスコードを検証
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	// エラーメッセージが含まれているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "アットネームは英数字とアンダースコアのみ使用できます") {
		t.Error("atname invalid format error not found in response")
	}
}

func TestCreate_AtnameAlreadyTaken(t *testing.T) {
	t.Parallel()

	// トランザクションなしでテスト用DBをセットアップ
	db := testutil.SetupTestDBWithoutTx(t)
	queries := query.New(db)

	// 設定を作成
	cfg := &config.Config{
		Env:             "test",
		Port:            "8080",
		Domain:          "localhost",
		CookieDomain:    "",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	// リポジトリを初期化
	userRepo := repository.NewUserRepository(queries)
	userPasswordRepo := repository.NewUserPasswordRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)

	// ユニークなテストデータを生成
	testID := time.Now().UnixNano()
	existingAtname := fmt.Sprintf("ex%d", testID%1000000000000)
	testEmail := fmt.Sprintf("atname_taken_%d@example.com", testID)

	// 既存ユーザーを作成
	_, err := userRepo.Create(t.Context(), repository.CreateUserInput{
		Email:       fmt.Sprintf("existing_%d@example.com", testID),
		Atname:      existingAtname,
		Name:        "",
		Description: "",
		Locale:      model.LocaleJa,
		TimeZone:    "Asia/Tokyo",
		JoinedAt:    time.Now(),
	})
	if err != nil {
		t.Fatalf("failed to create existing user: %v", err)
	}

	// ユースケースを初期化
	createAccountUC := usecase.NewCreateAccountUsecase(
		db,
		emailConfirmationRepo,
		userRepo,
		userPasswordRepo,
	)
	createUserSessionUC := usecase.NewCreateUserSessionUsecase(userSessionRepo)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// 確認済みのメール確認情報を作成
	now := time.Now()
	emailConfirmation, err := emailConfirmationRepo.Create(t.Context(), repository.CreateEmailConfirmationInput{
		Email:     testEmail,
		Event:     model.EmailConfirmationEventSignUp,
		Code:      generateUniqueCode(),
		StartedAt: now,
	})
	if err != nil {
		t.Fatalf("failed to create email confirmation: %v", err)
	}

	// メール確認を完了状態に更新
	err = emailConfirmationRepo.Succeed(t.Context(), emailConfirmation.ID)
	if err != nil {
		t.Fatalf("failed to succeed email confirmation: %v", err)
	}

	handler := account.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		emailConfirmationRepo,
		createAccountUC,
		createUserSessionUC,
	)

	// フォームデータを作成（既存のアットネームを使用）
	form := url.Values{}
	form.Set("atname", existingAtname)
	form.Set("password", "password123")

	// HTTPリクエストを作成
	req := httptest.NewRequest(http.MethodPost, "/accounts", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept-Language", "ja")

	// CSRFトークンと言語設定をコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = i18n.SetLocale(ctx, i18n.LangJa)
	req = req.WithContext(ctx)

	// email_confirmation_id を Cookie に設定
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: emailConfirmation.ID,
	})

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	// バリデーションエラーのステータスコードを検証
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusUnprocessableEntity)
	}

	// エラーメッセージが含まれているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "このアットネームは既に使用されています") {
		t.Error("atname already taken error not found in response")
	}
}

func TestCreate_NoEmailConfirmationID(t *testing.T) {
	t.Parallel()

	// トランザクションなしでテスト用DBをセットアップ
	db := testutil.SetupTestDBWithoutTx(t)
	queries := query.New(db)

	// 設定を作成
	cfg := &config.Config{
		Env:             "test",
		Port:            "8080",
		Domain:          "localhost",
		CookieDomain:    "",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	// リポジトリを初期化
	userRepo := repository.NewUserRepository(queries)
	userPasswordRepo := repository.NewUserPasswordRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)

	// ユースケースを初期化
	createAccountUC := usecase.NewCreateAccountUsecase(
		db,
		emailConfirmationRepo,
		userRepo,
		userPasswordRepo,
	)
	createUserSessionUC := usecase.NewCreateUserSessionUsecase(userSessionRepo)

	// セッションマネージャーを初期化
	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	handler := account.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		emailConfirmationRepo,
		createAccountUC,
		createUserSessionUC,
	)

	// フォームデータを作成
	form := url.Values{}
	form.Set("atname", "testuser")
	form.Set("password", "password123")

	// HTTPリクエストを作成（email_confirmation_id のCookieなし）
	req := httptest.NewRequest(http.MethodPost, "/accounts", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept-Language", "ja")

	// CSRFトークンをコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	// リダイレクトのステータスコードを検証
	if rr.Code != http.StatusFound {
		t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusFound)
	}

	// リダイレクト先を検証
	location := rr.Header().Get("Location")
	if location != "/sign_up" {
		t.Errorf("wrong redirect location: got %v want %v", location, "/sign_up")
	}
}
