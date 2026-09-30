package space_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/config"
	spacehandler "github.com/wikinoapp/wikino/go/internal/handler/space"
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

// newSpaceFormRequestはスペース作成のリクエストを作る。userIDが空なら未ログインのリクエスト
// にし、formがnilならボディを持たないリクエストにする。
func newSpaceFormRequest(t *testing.T, method, path string, userID model.UserID, form map[string]string) *http.Request {
	t.Helper()

	var req *http.Request
	if form == nil {
		req = httptest.NewRequest(method, path, nil)
	} else {
		values := url.Values{}
		for key, value := range form {
			values.Set(key, value)
		}
		req = httptest.NewRequest(method, path, strings.NewReader(values.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	ctx := middleware.SetCSRFTokenToContext(req.Context(), "test-csrf-token")
	if userID != "" {
		ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID, Atname: "space-user"})
	}
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	return req.WithContext(ctx)
}

// setupCreateHandlerは共有プールを使うハンドラーを組み立てる。作成のユースケースは自身で
// トランザクションを開くため、これらのテストが用意するフィクスチャはトランザクションに閉じ込めず
// コミットする。
func setupCreateHandler(t *testing.T, db *sql.DB) *spacehandler.Handler {
	t.Helper()

	cfg := &config.Config{Env: "test", Domain: "localhost"}
	queries := query.New(db)
	spaceRepo := repository.NewSpaceRepository(queries)
	spaceMemberRepo := repository.NewSpaceMemberRepository(queries)

	return spacehandler.NewHandler(
		cfg,
		session.NewFlashManager("", false, true),
		nil,
		usecase.NewCreateSpaceUsecase(db, spaceRepo, spaceMemberRepo, validator.NewSpaceCreateValidator(spaceRepo)),
	)
}

func TestNew_フォームを表示する(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail("space-new-form@example.com").
		WithAtname("space_new_form").
		Build()

	req := newSpaceFormRequest(t, http.MethodGet, "/spaces/new", userID, nil)
	rr := httptest.NewRecorder()
	setupCreateHandler(t, db).New(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("X-Robots-Tag"); got != "noindex" {
		t.Errorf("X-Robots-Tag = %q、期待値 = %q", got, "noindex")
	}

	body := rr.Body.String()
	for _, want := range []string{
		"<title>新規スペース | Wikino</title>",
		`action="/spaces"`,
		`name="csrf_token" value="test-csrf-token"`,
		`name="identifier"`,
		`name="name"`,
		"20文字以内で半角英数字とハイフンのみ使用できます",
		`href="/home"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに%qが含まれていない", want)
		}
	}
}

func TestNew_未ログインならサインイン画面へリダイレクトする(t *testing.T) {
	t.Parallel()

	req := newSpaceFormRequest(t, http.MethodGet, "/spaces/new", "", nil)
	rr := httptest.NewRecorder()
	setupCreateHandler(t, testutil.GetTestDB()).New(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusFound)
	}
	if got := rr.Header().Get("Location"); got != "/sign_in" {
		t.Errorf("Location = %q、期待値 = %q", got, "/sign_in")
	}
}

// findSpaceは識別子でスペースを探す。見つからなければnilを返す。
func findSpace(t *testing.T, db *sql.DB, identifier model.SpaceIdentifier) *model.Space {
	t.Helper()

	space, err := repository.NewSpaceRepository(query.New(db)).FindByIdentifier(context.Background(), identifier)
	if err != nil {
		t.Fatalf("スペースの取得に失敗: %v", err)
	}
	return space
}
