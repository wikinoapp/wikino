package space_deletion_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	spacehandler "github.com/wikinoapp/wikino/go/internal/handler/space"
	spacedeletion "github.com/wikinoapp/wikino/go/internal/handler/space_deletion"
	spacesettings "github.com/wikinoapp/wikino/go/internal/handler/space_settings"
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

// autofocusPatternは、autofocusを持つ要素の開始タグに一致する。画面を開いたときにどこへ
// フォーカスするかを確かめるために使う。
var autofocusPattern = regexp.MustCompile(`<[a-z]+\b[^>]*\bautofocus\b[^>]*>`)

// recordingJobInserterは投入されたジョブを記録する。同時に送った削除からも呼ばれるため、排他して記録する。
type recordingJobInserter struct {
	mu   sync.Mutex
	args []river.JobArgs
}

func (r *recordingJobInserter) Insert(_ context.Context, args river.JobArgs, _ *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.args = append(r.args, args)
	return &rivertype.JobInsertResult{}, nil
}

// deletionFixtureは削除のテストが使うスペースとメンバー。削除は自身でトランザクションを開くため、
// フィクスチャはDBに直接コミットする。
type deletionFixture struct {
	identifier string
	spaceName  string
	spaceID    model.SpaceID
	userID     model.UserID
	topicID    model.TopicID
}

// setupDeletionFixtureは、スペースとトピック1つと、渡したロールを持つメンバー1人を用意する。
func setupDeletionFixture(t *testing.T, identifier string, role model.SpaceRole) deletionFixture {
	t.Helper()

	db := testutil.GetTestDB()
	userID := testutil.NewUserBuilderDB(t, db).
		WithEmail(identifier + "@example.com").
		WithAtname(strings.ReplaceAll(identifier, "-", "_")).
		Build()
	spaceName := "削除するスペース"
	spaceID := testutil.NewSpaceBuilderDB(t, db).WithIdentifier(identifier).WithName(spaceName).Build()
	testutil.NewSpaceMemberBuilderDB(t, db).WithSpaceID(spaceID).WithUserID(userID).WithRole(role).Build()
	topicID := testutil.NewTopicBuilderDB(t, db).WithSpaceID(spaceID).WithNumber(1).WithName("トピック").Build()

	return deletionFixture{identifier: identifier, spaceName: spaceName, spaceID: spaceID, userID: userID, topicID: topicID}
}

// newRequestはchiのURLパラメータ・ログイン中のユーザー・画面を描画するロケールを載せたリクエストを
// 組み立てる。userIDが空のときはユーザーを載せない。
func newRequest(t *testing.T, method, path, identifier string, userID model.UserID, form url.Values) *http.Request {
	t.Helper()

	var req *http.Request
	if form != nil {
		req = httptest.NewRequest(method, path, strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("space_identifier", identifier)

	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	if userID != "" {
		ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID, Atname: "space-deletion-user"})
	}
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	return req.WithContext(ctx)
}

func testConfig() *config.Config {
	return &config.Config{Env: "test", Domain: "localhost"}
}

// setupHandlerは、テスト用のDBと渡したジョブ投入器を使うハンドラーを組み立てる。
func setupHandler(t *testing.T, inserter dispatcher.JobInserter) *spacedeletion.Handler {
	t.Helper()

	db := testutil.GetTestDB()
	queries := query.New(db)
	spaceRepo := repository.NewSpaceRepository(queries)
	spaceMemberRepo := repository.NewSpaceMemberRepository(queries)

	return spacedeletion.NewHandler(
		testConfig(),
		session.NewFlashManager("", false, true),
		usecase.NewGetSpaceDeletionNewUsecase(spaceRepo, spaceMemberRepo),
		usecase.NewDiscardSpaceUsecase(db, spaceRepo, spaceMemberRepo, repository.NewTopicRepository(queries), dispatcher.NewDispatcher(inserter, nil), validator.NewSpaceDeletionCreateValidator()),
	)
}

func newPath(identifier string) string {
	return "/s/" + identifier + "/settings/deletion/new"
}

func createPath(identifier string) string {
	return "/s/" + identifier + "/settings/deletion"
}

// isSpaceDiscardedはスペースが論理削除されているかを返す。
func isSpaceDiscarded(t *testing.T, spaceID model.SpaceID) bool {
	t.Helper()

	var discarded bool
	if err := testutil.GetTestDB().QueryRowContext(t.Context(), "SELECT discarded_at IS NOT NULL FROM spaces WHERE id = $1", string(spaceID)).Scan(&discarded); err != nil {
		t.Fatalf("スペースの確認に失敗: %v", err)
	}
	return discarded
}

func TestNew_削除できるメンバーに確認画面を表示する(t *testing.T) {
	t.Parallel()

	f := setupDeletionFixture(t, "space-deletion-new", model.SpaceRoleAdmin)

	for _, method := range []string{http.MethodGet, http.MethodHead} {
		t.Run(method, func(t *testing.T) {
			t.Parallel()

			rr := httptest.NewRecorder()
			setupHandler(t, &recordingJobInserter{}).New(rr, newRequest(t, method, newPath(f.identifier), f.identifier, f.userID, nil))

			if rr.Code != http.StatusOK {
				t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
			}
			if got := rr.Header().Get("X-Robots-Tag"); got != "noindex" {
				t.Errorf("X-Robots-Tag = %q、期待値 = %q", got, "noindex")
			}
			if method == http.MethodHead {
				return
			}

			body := rr.Body.String()
			for _, want := range []string{
				"<title>スペースの削除 | " + f.spaceName + "</title>",
				`action="` + createPath(f.identifier) + `"`,
				`method="POST"`,
				`name="csrf_token"`,
				`<label class="label" for="space_name">`,
				"スペース名 (" + f.spaceName + ") を入力してください",
				"トピック・ページ・添付ファイル・エクスポートもすべて削除されます",
				`href="/s/` + f.identifier + `/settings"`,
				`aria-current="page"`,
			} {
				if !strings.Contains(body, want) {
					t.Errorf("レスポンスに%qが含まれていない", want)
				}
			}
			if focused := autofocusPattern.FindAllString(body, -1); len(focused) != 0 {
				t.Errorf("初回表示でフォーカスする要素 = %v、期待値 = なし", focused)
			}
			if strings.Contains(body, `href="#space_name"`) {
				t.Error("初回表示にエラー要約のリンクが出ている")
			}
		})
	}
}

func TestNew_必須表示が入力欄のラベルに含まれる(t *testing.T) {
	t.Parallel()
	f := setupDeletionFixture(t, "space-deletion-required", model.SpaceRoleAdmin)
	labelPattern := regexp.MustCompile(`<label[^>]*for="space_name"[^>]*>.*?</label>`)
	for _, tt := range []struct {
		lang string
		want string
	}{
		{lang: i18n.LangJa, want: "必須"},
		{lang: i18n.LangEn, want: "Required"},
	} {
		t.Run(tt.lang, func(t *testing.T) {
			t.Parallel()
			req := newRequest(t, http.MethodGet, newPath(f.identifier), f.identifier, f.userID, nil)
			req = req.WithContext(i18n.SetLocale(req.Context(), tt.lang))
			rr := httptest.NewRecorder()
			setupHandler(t, &recordingJobInserter{}).New(rr, req)
			if rr.Code != http.StatusOK {
				t.Fatalf("ステータスコード = %d、期待値 = 200", rr.Code)
			}
			label := labelPattern.FindString(rr.Body.String())
			if !strings.Contains(label, tt.want) {
				t.Errorf("入力欄に関連付いたラベル = %q、%qを含むことを期待", label, tt.want)
			}
		})
	}
}

func TestNew_到達できない場合は404が返る(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		suffix   string
		role     model.SpaceRole
		outsider bool
	}{
		{name: "space:deleteを持たない編集者", suffix: "editor", role: model.SpaceRoleEditor},
		{name: "space:deleteを持たない閲覧者", suffix: "viewer", role: model.SpaceRoleViewer},
		{name: "メンバーでない閲覧者", suffix: "outsider", role: model.SpaceRoleAdmin, outsider: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			identifier := "space-deletion-new-" + tt.suffix
			f := setupDeletionFixture(t, identifier, tt.role)
			userID := f.userID
			if tt.outsider {
				userID = testutil.NewUserBuilderDB(t, testutil.GetTestDB()).
					WithEmail(identifier + "-other@example.com").
					WithAtname(strings.ReplaceAll(identifier, "-", "_") + "_other").
					Build()
			}

			rr := httptest.NewRecorder()
			setupHandler(t, &recordingJobInserter{}).New(rr, newRequest(t, http.MethodGet, newPath(identifier), identifier, userID, nil))

			if rr.Code != http.StatusNotFound {
				t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusNotFound)
			}
		})
	}
}

func TestNewとCreate_未ログインならログイン画面へリダイレクトする(t *testing.T) {
	t.Parallel()

	f := setupDeletionFixture(t, "space-deletion-anon", model.SpaceRoleAdmin)
	h := setupHandler(t, &recordingJobInserter{})

	for _, tc := range []struct {
		name    string
		handler http.HandlerFunc
		req     *http.Request
	}{
		{name: "確認画面", handler: h.New, req: newRequest(t, http.MethodGet, newPath(f.identifier), f.identifier, "", nil)},
		{name: "削除", handler: h.Create, req: newRequest(t, http.MethodPost, createPath(f.identifier), f.identifier, "", url.Values{"space_name": {f.spaceName}})},
	} {
		rr := httptest.NewRecorder()
		tc.handler(rr, tc.req)
		if rr.Code != http.StatusFound || rr.Header().Get("Location") != "/sign_in" {
			t.Errorf("%s: ステータスコード = %d・Location = %q、期待値 = 302・/sign_in", tc.name, rr.Code, rr.Header().Get("Location"))
		}
	}
	if isSpaceDiscarded(t, f.spaceID) {
		t.Error("未ログインの送信でスペースが削除された")
	}
}

func TestCreate_名前が一致すればスペースを削除してホームへリダイレクトする(t *testing.T) {
	t.Parallel()

	f := setupDeletionFixture(t, "space-deletion-create", model.SpaceRoleAdmin)
	inserter := &recordingJobInserter{}

	rr := httptest.NewRecorder()
	setupHandler(t, inserter).Create(rr, newRequest(t, http.MethodPost, createPath(f.identifier), f.identifier, f.userID, url.Values{"space_name": {f.spaceName}}))

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusSeeOther)
	}
	if location := rr.Header().Get("Location"); location != "/" {
		t.Errorf("Location = %q、期待値 = %q", location, "/")
	}
	if !hasFlashCookie(rr) {
		t.Error("フラッシュが設定されていない")
	}
	if !isSpaceDiscarded(t, f.spaceID) {
		t.Error("スペースが論理削除されていない")
	}
	if len(inserter.args) != 1 {
		t.Fatalf("投入したジョブの数 = %d、期待値 = 1", len(inserter.args))
	}
	if args, ok := inserter.args[0].(dispatcher.DeleteSpaceArgs); !ok || args.SpaceID != string(f.spaceID) {
		t.Errorf("投入したジョブ = %+v、スペース%sの物理削除を期待", inserter.args[0], f.spaceID)
	}

	// 削除したスペースは、スペースの画面・設定のトップ・この確認画面のどれからも見えなくなる
	db := testutil.GetTestDB()
	queries := query.New(db)
	spaceRepo := repository.NewSpaceRepository(queries)
	spaceMemberRepo := repository.NewSpaceMemberRepository(queries)
	featureFlagRepo := repository.NewFeatureFlagRepository(queries)
	spaceShow := spacehandler.NewHandler(testConfig(), session.NewFlashManager("", false, true), usecase.NewGetSpaceShowUsecase(
		spaceRepo, spaceMemberRepo, repository.NewPageRepository(queries), repository.NewTopicRepository(queries), repository.NewTopicMemberRepository(queries), featureFlagRepo,
	), nil)
	settingsShow := spacesettings.NewHandler(testConfig(), usecase.NewGetSpaceSettingsUsecase(spaceRepo, spaceMemberRepo, featureFlagRepo))

	for _, tc := range []struct {
		name    string
		handler http.HandlerFunc
		path    string
	}{
		{name: "スペースの画面", handler: spaceShow.Show, path: "/s/" + f.identifier},
		{name: "設定のトップ", handler: settingsShow.Show, path: "/s/" + f.identifier + "/settings"},
		{name: "削除の確認画面", handler: setupHandler(t, inserter).New, path: newPath(f.identifier)},
	} {
		rr := httptest.NewRecorder()
		tc.handler(rr, newRequest(t, http.MethodGet, tc.path, f.identifier, f.userID, nil))
		if rr.Code != http.StatusNotFound {
			t.Errorf("%s: ステータスコード = %d、期待値 = %d", tc.name, rr.Code, http.StatusNotFound)
		}
	}
}

func TestCreate_名前が一致しなければ確認画面を描画し直す(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		spaceName string
		wantMsg   string
	}{
		{name: "名前が違う", spaceName: "別のスペース", wantMsg: "スペース名が間違っています"},
		{name: "名前が空", spaceName: "", wantMsg: "入力してください"},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := setupDeletionFixture(t, "space-deletion-mismatch-"+string(rune('a'+i)), model.SpaceRoleAdmin)
			inserter := &recordingJobInserter{}

			rr := httptest.NewRecorder()
			setupHandler(t, inserter).Create(rr, newRequest(t, http.MethodPost, createPath(f.identifier), f.identifier, f.userID, url.Values{"space_name": {tt.spaceName}}))

			if rr.Code != http.StatusUnprocessableEntity {
				t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusUnprocessableEntity)
			}
			body := rr.Body.String()
			for _, want := range []string{tt.wantMsg, `aria-invalid="true"`, `value="` + tt.spaceName + `"`, `action="` + createPath(f.identifier) + `"`, "入力内容を確認してください", `href="#space_name"`} {
				if !strings.Contains(body, want) {
					t.Errorf("レスポンスに%qが含まれていない", want)
				}
			}
			focused := autofocusPattern.FindAllString(body, -1)
			if len(focused) != 1 || !strings.Contains(focused[0], `id="space_name"`) {
				t.Errorf("フォーカス先 = %v、期待値のID = %q", focused, "space_name")
			}
			if isSpaceDiscarded(t, f.spaceID) {
				t.Error("名前が一致しないのにスペースが削除された")
			}
			if len(inserter.args) != 0 {
				t.Errorf("投入したジョブの数 = %d、期待値 = 0", len(inserter.args))
			}
		})
	}
}

func TestCreate_space_deleteを持たないメンバーには404が返る(t *testing.T) {
	t.Parallel()

	f := setupDeletionFixture(t, "space-deletion-create-editor", model.SpaceRoleEditor)
	inserter := &recordingJobInserter{}

	rr := httptest.NewRecorder()
	setupHandler(t, inserter).Create(rr, newRequest(t, http.MethodPost, createPath(f.identifier), f.identifier, f.userID, url.Values{"space_name": {f.spaceName}}))

	if rr.Code != http.StatusNotFound {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusNotFound)
	}
	if isSpaceDiscarded(t, f.spaceID) {
		t.Error("space:deleteを持たないメンバーがスペースを削除できた")
	}
	if len(inserter.args) != 0 {
		t.Errorf("投入したジョブの数 = %d、期待値 = 0", len(inserter.args))
	}
}

func TestCreate_同時に2回送っても1回だけ削除する(t *testing.T) {
	t.Parallel()

	f := setupDeletionFixture(t, "space-deletion-concurrent", model.SpaceRoleAdmin)
	inserter := &recordingJobInserter{}
	h := setupHandler(t, inserter)

	codes := make([]int, 2)
	var wg sync.WaitGroup
	for i := range codes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rr := httptest.NewRecorder()
			h.Create(rr, newRequest(t, http.MethodPost, createPath(f.identifier), f.identifier, f.userID, url.Values{"space_name": {f.spaceName}}))
			codes[i] = rr.Code
		}()
	}
	wg.Wait()

	seeOther, notFound := 0, 0
	for _, code := range codes {
		switch code {
		case http.StatusSeeOther:
			seeOther++
		case http.StatusNotFound:
			notFound++
		}
	}
	if seeOther != 1 || notFound != 1 {
		t.Errorf("ステータスコード = %v、期待値 = 303と404が1つずつ", codes)
	}
	if len(inserter.args) != 1 {
		t.Errorf("投入したジョブの数 = %d、期待値 = 1", len(inserter.args))
	}
}

func TestCreate_CSRFトークンが無いと403が返り削除されない(t *testing.T) {
	t.Parallel()

	f := setupDeletionFixture(t, "space-deletion-csrf", model.SpaceRoleAdmin)
	inserter := &recordingJobInserter{}

	rr := httptest.NewRecorder()
	csrf := middleware.NewCSRF(&config.Config{SessionHTTPOnly: true})
	csrf.Middleware(http.HandlerFunc(setupHandler(t, inserter).Create)).ServeHTTP(rr, newRequest(t, http.MethodPost, createPath(f.identifier), f.identifier, f.userID, url.Values{"space_name": {f.spaceName}}))

	if rr.Code != http.StatusForbidden {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusForbidden)
	}
	if isSpaceDiscarded(t, f.spaceID) {
		t.Error("CSRFトークンの無い削除でスペースが削除された")
	}
}

// hasFlashCookieはレスポンスがフラッシュのクッキーを設定したかを返す。
func hasFlashCookie(rr *httptest.ResponseRecorder) bool {
	for _, c := range rr.Result().Cookies() {
		if c.Name == session.FlashCookieName && c.Value != "" {
			return true
		}
	}
	return false
}
