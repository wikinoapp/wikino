package settings_account_deletion_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	"github.com/wikinoapp/wikino/go/internal/handler/settings_account_deletion"
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

const (
	newPath    = "/settings/account/deletion/new"
	createPath = "/settings/account/deletion"
)

// recordingJobInserterは投入されたジョブを記録する。
type recordingJobInserter struct {
	args []river.JobArgs
}

func (r *recordingJobInserter) Insert(_ context.Context, args river.JobArgs, _ *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	r.args = append(r.args, args)
	return &rivertype.JobInsertResult{}, nil
}

// setupHandlerは、削除の処理が自前でトランザクションを開くため、共有の接続でハンドラーを組み立てる。
func setupHandler(t *testing.T, db *sql.DB, inserter dispatcher.JobInserter) *settings_account_deletion.Handler {
	t.Helper()

	cfg := &config.Config{Env: "test", Domain: "localhost", SessionHTTPOnly: true}
	queries := query.New(db)
	userRepo := repository.NewUserRepository(queries)
	userSessionRepo := repository.NewUserSessionRepository(queries)
	spaceRepo := repository.NewSpaceRepository(queries)

	return settings_account_deletion.NewHandler(
		cfg,
		session.NewManager(userRepo, userSessionRepo, cfg),
		session.NewFlashManager("", false, true),
		usecase.NewGetAccountDeletionNewUsecase(spaceRepo),
		usecase.NewDiscardAccountUsecase(db, userRepo, userSessionRepo, spaceRepo, dispatcher.NewDispatcher(inserter, nil), validator.NewSettingsAccountDeletionCreateValidator(spaceRepo)),
	)
}

// newRequestは、ログイン中のユーザー (nilなら載せない)・CSRFトークン・ロケールを載せたリクエストを組み立てる。
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

// newUserはユーザーを作り、そのユーザーを返す。spaceIdentifierが空でなければ、そのスペースに参加させる。
func newUser(t *testing.T, db *sql.DB, suffix, spaceIdentifier string) *model.User {
	t.Helper()

	atname := "account_deletion_" + strings.ReplaceAll(suffix, "-", "_")
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("account-deletion-" + suffix + "@example.com").
		WithAtname(atname).
		Build()
	if spaceIdentifier != "" {
		spaceID := testutil.NewSpaceBuilderDB(t, db).
			WithIdentifier(spaceIdentifier).
			WithName("参加中のスペース " + suffix).
			Build()
		testutil.NewSpaceMemberBuilderDB(t, db).WithSpaceID(spaceID).WithUserID(userID).Build()
	}

	return &model.User{ID: userID, Atname: atname}
}

// isDiscardedはユーザーが論理削除されているかを返す。
func isDiscarded(t *testing.T, db *sql.DB, userID model.UserID) bool {
	t.Helper()

	var discarded bool
	if err := db.QueryRowContext(t.Context(), `SELECT discarded_at IS NOT NULL FROM users WHERE id = $1`, string(userID)).Scan(&discarded); err != nil {
		t.Fatalf("ユーザーの取得に失敗: %v", err)
	}
	return discarded
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

func TestNew_ログインしていなければログインの画面へ送る(t *testing.T) {
	t.Parallel()

	rr := httptest.NewRecorder()
	setupHandler(t, testutil.GetTestDB(), &recordingJobInserter{}).New(rr, newRequest(t, http.MethodGet, newPath, nil, nil))

	if rr.Code != http.StatusFound {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != "/sign_in" {
		t.Errorf("Location = %q、期待値 = /sign_in", got)
	}
}

func TestNew_参加しているスペースが無ければ削除のフォームを表示する(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	user := newUser(t, db, "new-form", "")

	rr := httptest.NewRecorder()
	setupHandler(t, db, &recordingJobInserter{}).New(rr, newRequest(t, http.MethodGet, newPath, user, nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("X-Robots-Tag"); got != "noindex" {
		t.Errorf("X-Robots-Tag = %q、期待値 = noindex", got)
	}
	if got := rr.Header().Get("Cache-Control"); got != "private, no-store" {
		t.Errorf("Cache-Control = %q、期待値 = private, no-store", got)
	}
	body := rr.Body.String()
	for _, want := range []string{
		`action="/settings/account/deletion"`,
		`name="atname"`,
		`name="csrf_token" value="test-csrf-token"`,
		"アットネーム (" + user.Atname + ") を入力してください",
		"data-confirm=",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに %qが含まれていない", want)
		}
	}
}

func TestNew_参加しているスペースがあれば削除できない理由と一覧を表示する(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	user := newUser(t, db, "new-blocked", "account-deletion-new-blocked")

	rr := httptest.NewRecorder()
	setupHandler(t, db, &recordingJobInserter{}).New(rr, newRequest(t, http.MethodGet, newPath, user, nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("Cache-Control"); got != "private, no-store" {
		t.Errorf("Cache-Control = %q、期待値 = private, no-store", got)
	}
	body := rr.Body.String()
	for _, want := range []string{"このアカウントは削除できません", `href="/s/account-deletion-new-blocked"`, "参加中のスペース new-blocked"} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに %qが含まれていない", want)
		}
	}
	if strings.Contains(body, `name="atname"`) {
		t.Error("削除できないのに削除のフォームが表示された")
	}
}

func TestCreate_アットネームが一致すれば削除してトップへ送る(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	user := newUser(t, db, "create-ok", "")
	inserter := &recordingJobInserter{}

	rr := httptest.NewRecorder()
	setupHandler(t, db, inserter).Create(rr, newRequest(t, http.MethodPost, createPath, user, url.Values{"atname": {user.Atname}}))

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusSeeOther)
	}
	if got := rr.Header().Get("Location"); got != "/" {
		t.Errorf("Location = %q、期待値 = /", got)
	}
	if !isDiscarded(t, db, user.ID) {
		t.Error("ユーザーが論理削除されていない")
	}
	if len(inserter.args) != 1 {
		t.Fatalf("投入したジョブ = %d件、期待値 = 1件", len(inserter.args))
	}
	if args, ok := inserter.args[0].(dispatcher.DeleteAccountArgs); !ok || args.UserID != string(user.ID) {
		t.Errorf("投入したジョブ = %+v、ユーザー%sの物理削除を期待", inserter.args[0], user.ID)
	}

	var sessionDeleted, flashSet bool
	for _, c := range rr.Result().Cookies() {
		if c.Name == session.CookieName && c.MaxAge < 0 {
			sessionDeleted = true
		}
		if c.Name == session.FlashCookieName && c.Value != "" {
			flashSet = true
		}
	}
	if !sessionDeleted {
		t.Error("セッションのCookieが削除されていない")
	}
	if !flashSet {
		t.Error("フラッシュが設定されていない")
	}
}

func TestCreate_削除できなければ確認画面を描き直す(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()

	// 誤りがアットネームの欄だけのときはその欄へ、項目に結び付かない誤りのときは
	// エラーの要約を包むフォーカスできる要素へ、フォーカスを移す
	tests := []struct {
		name      string
		suffix    string
		space     string
		atname    string
		wantText  string
		wantFocus string
	}{
		{name: "アットネームが一致しない", suffix: "create-mismatch", atname: "someone_else", wantText: "アットネームが間違っています", wantFocus: `id="atname"`},
		{name: "参加しているスペースがある", suffix: "create-blocked", space: "account-deletion-create-blocked", wantText: "参加しているスペースがあるため、アカウントを削除できません", wantFocus: `tabindex="-1"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := newUser(t, db, tt.suffix, tt.space)
			atname := tt.atname
			if atname == "" {
				atname = user.Atname
			}
			inserter := &recordingJobInserter{}

			rr := httptest.NewRecorder()
			setupHandler(t, db, inserter).Create(rr, newRequest(t, http.MethodPost, createPath, user, url.Values{"atname": {atname}}))

			if rr.Code != http.StatusUnprocessableEntity {
				t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusUnprocessableEntity)
			}
			if got := rr.Header().Get("Cache-Control"); got != "private, no-store" {
				t.Errorf("Cache-Control = %q、期待値 = private, no-store", got)
			}
			if !strings.Contains(rr.Body.String(), tt.wantText) {
				t.Errorf("レスポンスに %qが含まれていない", tt.wantText)
			}
			assertFocus(t, rr.Body.String(), tt.wantFocus)
			if isDiscarded(t, db, user.ID) {
				t.Error("削除できないのにユーザーが論理削除された")
			}
			if len(inserter.args) != 0 {
				t.Errorf("投入したジョブ = %d件、期待値 = 0件", len(inserter.args))
			}
		})
	}
}

func TestCreate_二重送信は成功を知らせない(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	user := newUser(t, db, "create-duplicate", "")
	inserter := &recordingJobInserter{}
	h := setupHandler(t, db, inserter)
	form := url.Values{"atname": {user.Atname}}
	first := httptest.NewRecorder()
	h.Create(first, newRequest(t, http.MethodPost, createPath, user, form))
	if first.Code != http.StatusSeeOther {
		t.Fatalf("先行要求のステータスコード = %d、期待値 = %d", first.Code, http.StatusSeeOther)
	}

	rr := httptest.NewRecorder()
	h.Create(rr, newRequest(t, http.MethodPost, createPath, user, form))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("二重送信のステータスコード = %d、期待値 = %d", rr.Code, http.StatusNotFound)
	}
	if got := rr.Header().Get("Location"); got != "" {
		t.Errorf("二重送信のLocation = %q、リダイレクトを期待しない", got)
	}
	for _, cookie := range rr.Result().Cookies() {
		if cookie.Name == session.FlashCookieName {
			t.Error("二重送信に成功のフラッシュが設定された")
		}
	}
	if len(inserter.args) != 1 {
		t.Errorf("投入したジョブ = %d件、期待値 = 1件", len(inserter.args))
	}
}

func TestCreate_一致しないアットネームは入力欄に戻す(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	user := newUser(t, db, "create-echo", "")

	rr := httptest.NewRecorder()
	setupHandler(t, db, &recordingJobInserter{}).Create(rr, newRequest(t, http.MethodPost, createPath, user, url.Values{"atname": {"typo_name"}}))

	body := rr.Body.String()
	if !strings.Contains(body, `value="typo_name"`) {
		t.Error("送信したアットネームが入力欄に戻っていない")
	}
	if !strings.Contains(body, `aria-invalid="true"`) {
		t.Error("入力欄に誤りが示されていない")
	}
}
