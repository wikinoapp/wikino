package settings_email_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	"github.com/wikinoapp/wikino/go/internal/handler/settings_email"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/ratelimit"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

const testPassword = "password123"

// mockJobInserterはメールの送信のジョブの投入を記録する
type mockJobInserter struct {
	called bool
}

func (m *mockJobInserter) Insert(_ context.Context, _ river.JobArgs, _ *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	m.called = true
	return &rivertype.JobInsertResult{}, nil
}

type testSetup struct {
	handler  *settings_email.Handler
	inserter *mockJobInserter
	limiter  *ratelimit.Limiter
}

func setupHandler(t *testing.T, tx *sql.Tx) *testSetup {
	t.Helper()

	cfg := &config.Config{Env: "test", Domain: "localhost", SessionHTTPOnly: true}
	queries := testutil.QueriesWithTx(tx)
	userRepo := repository.NewUserRepository(queries)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)
	inserter := &mockJobInserter{}
	createEmailConfirmationUC := usecase.NewCreateEmailConfirmationUsecase(
		cfg,
		emailConfirmationRepo,
		dispatcher.NewDispatcher(inserter),
		validator.NewEmailConfirmationCreateValidator(userRepo),
	)
	limiter := ratelimit.NewLimiter(repository.NewRateLimitRepository(queries))

	return &testSetup{
		handler: settings_email.NewHandler(
			cfg,
			session.NewManager(userRepo, repository.NewUserSessionRepository(queries), cfg),
			session.NewFlashManager("", false, true),
			usecase.NewCreateEmailUpdateConfirmationUsecase(
				validator.NewSettingsEmailUpdateValidator(userRepo, repository.NewUserPasswordRepository(queries)),
				createEmailConfirmationUC,
			),
			limiter,
		),
		inserter: inserter,
		limiter:  limiter,
	}
}

// newEmailRequestはログイン中のユーザー・CSRFトークン・ロケールを載せたリクエストを組み立てる。
// formがnilなら本文を持たないリクエストにし、userがnilならユーザーを載せない。
func newEmailRequest(t *testing.T, method string, user *model.User, form url.Values) *http.Request {
	t.Helper()

	var req *http.Request
	if form == nil {
		req = httptest.NewRequest(method, "/settings/email", nil)
	} else {
		req = httptest.NewRequest(method, "/settings/email", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	if user != nil {
		ctx = middleware.SetUserToContext(ctx, user)
	}
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	return req.WithContext(ctx)
}

// emailUserはパスワードを持つユーザーを作り、コンテキストに載せるモデルを返す
func emailUser(t *testing.T, tx *sql.Tx, atname string) *model.User {
	t.Helper()

	digest, err := auth.HashPassword(testPassword)
	if err != nil {
		t.Fatalf("パスワードのハッシュ化に失敗: %v", err)
	}
	email := atname + "@example.com"
	userID := testutil.NewUserBuilder(t, tx).
		WithEmail(email).
		WithAtname(atname).
		BuildWithPassword(digest)

	return &model.User{ID: userID, Atname: atname, Email: email}
}

func findCookie(rr *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, c := range rr.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func countConfirmations(t *testing.T, tx *sql.Tx, email string) int {
	t.Helper()

	var count int
	if err := tx.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM email_confirmations WHERE email = $1", email).Scan(&count); err != nil {
		t.Fatalf("確認の件数の取得に失敗: %v", err)
	}
	return count
}

func TestShow_現在のメールアドレスとフォームを表示する(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	user := emailUser(t, tx, "settings_email_show")

	rr := httptest.NewRecorder()
	setupHandler(t, tx).handler.Show(rr, newEmailRequest(t, http.MethodGet, user, nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("X-Robots-Tag"); got != "noindex" {
		t.Errorf("X-Robots-Tag = %q、期待値 = %q", got, "noindex")
	}
	body := rr.Body.String()
	for _, want := range []string{
		"<title>メールアドレスの変更 | Wikino</title>",
		"settings_email_show@example.com",
		`name="new_email"`,
		`name="password"`,
		`autocomplete="current-password"`,
		`action="/settings/email"`,
		`value="PATCH"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに %q が含まれていない", want)
		}
	}
}

func TestShow_ログインしていなければログインの画面へ送る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	rr := httptest.NewRecorder()
	setupHandler(t, tx).handler.Show(rr, newEmailRequest(t, http.MethodGet, nil, nil))

	if rr.Code != http.StatusFound || rr.Header().Get("Location") != "/sign_in" {
		t.Errorf("ステータスコード = %d、Location = %q、/sign_inへのリダイレクトを期待", rr.Code, rr.Header().Get("Location"))
	}
}

func TestUpdate_確認コードを送って入力の画面へ送る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	user := emailUser(t, tx, "settings_email_send")
	setup := setupHandler(t, tx)

	rr := httptest.NewRecorder()
	setup.handler.Update(rr, newEmailRequest(t, http.MethodPatch, user, url.Values{
		"new_email": {"settings-email-send-new@example.com"},
		"password":  {testPassword},
	}))

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusSeeOther)
	}
	if got := rr.Header().Get("Location"); got != "/settings/email/confirmation/edit" {
		t.Errorf("Location = %q、期待値 = %q", got, "/settings/email/confirmation/edit")
	}
	cookie := findCookie(rr, session.SettingsEmailConfirmationCookieName)
	if cookie == nil || cookie.Value == "" {
		t.Fatal("確認のIDのCookieが設定されていない")
	}
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode || cookie.MaxAge != 15*60 {
		t.Errorf("Cookieの属性 = %+v、HttpOnly・SameSite=Lax・15分を期待", cookie)
	}
	if findCookie(rr, session.EmailConfirmationCookieName) != nil {
		t.Error("新規登録の確認のCookieが設定された")
	}
	if !setup.inserter.called {
		t.Error("メールの送信のジョブが投入されていない")
	}
	if got := countConfirmations(t, tx, "settings-email-send-new@example.com"); got != 1 {
		t.Errorf("確認の件数 = %d、期待値 = 1", got)
	}
}

func TestUpdate_入力が誤っていれば確認コードを送らない(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		newEmail  string
		password  string
		wantError string
	}{
		{name: "パスワードが違う", newEmail: "settings-email-wrong-new@example.com", password: "wrong-password", wantError: "パスワードが正しくありません"},
		{name: "形式が誤り", newEmail: "invalid-email", password: testPassword, wantError: "有効なメールアドレスを入力してください"},
		{name: "登録済みのメールアドレス", newEmail: "settings_email_taken@example.com", password: testPassword, wantError: "このメールアドレスは既に登録されています"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			user := emailUser(t, tx, "settings_email_invalid")
			emailUser(t, tx, "settings_email_taken")
			setup := setupHandler(t, tx)

			rr := httptest.NewRecorder()
			setup.handler.Update(rr, newEmailRequest(t, http.MethodPatch, user, url.Values{
				"new_email": {tt.newEmail},
				"password":  {tt.password},
			}))

			if rr.Code != http.StatusUnprocessableEntity {
				t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusUnprocessableEntity)
			}
			body := rr.Body.String()
			if !strings.Contains(body, tt.wantError) {
				t.Errorf("レスポンスに %q が含まれていない", tt.wantError)
			}
			if strings.Contains(body, testPassword) {
				t.Error("入力したパスワードがレスポンスに含まれている")
			}
			if findCookie(rr, session.SettingsEmailConfirmationCookieName) != nil {
				t.Error("確認のIDのCookieが設定された")
			}
			if setup.inserter.called {
				t.Error("メールの送信のジョブが投入された")
			}
			if got := countConfirmations(t, tx, tt.newEmail); got != 0 {
				t.Errorf("確認の件数 = %d、期待値 = 0", got)
			}
		})
	}
}

func TestUpdate_レート制限を超えたら確認コードを送らない(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  func(user *model.User) string
	}{
		{name: "ユーザー単位", key: func(user *model.User) string { return ratelimit.SettingsEmailUserKey(string(user.ID)) }},
		{name: "宛先のメールアドレス単位", key: func(*model.User) string { return ratelimit.EmailKey("settings-email-limited@example.com") }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			user := emailUser(t, tx, "settings_email_limit")
			setup := setupHandler(t, tx)

			// 制限の回数まで使い切っておく
			for range 5 {
				_ = setup.limiter.Allow(context.Background(), ratelimit.CheckInput{Key: tt.key(user), Limit: 5, Window: time.Hour})
			}

			rr := httptest.NewRecorder()
			setup.handler.Update(rr, newEmailRequest(t, http.MethodPatch, user, url.Values{
				"new_email": {"settings-email-limited@example.com"},
				"password":  {testPassword},
			}))

			if rr.Code != http.StatusUnprocessableEntity {
				t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusUnprocessableEntity)
			}
			if !strings.Contains(rr.Body.String(), "リクエストが多すぎます") {
				t.Error("レート制限のエラーが表示されていない")
			}
			if setup.inserter.called {
				t.Error("メールの送信のジョブが投入された")
			}
		})
	}
}
