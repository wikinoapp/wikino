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
	"github.com/wikinoapp/wikino/go/internal/timezone"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// generateUniqueCodeはユニークな確認コードを生成します
func generateUniqueCode() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, 6)
	for i := range code {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		code[i] = charset[n.Int64()]
	}
	return string(code)
}

func setupHandler(t *testing.T) (*account.Handler, *repository.UserRepository, *repository.EmailConfirmationRepository) {
	t.Helper()

	db := testutil.GetTestDB()
	queries := query.New(db)

	cfg := &config.Config{
		Env:             "test",
		Port:            "8080",
		Domain:          "localhost",
		CookieDomain:    "",
		SessionSecure:   false,
		SessionHTTPOnly: true,
	}

	userRepo := repository.NewUserRepository(queries)
	userPasswordRepo := repository.NewUserPasswordRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)

	accountCreateValidator := validator.NewAccountCreateValidator(userRepo)
	createAccountUC := usecase.NewCreateAccountUsecase(db, emailConfirmationRepo, userRepo, userPasswordRepo, accountCreateValidator)
	createUserSessionUC := usecase.NewCreateUserSessionUsecase(userSessionRepo)

	sessionMgr := session.NewManager(userRepo, userSessionRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	getAccountNewDataUC := usecase.NewGetAccountNewDataUsecase(emailConfirmationRepo)

	handler := account.NewHandler(
		cfg,
		sessionMgr,
		flashMgr,
		getAccountNewDataUC,
		createAccountUC,
		createUserSessionUC,
	)

	return handler, userRepo, emailConfirmationRepo
}

func createConfirmedEmailConfirmation(t *testing.T, emailConfirmationRepo *repository.EmailConfirmationRepository, email string) string {
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

	err = emailConfirmationRepo.Succeed(t.Context(), emailConfirmation.ID)
	if err != nil {
		t.Fatalf("メールアドレス確認の成功処理に失敗: %v", err)
	}

	return emailConfirmation.ID
}

func TestCreate_Success(t *testing.T) {
	t.Parallel()

	handler, userRepo, emailConfirmationRepo := setupHandler(t)

	testID := time.Now().UnixNano()
	testEmail := fmt.Sprintf("create_success_%d@example.com", testID)
	testAtname := fmt.Sprintf("cs%d", testID%1000000000000)

	ecID := createConfirmedEmailConfirmation(t, emailConfirmationRepo, testEmail)

	// フォームデータを作成
	form := url.Values{}
	form.Set("atname", testAtname)
	form.Set("password", "password123")

	// HTTPリクエストを作成
	req := httptest.NewRequest(http.MethodPost, "/accounts", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept-Language", "ja")

	// CSRFトークンとタイムゾーンをコンテキストに設定
	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = timezone.ToContext(ctx, "America/New_York")
	req = req.WithContext(ctx)

	// email_confirmation_idをCookieに設定
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: ecID,
	})

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	// リダイレクトのステータスコードを検証
	if rr.Code != http.StatusFound {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusFound)
	}

	// リダイレクト先を検証
	location := rr.Header().Get("Location")
	if location != "/home" {
		t.Errorf("リダイレクト先 = %v、期待値 = %v", location, "/home")
	}

	// セッションCookieが設定されているか確認
	cookies := rr.Result().Cookies()
	var hasSessionCookie bool
	for _, cookie := range cookies {
		if cookie.Name == session.CookieName {
			hasSessionCookie = true
			if cookie.Value == "" {
				t.Error("セッションCookieの値が空")
			}
		}
	}
	if !hasSessionCookie {
		t.Error("セッションCookieがセットされていない")
	}

	// ユーザーが作成されているか確認
	user, err := userRepo.FindByAtname(t.Context(), testAtname)
	if err != nil {
		t.Fatalf("ユーザーの取得に失敗: %v", err)
	}
	if user == nil {
		t.Error("ユーザーが作成されていない")
	}
	if user != nil && user.Email != testEmail {
		t.Errorf("メールアドレス = %v、期待値 = %v", user.Email, testEmail)
	}
	// コンテキストから取得したタイムゾーンが保存されているか確認
	if user != nil && user.TimeZone != "America/New_York" {
		t.Errorf("タイムゾーン = %v、期待値 = %v", user.TimeZone, "America/New_York")
	}
}

func TestCreate_ValidationError_AtnameRequired(t *testing.T) {
	t.Parallel()

	handler, _, emailConfirmationRepo := setupHandler(t)

	testID := time.Now().UnixNano()
	testEmail := fmt.Sprintf("atname_required_%d@example.com", testID)

	ecID := createConfirmedEmailConfirmation(t, emailConfirmationRepo, testEmail)

	// フォームデータを作成 (アットネーム空)
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

	// email_confirmation_idをCookieに設定
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: ecID,
	})

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	// バリデーションエラーのステータスコードを検証
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusUnprocessableEntity)
	}

	// エラーメッセージが含まれているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "アットネームを入力してください") {
		t.Error("レスポンスにatname必須のエラーが見つからない")
	}
}

func TestCreate_ValidationError_PasswordTooShort(t *testing.T) {
	t.Parallel()

	handler, _, emailConfirmationRepo := setupHandler(t)

	testID := time.Now().UnixNano()
	testEmail := fmt.Sprintf("pw_short_%d@example.com", testID)

	ecID := createConfirmedEmailConfirmation(t, emailConfirmationRepo, testEmail)

	// フォームデータを作成 (パスワードが短い)
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

	// email_confirmation_idをCookieに設定
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: ecID,
	})

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	// バリデーションエラーのステータスコードを検証
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusUnprocessableEntity)
	}

	// エラーメッセージが含まれているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "パスワードは8文字以上で入力してください") {
		t.Error("レスポンスにパスワードが短すぎるエラーが見つからない")
	}
}

func TestCreate_ValidationError_AtnameInvalidFormat(t *testing.T) {
	t.Parallel()

	handler, _, emailConfirmationRepo := setupHandler(t)

	testID := time.Now().UnixNano()
	testEmail := fmt.Sprintf("atname_invalid_%d@example.com", testID)

	ecID := createConfirmedEmailConfirmation(t, emailConfirmationRepo, testEmail)

	// フォームデータを作成 (アットネームに無効な文字)
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

	// email_confirmation_idをCookieに設定
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: ecID,
	})

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	// バリデーションエラーのステータスコードを検証
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusUnprocessableEntity)
	}

	// エラーメッセージが含まれているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "アットネームは英数字とアンダースコアのみ使用できます") {
		t.Error("レスポンスにatnameの形式が不正なエラーが見つからない")
	}
}

func TestCreate_AtnameAlreadyTaken(t *testing.T) {
	t.Parallel()

	handler, userRepo, emailConfirmationRepo := setupHandler(t)

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
		t.Fatalf("既存ユーザーの作成に失敗: %v", err)
	}

	ecID := createConfirmedEmailConfirmation(t, emailConfirmationRepo, testEmail)

	// フォームデータを作成 (既存のアットネームを使用)
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

	// email_confirmation_idをCookieに設定
	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: ecID,
	})

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	// バリデーションエラーのステータスコードを検証
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusUnprocessableEntity)
	}

	// エラーメッセージが含まれているか確認
	body := rr.Body.String()
	if !strings.Contains(body, "このアットネームは既に使用されています") {
		t.Error("レスポンスにatname使用済みのエラーが見つからない")
	}
}

func TestCreate_NoEmailConfirmationID(t *testing.T) {
	t.Parallel()

	handler, _, _ := setupHandler(t)

	// フォームデータを作成
	form := url.Values{}
	form.Set("atname", "testuser")
	form.Set("password", "password123")

	// HTTPリクエストを作成 (email_confirmation_idのCookieなし)
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
		t.Errorf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusFound)
	}

	// リダイレクト先を検証
	location := rr.Header().Get("Location")
	if location != "/sign_up" {
		t.Errorf("リダイレクト先 = %v、期待値 = %v", location, "/sign_up")
	}
}

// 作成されるセッションのIPが、生のr.RemoteAddrではなくinternal/clientip
// (CF-Connecting-IP優先) で解決したクライアントIPで記録されることを固定する。
// chiのRealIPミドルウェアを削除したためセッションIPはclientip.GetClientIP由来になる。
// CF-Connecting-IPと異なるX-Forwarded-Forを同時に送り、CF-Connecting-IPが勝つことを確認する。
func TestCreate_SessionIPAddress_PrioritizesCFConnectingIP(t *testing.T) {
	t.Parallel()

	handler, _, emailConfirmationRepo := setupHandler(t)

	testID := time.Now().UnixNano()
	testEmail := fmt.Sprintf("session_ip_%d@example.com", testID)
	testAtname := fmt.Sprintf("si%d", testID%1000000000000)

	ecID := createConfirmedEmailConfirmation(t, emailConfirmationRepo, testEmail)

	form := url.Values{}
	form.Set("atname", testAtname)
	form.Set("password", "password123")

	req := httptest.NewRequest(http.MethodPost, "/accounts", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept-Language", "ja")
	req.Header.Set("CF-Connecting-IP", "203.0.113.7")
	req.Header.Set("X-Forwarded-For", "198.51.100.9")

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	ctx = timezone.ToContext(ctx, "America/New_York")
	req = req.WithContext(ctx)

	req.AddCookie(&http.Cookie{
		Name:  session.EmailConfirmationCookieName,
		Value: ecID,
	})

	rr := httptest.NewRecorder()
	handler.Create(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("ステータスコード = %v、期待値 = %v", rr.Code, http.StatusFound)
	}

	// レスポンスのCookieからセッショントークンを取り出す。
	var sessionToken string
	for _, cookie := range rr.Result().Cookies() {
		if cookie.Name == session.CookieName {
			sessionToken = cookie.Value
			break
		}
	}
	if sessionToken == "" {
		t.Fatal("セッションCookieがセットされていない")
	}

	// 永続化されたセッションを読み戻し、記録されたIPを検証する。
	userSessionRepo := repository.NewUserSessionRepository(query.New(testutil.GetTestDB()))
	sess, err := userSessionRepo.FindByToken(t.Context(), sessionToken)
	if err != nil {
		t.Fatalf("セッションの取得に失敗: %v", err)
	}
	if sess == nil {
		t.Fatal("セッションが見つからない")
	}
	if sess.IPAddress != "203.0.113.7" {
		t.Errorf("セッションのIPアドレス = %q、期待値 = %q", sess.IPAddress, "203.0.113.7")
	}
}
