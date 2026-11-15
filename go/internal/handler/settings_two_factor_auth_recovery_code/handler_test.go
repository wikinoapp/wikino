package settings_two_factor_auth_recovery_code_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/settings_two_factor_auth_recovery_code"
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
	recoveryPath = "/settings/two_factor_auth/recovery_codes"
)

type testSetup struct {
	handler *settings_two_factor_auth_recovery_code.Handler
	repo    *repository.UserTwoFactorAuthRepository
}

func setupHandler(t *testing.T, tx *sql.Tx) *testSetup {
	t.Helper()

	cfg := &config.Config{Env: "test", Domain: "localhost", SessionHTTPOnly: true}
	queries := testutil.QueriesWithTx(tx)
	repo := repository.NewUserTwoFactorAuthRepository(queries)

	return &testSetup{
		handler: settings_two_factor_auth_recovery_code.NewHandler(
			cfg,
			session.NewFlashManager("", false, true),
			ratelimit.NewLimiter(repository.NewRateLimitRepository(queries)),
			usecase.NewGetTwoFactorAuthRecoveryCodesUsecase(repo),
			usecase.NewRegenerateTwoFactorAuthRecoveryCodesUsecase(
				validator.NewSettingsTwoFactorAuthRecoveryCodeCreateValidator(repository.NewUserPasswordRepository(queries)),
				repo,
			),
		),
		repo: repo,
	}
}

func newRequest(t *testing.T, method string, user *model.User, form url.Values) *http.Request {
	t.Helper()

	var req *http.Request
	if form == nil {
		req = httptest.NewRequest(method, recoveryPath, nil)
	} else {
		req = httptest.NewRequest(method, recoveryPath, strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	if user != nil {
		ctx = middleware.SetUserToContext(ctx, user)
	}
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	return req.WithContext(ctx)
}

// newUserは、パスワードと二要素認証の設定 (enabledで有効か・残っているリカバリーコード) を持つユーザーを作る
func newUser(t *testing.T, tx *sql.Tx, atname string, enabled bool, recoveryCodes []string) *model.User {
	t.Helper()

	digest, err := auth.HashPassword(testPassword)
	if err != nil {
		t.Fatalf("パスワードのハッシュ化に失敗: %v", err)
	}
	email := atname + "@example.com"
	userID := testutil.NewUserBuilder(t, tx).WithEmail(email).WithAtname(atname).BuildWithPassword(digest)
	if _, err := tx.ExecContext(context.Background(),
		"INSERT INTO user_two_factor_auths (user_id, secret, enabled, enabled_at, recovery_codes, created_at, updated_at) VALUES ($1, $2, $3, CASE WHEN $3 THEN NOW() END, $4, NOW(), NOW())",
		string(userID), testSecret, enabled, "{"+strings.Join(recoveryCodes, ",")+"}"); err != nil {
		t.Fatalf("二要素認証の設定の作成に失敗: %v", err)
	}

	return &model.User{ID: userID, Atname: atname, Email: email}
}

// autofocusPatternは、autofocusを持つ要素の開始タグに一致する。誤りを示して描き直したときに
// どこへフォーカスするかを確かめるために使う。
var autofocusPattern = regexp.MustCompile(`<[a-z]+\b[^>]*\bautofocus\b[^>]*>`)

// assertFocusは、autofocusを持つ要素が1つだけで、その開始タグにwantを含むことを確かめる。
// 2つあるとブラウザは最初の要素にだけフォーカスし、無いと誤りへフォーカスが移らない。
func assertFocus(t *testing.T, body, want string) {
	t.Helper()

	if focused := autofocusPattern.FindAllString(body, -1); len(focused) != 1 || !strings.Contains(focused[0], want) {
		t.Errorf("フォーカス先 = %v、%qを含む要素1つだけを期待", focused, want)
	}
}

func TestIndex_有効ならリカバリーコードと作り直すフォームを出す(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	user := newUser(t, tx, "settings_2fa_codes", true, []string{"abcd1234", "efgh5678"})

	rr := httptest.NewRecorder()
	setupHandler(t, tx).handler.Index(rr, newRequest(t, http.MethodGet, user, nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("X-Robots-Tag"); got != "noindex" {
		t.Errorf("X-Robots-Tag = %q、期待値 = %q", got, "noindex")
	}
	if got := rr.Header().Get("Cache-Control"); got != "private, no-store" {
		t.Errorf("Cache-Control = %q、期待値 = %q", got, "private, no-store")
	}
	body := rr.Body.String()
	for _, want := range []string{
		"<title>リカバリーコード | Wikino</title>",
		"abcd1234",
		"efgh5678",
		`action="/settings/two_factor_auth/recovery_codes"`,
		`name="password"`,
		`autocomplete="current-password"`,
		`data-confirm="リカバリーコードを再生成してもよろしいですか？今のコードはすべて使えなくなります"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに %q が含まれていない", want)
		}
	}
}

func TestIndex_使い切ったら作り直すよう案内する(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	user := newUser(t, tx, "settings_2fa_codes_empty", true, nil)

	rr := httptest.NewRecorder()
	setupHandler(t, tx).handler.Index(rr, newRequest(t, http.MethodGet, user, nil))

	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), "残っているリカバリーコードはありません") {
		t.Errorf("ステータスコード = %d、使い切ったことの案内を期待", rr.Code)
	}
}

func TestIndex_有効でなければ設定の画面へ送る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	user := newUser(t, tx, "settings_2fa_codes_off", false, nil)

	rr := httptest.NewRecorder()
	setupHandler(t, tx).handler.Index(rr, newRequest(t, http.MethodGet, user, nil))

	if rr.Code != http.StatusSeeOther || rr.Header().Get("Location") != "/settings/two_factor_auth" {
		t.Errorf("ステータスコード = %d、Location = %q、設定の画面へのリダイレクトを期待", rr.Code, rr.Header().Get("Location"))
	}
	if len(rr.Header().Values("Set-Cookie")) == 0 {
		t.Error("有効でないことをフラッシュで知らせていない")
	}
}

func TestIndex_HEADは有効でなくてもフラッシュを設定しない(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	user := newUser(t, tx, "settings_2fa_codes_head_off", false, nil)

	rr := httptest.NewRecorder()
	setupHandler(t, tx).handler.Index(rr, newRequest(t, http.MethodHead, user, nil))

	if rr.Code != http.StatusSeeOther || rr.Header().Get("Location") != "/settings/two_factor_auth" {
		t.Errorf("ステータスコード = %d、Location = %q、設定の画面へのリダイレクトを期待", rr.Code, rr.Header().Get("Location"))
	}
	// 本文を読まないHEADでフラッシュを設定すると、次に開いた画面に覚えの無い警告が出る
	if got := rr.Header().Values("Set-Cookie"); len(got) != 0 {
		t.Errorf("Set-Cookie = %v、HEADではフラッシュを設定しないことを期待", got)
	}
}

func TestIndex_ログインしていなければログインの画面へ送る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	rr := httptest.NewRecorder()
	setupHandler(t, tx).handler.Index(rr, newRequest(t, http.MethodGet, nil, nil))

	if rr.Code != http.StatusFound || rr.Header().Get("Location") != "/sign_in" {
		t.Errorf("ステータスコード = %d、Location = %q、/sign_inへのリダイレクトを期待", rr.Code, rr.Header().Get("Location"))
	}
}

func TestCreate_作り直してリカバリーコードの画面へ送る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	user := newUser(t, tx, "settings_2fa_regen", true, []string{"oldcode1"})
	setup := setupHandler(t, tx)

	rr := httptest.NewRecorder()
	setup.handler.Create(rr, newRequest(t, http.MethodPost, user, url.Values{"password": {testPassword}}))

	if rr.Code != http.StatusSeeOther || rr.Header().Get("Location") != recoveryPath {
		t.Fatalf("ステータスコード = %d、Location = %q、リカバリーコードの画面へのリダイレクトを期待", rr.Code, rr.Header().Get("Location"))
	}
	current, err := setup.repo.FindByUserID(context.Background(), user.ID)
	if err != nil || current == nil || len(current.RecoveryCodes) != auth.RecoveryCodeCount {
		t.Fatalf("設定 = %+v、新しいリカバリーコードを期待 (エラー = %v)", current, err)
	}
	for _, code := range current.RecoveryCodes {
		if code == "oldcode1" {
			t.Error("前のリカバリーコードが残っている")
		}
	}
}

func TestCreate_パスワードを誤ったら今のコードのまま描き直す(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	user := newUser(t, tx, "settings_2fa_regen_ng", true, []string{"keepcode"})
	setup := setupHandler(t, tx)

	rr := httptest.NewRecorder()
	setup.handler.Create(rr, newRequest(t, http.MethodPost, user, url.Values{"password": {"wrong-password"}}))

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusUnprocessableEntity)
	}
	body := rr.Body.String()
	for _, want := range []string{"keepcode", "パスワードが正しくありません", `aria-invalid="true"`} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに %q が含まれていない", want)
		}
	}
	// 誤りがパスワードの欄だけのときは、その欄へフォーカスを移す
	assertFocus(t, body, `id="password"`)
	current, err := setup.repo.FindByUserID(context.Background(), user.ID)
	if err != nil || current == nil || len(current.RecoveryCodes) != 1 || current.RecoveryCodes[0] != "keepcode" {
		t.Errorf("設定 = %+v、リカバリーコードが変わらないことを期待 (エラー = %v)", current, err)
	}
}

func TestCreate_有効でなければ設定の画面へ送る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	user := newUser(t, tx, "settings_2fa_regen_off", false, nil)

	rr := httptest.NewRecorder()
	setupHandler(t, tx).handler.Create(rr, newRequest(t, http.MethodPost, user, url.Values{"password": {testPassword}}))

	if rr.Code != http.StatusSeeOther || rr.Header().Get("Location") != "/settings/two_factor_auth" {
		t.Errorf("ステータスコード = %d、Location = %q、設定の画面へのリダイレクトを期待", rr.Code, rr.Header().Get("Location"))
	}
}

func TestCreate_試行が多すぎたら照合せずに描き直す(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	user := newUser(t, tx, "settings_2fa_regen_rl", true, []string{"keepcode"})
	setup := setupHandler(t, tx)

	for i := range 5 {
		rr := httptest.NewRecorder()
		setup.handler.Create(rr, newRequest(t, http.MethodPost, user, url.Values{"password": {"wrong-password"}}))
		if rr.Code != http.StatusUnprocessableEntity {
			t.Fatalf("%d回目のステータスコード = %d、期待値 = %d", i+1, rr.Code, http.StatusUnprocessableEntity)
		}
	}

	rr := httptest.NewRecorder()
	setup.handler.Create(rr, newRequest(t, http.MethodPost, user, url.Values{"password": {testPassword}}))
	if rr.Code != http.StatusUnprocessableEntity || !strings.Contains(rr.Body.String(), "リクエストが多すぎます") {
		t.Fatalf("ステータスコード = %d、レート制限のエラーを期待", rr.Code)
	}
	// 項目に結び付かない誤りは、エラーの要約を包むフォーカスできる要素へフォーカスを移す
	assertFocus(t, rr.Body.String(), `tabindex="-1"`)
	current, err := setup.repo.FindByUserID(context.Background(), user.ID)
	if err != nil || current == nil || len(current.RecoveryCodes) != 1 {
		t.Errorf("設定 = %+v、リカバリーコードが変わらないことを期待 (エラー = %v)", current, err)
	}
}
