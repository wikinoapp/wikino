package settings_email_confirmation_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/settings_email_confirmation"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// setupHandlerは、確定の処理が自前でトランザクションを開くため、共有の接続でハンドラーを組み立てる
func setupHandler(t *testing.T, db *sql.DB) *settings_email_confirmation.Handler {
	t.Helper()

	cfg := &config.Config{Env: "test", Domain: "localhost", SessionHTTPOnly: true}
	queries := query.New(db)
	userRepo := repository.NewUserRepository(queries)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)

	return settings_email_confirmation.NewHandler(
		cfg,
		session.NewManager(userRepo, repository.NewUserSessionRepository(queries), cfg),
		session.NewFlashManager("", false, true),
		usecase.NewUpdateEmailUsecase(
			db,
			validator.NewSettingsEmailConfirmationUpdateValidator(emailConfirmationRepo, userRepo),
			emailConfirmationRepo,
			userRepo,
		),
	)
}

// newConfirmationRequestは、ログイン中のユーザー・CSRFトークン・ロケールと、confirmationIDが
// 空でなければ確認のIDのCookieを載せたリクエストを組み立てる
func newConfirmationRequest(t *testing.T, method, path string, user *model.User, confirmationID string, form url.Values) *http.Request {
	t.Helper()

	var req *http.Request
	if form == nil {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if confirmationID != "" {
		req.AddCookie(&http.Cookie{Name: session.SettingsEmailConfirmationCookieName, Value: confirmationID})
	}

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	if user != nil {
		ctx = middleware.SetUserToContext(ctx, user)
	}
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	return req.WithContext(ctx)
}

// confirmationUserはユーザーと、そのユーザーのメールアドレスの変更の確認を作る。確認コードには
// 一意制約があり、テストの間でデータをコミットするため、テストごとに別のコードを渡す
func confirmationUser(t *testing.T, db *sql.DB, atname, newEmail, code string, event model.EmailConfirmationEvent) (*model.User, string) {
	t.Helper()

	email := atname + "@example.com"
	userID := testutil.NewUserBuilderDB(t, db).WithEmail(email).WithAtname(atname).Build()
	confirmationID := testutil.NewEmailConfirmationBuilderDB(t, db).
		WithEmail(newEmail).
		WithUserID(userID).
		WithEvent(event).
		WithCode(code).
		WithStartedAt(time.Now()).
		Build()

	return &model.User{ID: userID, Atname: atname, Email: email}, confirmationID
}

func findUserEmail(t *testing.T, db *sql.DB, userID model.UserID) string {
	t.Helper()

	var email string
	if err := db.QueryRowContext(context.Background(), "SELECT email FROM users WHERE id = $1", string(userID)).Scan(&email); err != nil {
		t.Fatalf("メールアドレスの取得に失敗: %v", err)
	}
	return email
}

func TestUpdate_不正な確認IDを検証エラーにする(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	user, _ := confirmationUser(t, db, "sec_bad_cookie", "sec-bad-cookie-new@example.com", "SE0091", model.EmailConfirmationEventEmailUpdate)
	rr := httptest.NewRecorder()
	setupHandler(t, db).Update(rr, newConfirmationRequest(t, http.MethodPatch, "/settings/email/confirmation", user, "invalid-id", url.Values{"code": {"SE0091"}}))

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusUnprocessableEntity)
	}
	if !strings.Contains(rr.Body.String(), "確認情報が見つかりません") {
		t.Error("無効な確認であることが表示されていない")
	}
	if got := findUserEmail(t, db, user.ID); got != user.Email {
		t.Errorf("Email = %q、変更前 = %q", got, user.Email)
	}
}

func TestEdit_確認コードの入力欄を表示する(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	user, confirmationID := confirmationUser(t, db, "sec_edit_show", "sec-edit-show-new@example.com", "SE0001", model.EmailConfirmationEventEmailUpdate)

	rr := httptest.NewRecorder()
	setupHandler(t, db).Edit(rr, newConfirmationRequest(t, http.MethodGet, "/settings/email/confirmation/edit", user, confirmationID, nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("X-Robots-Tag"); got != "noindex" {
		t.Errorf("X-Robots-Tag = %q、期待値 = %q", got, "noindex")
	}
	body := rr.Body.String()
	for _, want := range []string{
		"<title>確認コードの入力 | Wikino</title>",
		`name="code"`,
		`autocomplete="one-time-code"`,
		`action="/settings/email/confirmation"`,
		`href="/settings/email"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに %q が含まれていない", want)
		}
	}
}

func TestEdit_Cookieが無ければ変更のフォームへ送る(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	user, _ := confirmationUser(t, db, "sec_edit_nocookie", "sec-edit-nocookie-new@example.com", "SE0002", model.EmailConfirmationEventEmailUpdate)

	rr := httptest.NewRecorder()
	setupHandler(t, db).Edit(rr, newConfirmationRequest(t, http.MethodGet, "/settings/email/confirmation/edit", user, "", nil))

	if rr.Code != http.StatusFound || rr.Header().Get("Location") != "/settings/email" {
		t.Errorf("ステータスコード = %d、Location = %q、/settings/emailへのリダイレクトを期待", rr.Code, rr.Header().Get("Location"))
	}
}

func TestEdit_ログインしていなければログインの画面へ送る(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	rr := httptest.NewRecorder()
	setupHandler(t, db).Edit(rr, newConfirmationRequest(t, http.MethodGet, "/settings/email/confirmation/edit", nil, "dummy", nil))

	if rr.Code != http.StatusFound || rr.Header().Get("Location") != "/sign_in" {
		t.Errorf("ステータスコード = %d、Location = %q、/sign_inへのリダイレクトを期待", rr.Code, rr.Header().Get("Location"))
	}
}

func TestUpdate_メールアドレスを変更して変更のフォームへ送る(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	user, confirmationID := confirmationUser(t, db, "sec_update_ok", "sec-update-ok-new@example.com", "SE0003", model.EmailConfirmationEventEmailUpdate)

	rr := httptest.NewRecorder()
	setupHandler(t, db).Update(rr, newConfirmationRequest(t, http.MethodPatch, "/settings/email/confirmation", user, confirmationID, url.Values{
		"code": {"se0003"},
	}))

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusSeeOther)
	}
	if got := rr.Header().Get("Location"); got != "/settings/email" {
		t.Errorf("Location = %q、期待値 = %q", got, "/settings/email")
	}
	if got := findUserEmail(t, db, user.ID); got != "sec-update-ok-new@example.com" {
		t.Errorf("メールアドレス = %q、期待値 = %q", got, "sec-update-ok-new@example.com")
	}
	var deleted bool
	for _, c := range rr.Result().Cookies() {
		if c.Name == session.SettingsEmailConfirmationCookieName && c.MaxAge < 0 {
			deleted = true
		}
	}
	if !deleted {
		t.Error("確認のIDのCookieが削除されていない")
	}
}

func TestUpdate_確定できなければメールアドレスを変えない(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		atname    string
		event     model.EmailConfirmationEvent
		savedCode string
		code      string
		wantError string
	}{
		{name: "確認コードが違う", atname: "sec_update_mismatch", event: model.EmailConfirmationEventEmailUpdate, savedCode: "SE0005", code: "XYZ999", wantError: "確認コードが正しくありません"},
		{name: "新規登録の確認", atname: "sec_update_signup", event: model.EmailConfirmationEventSignUp, savedCode: "SE0006", code: "SE0006", wantError: "確認情報が見つかりません"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := testutil.GetTestDB()
			newEmail := strings.ReplaceAll(tt.atname, "_", "-") + "-new@example.com"
			user, confirmationID := confirmationUser(t, db, tt.atname, newEmail, tt.savedCode, tt.event)

			rr := httptest.NewRecorder()
			setupHandler(t, db).Update(rr, newConfirmationRequest(t, http.MethodPatch, "/settings/email/confirmation", user, confirmationID, url.Values{
				"code": {tt.code},
			}))

			if rr.Code != http.StatusUnprocessableEntity {
				t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusUnprocessableEntity)
			}
			if !strings.Contains(rr.Body.String(), tt.wantError) {
				t.Errorf("レスポンスに %q が含まれていない", tt.wantError)
			}
			if got := findUserEmail(t, db, user.ID); got != user.Email {
				t.Errorf("メールアドレス = %q、変わっていないことを期待", got)
			}
		})
	}
}

func TestUpdate_Cookieが無ければ変更のフォームへ送る(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	user, _ := confirmationUser(t, db, "sec_update_nocookie", "sec-update-nocookie-new@example.com", "SE0004", model.EmailConfirmationEventEmailUpdate)

	rr := httptest.NewRecorder()
	setupHandler(t, db).Update(rr, newConfirmationRequest(t, http.MethodPatch, "/settings/email/confirmation", user, "", url.Values{
		"code": {"ABC123"},
	}))

	if rr.Code != http.StatusSeeOther || rr.Header().Get("Location") != "/settings/email" {
		t.Errorf("ステータスコード = %d、Location = %q、/settings/emailへのリダイレクトを期待", rr.Code, rr.Header().Get("Location"))
	}
	if got := findUserEmail(t, db, user.ID); got != user.Email {
		t.Errorf("メールアドレス = %q、変わっていないことを期待", got)
	}
}
