package settings_two_factor_auth_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/settings_two_factor_auth"
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

const (
	testPassword = "password123"
	testSecret   = "JBSWY3DPEHPK3PXPJBSWY3DPEHPK3PXP"
)

type testSetup struct {
	handler *settings_two_factor_auth.Handler
	repo    *repository.UserTwoFactorAuthRepository
}

func setupHandler(t *testing.T, tx *sql.Tx) *testSetup {
	t.Helper()

	cfg := &config.Config{Env: "test", Domain: "localhost", SessionHTTPOnly: true}
	queries := testutil.QueriesWithTx(tx)
	repo := repository.NewUserTwoFactorAuthRepository(queries)

	return &testSetup{
		handler: settings_two_factor_auth.NewHandler(
			cfg,
			session.NewFlashManager("", false, true),
			ratelimit.NewLimiter(repository.NewRateLimitRepository(queries)),
			usecase.NewGetTwoFactorAuthSettingsUsecase(repo),
			usecase.NewPrepareTwoFactorAuthUsecase(repo),
			usecase.NewGetPendingTwoFactorAuthUsecase(repo),
			usecase.NewEnableTwoFactorAuthUsecase(
				validator.NewSettingsTwoFactorAuthCreateValidator(repository.NewUserPasswordRepository(queries)),
				repo,
			),
			usecase.NewDisableTwoFactorAuthUsecase(
				validator.NewSettingsTwoFactorAuthDeleteValidator(repository.NewUserPasswordRepository(queries)),
				repo,
			),
		),
		repo: repo,
	}
}

// newRequestはログイン中のユーザー・CSRFトークン・ロケールを載せたリクエストを組み立てる。
// formがnilなら本文を持たないリクエストにし、userがnilならユーザーを載せない。
func newRequest(t *testing.T, method, path string, user *model.User, form url.Values) *http.Request {
	t.Helper()

	var req *http.Request
	if form == nil {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	if user != nil {
		ctx = middleware.SetUserToContext(ctx, user)
	}
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	return req.WithContext(ctx)
}

// newUserはパスワードを持つユーザーを作る。secretが空でなければ二要素認証の設定 (enabledで有効か) も作る
func newUser(t *testing.T, tx *sql.Tx, atname, secret string, enabled bool) *model.User {
	t.Helper()

	digest, err := auth.HashPassword(testPassword)
	if err != nil {
		t.Fatalf("パスワードのハッシュ化に失敗: %v", err)
	}
	email := atname + "@example.com"
	builder := testutil.NewUserBuilder(t, tx).WithEmail(email).WithAtname(atname)
	var userID model.UserID
	if secret == "" {
		userID = builder.BuildWithPassword(digest)
	} else {
		userID = builder.BuildWithPasswordAndTwoFactorAuth(digest, secret, enabled)
	}

	return &model.User{ID: userID, Atname: atname, Email: email}
}

func currentCode(t *testing.T, secret string) string {
	t.Helper()

	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatalf("コードの生成に失敗: %v", err)
	}
	return code
}

// wrongCodeは、今の許容幅のどのコードとも一致しない6桁の数字を返す
func wrongCode(t *testing.T, secret string) string {
	t.Helper()

	for _, c := range []string{"000000", "111111", "222222", "333333"} {
		if !auth.ValidateTOTPCode(c, secret, time.Now()) {
			return c
		}
	}
	t.Fatal("誤りのコードを用意できなかった")
	return ""
}

func assertSettingsHeaders(t *testing.T, rr *httptest.ResponseRecorder) {
	t.Helper()

	if got := rr.Header().Get("X-Robots-Tag"); got != "noindex" {
		t.Errorf("X-Robots-Tag = %q、期待値 = %q", got, "noindex")
	}
	if got := rr.Header().Get("Cache-Control"); got != "private, no-store" {
		t.Errorf("Cache-Control = %q、期待値 = %q", got, "private, no-store")
	}
}

func TestShow_無効なら有効にする画面への入口を出す(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	user := newUser(t, tx, "settings_2fa_show_off", "", false)

	rr := httptest.NewRecorder()
	setupHandler(t, tx).handler.Show(rr, newRequest(t, http.MethodGet, "/settings/two_factor_auth", user, nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	assertSettingsHeaders(t, rr)
	body := rr.Body.String()
	for _, want := range []string{
		"<title>二要素認証の設定 | Wikino</title>",
		"二要素認証は未設定です",
		`href="/settings/two_factor_auth/new"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに %q が含まれていない", want)
		}
	}
	if strings.Contains(body, "/settings/two_factor_auth/recovery_codes") {
		t.Error("無効なのにリカバリーコードへのリンクが出ている")
	}
}

func TestShow_登録の途中は無効として出す(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	user := newUser(t, tx, "settings_2fa_show_pend", testSecret, false)

	rr := httptest.NewRecorder()
	setupHandler(t, tx).handler.Show(rr, newRequest(t, http.MethodGet, "/settings/two_factor_auth", user, nil))

	if body := rr.Body.String(); !strings.Contains(body, "二要素認証は未設定です") {
		t.Error("登録の途中なのに無効と表示されていない")
	}
}

func TestShow_有効なら状態とリカバリーコードへのリンクを出す(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	user := newUser(t, tx, "settings_2fa_show_on", testSecret, true)

	rr := httptest.NewRecorder()
	setupHandler(t, tx).handler.Show(rr, newRequest(t, http.MethodGet, "/settings/two_factor_auth", user, nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, want := range []string{
		"二要素認証は有効です",
		"に有効化",
		`href="/settings/two_factor_auth/recovery_codes"`,
		`value="DELETE"`,
		`data-confirm="二要素認証を無効にしてもよろしいですか？"`,
		`autocomplete="current-password"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに %q が含まれていない", want)
		}
	}
	if strings.Contains(body, testSecret) {
		t.Error("設定の画面に秘密鍵が出ている")
	}
}

func TestShow_ログインしていなければログインの画面へ送る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	rr := httptest.NewRecorder()
	setupHandler(t, tx).handler.Show(rr, newRequest(t, http.MethodGet, "/settings/two_factor_auth", nil, nil))

	if rr.Code != http.StatusFound || rr.Header().Get("Location") != "/sign_in" {
		t.Errorf("ステータスコード = %d、Location = %q、/sign_inへのリダイレクトを期待", rr.Code, rr.Header().Get("Location"))
	}
}

func TestNew_新しい秘密鍵とQRコードを出す(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	user := newUser(t, tx, "settings_2fa_new", "", false)
	setup := setupHandler(t, tx)

	rr := httptest.NewRecorder()
	setup.handler.New(rr, newRequest(t, http.MethodGet, "/settings/two_factor_auth/new", user, nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	assertSettingsHeaders(t, rr)

	pending, err := setup.repo.FindByUserID(context.Background(), user.ID)
	if err != nil || pending == nil || pending.Enabled {
		t.Fatalf("登録の途中の設定 = %+v、エラー = %v", pending, err)
	}
	body := rr.Body.String()
	for _, want := range []string{
		"<title>二要素認証を有効にする | Wikino</title>",
		pending.Secret,
		`<svg aria-label="認証アプリに登録するQRコード"`,
		`<path fill="#000" d="M0 0h7v1h-7z`,
		`action="/settings/two_factor_auth"`,
		`name="password"`,
		`autocomplete="current-password"`,
		`name="totp_code"`,
		`autocomplete="one-time-code"`,
		`inputmode="numeric"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに %q が含まれていない", want)
		}
	}
	// 空白を含むコードを貼り付けたときに、ブラウザが末尾の数字を切り詰めないようにする
	if strings.Contains(body, "maxlength") {
		t.Error("認証コードの欄にmaxlengthがあり、空白付きのコードの貼り付けで末尾が切り詰められる")
	}
}

func TestNew_有効なら秘密鍵を差し替えずに設定の画面へ送る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	user := newUser(t, tx, "settings_2fa_new_on", testSecret, true)
	setup := setupHandler(t, tx)

	rr := httptest.NewRecorder()
	setup.handler.New(rr, newRequest(t, http.MethodGet, "/settings/two_factor_auth/new", user, nil))

	if rr.Code != http.StatusFound || rr.Header().Get("Location") != "/settings/two_factor_auth" {
		t.Errorf("ステータスコード = %d、Location = %q、設定の画面へのリダイレクトを期待", rr.Code, rr.Header().Get("Location"))
	}
	current, err := setup.repo.FindByUserID(context.Background(), user.ID)
	if err != nil || current == nil || current.Secret != testSecret || !current.Enabled {
		t.Errorf("設定 = %+v、有効な設定がそのまま残ることを期待", current)
	}
}

func TestHeadNew_設定を作成も変更もしない(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name       string
		atname     string
		secret     string
		enabled    bool
		wantStatus int
	}{
		{name: "設定が無い", atname: "head_2fa_none", wantStatus: http.StatusOK},
		{name: "登録の途中", atname: "head_2fa_pending", secret: testSecret, wantStatus: http.StatusOK},
		{name: "既に有効", atname: "head_2fa_enabled", secret: testSecret, enabled: true, wantStatus: http.StatusFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			user := newUser(t, tx, tc.atname, tc.secret, tc.enabled)
			setup := setupHandler(t, tx)
			before, err := setup.repo.FindByUserID(t.Context(), user.ID)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			setup.handler.HeadNew(rr, newRequest(t, http.MethodHead, "/settings/two_factor_auth/new", user, nil))
			if rr.Code != tc.wantStatus {
				t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, tc.wantStatus)
			}
			assertSettingsHeaders(t, rr)
			if got := rr.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
				t.Errorf("Content-Type = %q、期待値 = text/html; charset=utf-8", got)
			}
			if rr.Body.Len() != 0 {
				t.Error("HEADの応答に本文がある")
			}
			if tc.enabled && rr.Header().Get("Location") != "/settings/two_factor_auth" {
				t.Errorf("Location = %q、設定の画面へのリダイレクトを期待", rr.Header().Get("Location"))
			}

			after, err := setup.repo.FindByUserID(t.Context(), user.ID)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(before, after) {
				t.Errorf("HEADで設定が変わった: 変更前 = %+v、変更後 = %+v", before, after)
			}
		})
	}
}

func TestHeadNew_ログインしていなければログインの画面へ送る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	rr := httptest.NewRecorder()
	setupHandler(t, tx).handler.HeadNew(rr, newRequest(t, http.MethodHead, "/settings/two_factor_auth/new", nil, nil))
	if rr.Code != http.StatusFound || rr.Header().Get("Location") != "/sign_in" {
		t.Errorf("ステータスコード = %d、Location = %q、ログインの画面へのリダイレクトを期待", rr.Code, rr.Header().Get("Location"))
	}
	if rr.Body.Len() != 0 {
		t.Error("HEADの応答に本文がある")
	}
}

func TestCreate_有効にしてリカバリーコードの画面へ送る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	user := newUser(t, tx, "settings_2fa_create", testSecret, false)
	setup := setupHandler(t, tx)

	rr := httptest.NewRecorder()
	setup.handler.Create(rr, newRequest(t, http.MethodPost, "/settings/two_factor_auth", user, url.Values{
		"password":  {testPassword},
		"totp_code": {currentCode(t, testSecret)},
	}))

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusSeeOther)
	}
	if got := rr.Header().Get("Location"); got != "/settings/two_factor_auth/recovery_codes" {
		t.Errorf("Location = %q、期待値 = %q", got, "/settings/two_factor_auth/recovery_codes")
	}
	current, err := setup.repo.FindByUserID(context.Background(), user.ID)
	if err != nil || current == nil || !current.Enabled {
		t.Errorf("設定 = %+v、有効になっていることを期待", current)
	}
}

func TestCreate_コードを誤ったら同じ秘密鍵で描き直す(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	user := newUser(t, tx, "settings_2fa_create_ng", testSecret, false)
	setup := setupHandler(t, tx)

	rr := httptest.NewRecorder()
	setup.handler.Create(rr, newRequest(t, http.MethodPost, "/settings/two_factor_auth", user, url.Values{
		"password":  {testPassword},
		"totp_code": {wrongCode(t, testSecret)},
	}))

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusUnprocessableEntity)
	}
	assertSettingsHeaders(t, rr)
	body := rr.Body.String()
	for _, want := range []string{testSecret, "認証コードが正しくありません", `aria-invalid="true"`} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに %q が含まれていない", want)
		}
	}
	current, err := setup.repo.FindByUserID(context.Background(), user.ID)
	if err != nil || current == nil || current.Enabled || current.Secret != testSecret {
		t.Errorf("設定 = %+v、同じ秘密鍵の無効な設定のままを期待", current)
	}
}

func TestCreate_登録の途中の設定が無ければ設定の画面へ送る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	user := newUser(t, tx, "settings_2fa_create_on", testSecret, true)

	rr := httptest.NewRecorder()
	setupHandler(t, tx).handler.Create(rr, newRequest(t, http.MethodPost, "/settings/two_factor_auth", user, url.Values{
		"password":  {testPassword},
		"totp_code": {currentCode(t, testSecret)},
	}))

	if rr.Code != http.StatusSeeOther || rr.Header().Get("Location") != "/settings/two_factor_auth" {
		t.Errorf("ステータスコード = %d、Location = %q、設定の画面へのリダイレクトを期待", rr.Code, rr.Header().Get("Location"))
	}
}

func TestCreate_試行が多すぎたら照合せずに描き直す(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	user := newUser(t, tx, "settings_2fa_create_rl", testSecret, false)
	setup := setupHandler(t, tx)

	for i := range 5 {
		rr := httptest.NewRecorder()
		setup.handler.Create(rr, newRequest(t, http.MethodPost, "/settings/two_factor_auth", user, url.Values{
			"password":  {testPassword},
			"totp_code": {wrongCode(t, testSecret)},
		}))
		if rr.Code != http.StatusUnprocessableEntity {
			t.Fatalf("%d回目のステータスコード = %d、期待値 = %d", i+1, rr.Code, http.StatusUnprocessableEntity)
		}
	}

	// 上限を超えたら、正しいコードでも有効にしない
	rr := httptest.NewRecorder()
	setup.handler.Create(rr, newRequest(t, http.MethodPost, "/settings/two_factor_auth", user, url.Values{
		"password":  {testPassword},
		"totp_code": {currentCode(t, testSecret)},
	}))
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusUnprocessableEntity)
	}
	if !strings.Contains(rr.Body.String(), "リクエストが多すぎます") {
		t.Error("レート制限のエラーが表示されていない")
	}
	current, err := setup.repo.FindByUserID(context.Background(), user.ID)
	if err != nil || current == nil || current.Enabled {
		t.Errorf("設定 = %+v、有効になっていないことを期待", current)
	}
}
