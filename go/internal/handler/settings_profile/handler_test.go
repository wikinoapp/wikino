package settings_profile_test

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/settings_profile"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

func setupHandler(t *testing.T, tx *sql.Tx) *settings_profile.Handler {
	t.Helper()

	userRepo := repository.NewUserRepository(testutil.QueriesWithTx(tx))
	return settings_profile.NewHandler(
		&config.Config{Env: "test", Domain: "localhost"},
		session.NewFlashManager("", false, true),
		usecase.NewUpdateProfileUsecase(validator.NewProfileUpdateValidator(userRepo), userRepo),
	)
}

// newProfileRequestはログイン中のユーザー・CSRFトークン・ロケールを載せたリクエストを組み立てる。
// formがnilなら本文を持たないリクエストにし、userがnilならユーザーを載せない。
func newProfileRequest(t *testing.T, method string, user *model.User, form url.Values) *http.Request {
	t.Helper()

	var req *http.Request
	if form == nil {
		req = httptest.NewRequest(method, "/settings/profile", nil)
	} else {
		req = httptest.NewRequest(method, "/settings/profile", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	if user != nil {
		ctx = middleware.SetUserToContext(ctx, user)
	}
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	return req.WithContext(ctx)
}

// profileUserは名前と自己紹介を持つユーザーを作り、コンテキストに載せるモデルを返す
func profileUser(t *testing.T, tx *sql.Tx, atname string) *model.User {
	t.Helper()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail(atname + "@example.com").
		WithAtname(atname).
		WithName("元の名前").
		Build()

	return &model.User{ID: userID, Atname: atname, Name: "元の名前", Description: "Test description"}
}

func TestShow_保存されている値をフォームに表示する(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	user := profileUser(t, tx, "settings_profile_show")

	rr := httptest.NewRecorder()
	setupHandler(t, tx).Show(rr, newProfileRequest(t, http.MethodGet, user, nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	if got := rr.Result().Header.Get("X-Robots-Tag"); got != "noindex" {
		t.Errorf("X-Robots-Tag = %q、期待値 = %q", got, "noindex")
	}

	body := rr.Body.String()
	for _, want := range []string{
		"<title>プロフィールの編集 | Wikino</title>",
		`action="/settings/profile"`,
		`name="_method" value="PATCH"`,
		`name="csrf_token" value="test-csrf-token"`,
		`value="settings_profile_show"`,
		`value="元の名前"`,
		`value="Test description"`,
		`href="/settings"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに %q が含まれていない", want)
		}
	}
	if strings.Contains(body, "autofocus") {
		t.Error("誤りが無いのにフォーカスを置いている")
	}
}

func TestUpdate_保存してプロフィールの編集へリダイレクトする(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	user := profileUser(t, tx, "settings_profile_save")

	rr := httptest.NewRecorder()
	setupHandler(t, tx).Update(rr, newProfileRequest(t, http.MethodPatch, user, url.Values{
		"atname":      {"profile_saved"},
		"name":        {"新しい名前"},
		"description": {"新しい自己紹介"},
	}))

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusSeeOther)
	}
	if got := rr.Header().Get("Location"); got != "/settings/profile" {
		t.Errorf("Location = %q、期待値 = %q", got, "/settings/profile")
	}

	saved, err := repository.NewUserRepository(testutil.QueriesWithTx(tx)).FindByID(t.Context(), user.ID)
	if err != nil {
		t.Fatalf("FindByID()のエラー = %v", err)
	}
	if saved.Atname != "profile_saved" || saved.Name != "新しい名前" || saved.Description != "新しい自己紹介" {
		t.Errorf("保存されたユーザー = %+v", saved)
	}
}

func TestUpdate_入力が不正なら422でフォームを再描画する(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	user := profileUser(t, tx, "settings_profile_bad")

	rr := httptest.NewRecorder()
	setupHandler(t, tx).Update(rr, newProfileRequest(t, http.MethodPatch, user, url.Values{
		"atname":      {"admin"},
		"name":        {strings.Repeat("あ", validator.UserNameMaxLength+1)},
		"description": {"送った自己紹介"},
	}))

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusUnprocessableEntity)
	}
	if got := rr.Result().Header.Get("X-Robots-Tag"); got != "noindex" {
		t.Errorf("X-Robots-Tag = %q、期待値 = %q", got, "noindex")
	}

	body := rr.Body.String()
	for _, want := range []string{
		"このアットネームは使用できません",
		"名前は30文字以内で入力してください",
		`id="atname-error"`,
		`aria-describedby="atname-hint atname-error"`,
		`aria-describedby="name-hint name-error"`,
		`aria-invalid="true"`,
		`value="admin"`,
		`value="送った自己紹介"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに %q が含まれていない", want)
		}
	}
	// 最初に誤りのある項目 (アットネーム) にだけフォーカスを置く
	if got := strings.Count(body, "autofocus"); got != 1 {
		t.Errorf("autofocusの数 = %d、期待値 = 1", got)
	}

	saved, err := repository.NewUserRepository(testutil.QueriesWithTx(tx)).FindByID(t.Context(), user.ID)
	if err != nil {
		t.Fatalf("FindByID()のエラー = %v", err)
	}
	if saved.Atname != "settings_profile_bad" {
		t.Errorf("Atname = %q、変わっていないことを期待", saved.Atname)
	}
}

func TestShowとUpdate_未ログインならログインへリダイレクトする(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h := setupHandler(t, tx)

	tests := []struct {
		name   string
		method string
		serve  func(http.ResponseWriter, *http.Request)
	}{
		{name: "編集画面", method: http.MethodGet, serve: h.Show},
		{name: "保存処理", method: http.MethodPatch, serve: h.Update},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			tt.serve(rr, newProfileRequest(t, tt.method, nil, url.Values{"atname": {"anyone"}}))

			if rr.Code != http.StatusFound {
				t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusFound)
			}
			if got := rr.Header().Get("Location"); got != "/sign_in" {
				t.Errorf("Location = %q、期待値 = %q", got, "/sign_in")
			}
		})
	}
}
