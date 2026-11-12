package topic_deletion_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/dispatcher"
	topicdeletion "github.com/wikinoapp/wikino/go/internal/handler/topic_deletion"
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

const deletionTopicName = "削除するトピック"

// autofocusPatternは、autofocusを持つ要素の開始タグに一致する。画面を開いたときにどこへ
// フォーカスするかを確かめるために使う。
var autofocusPattern = regexp.MustCompile(`<[a-z]+\b[^>]*\bautofocus\b[^>]*>`)

// recordingJobInserterは投入されたジョブを記録する。
type recordingJobInserter struct {
	args []river.JobArgs
}

func (r *recordingJobInserter) Insert(_ context.Context, args river.JobArgs, _ *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	r.args = append(r.args, args)
	return &rivertype.JobInsertResult{}, nil
}

// newDeletionRequestはchiのURLパラメータ・ログイン中のユーザー・画面を描画するロケールを
// 載せたリクエストを組み立てる。userIDが空のときはユーザーを載せない。
func newDeletionRequest(t *testing.T, method, path, identifier, topicNumber string, userID model.UserID, form url.Values) *http.Request {
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
	rctx.URLParams.Add("topic_number", topicNumber)

	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	if userID != "" {
		ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID, Atname: "topic-deletion-user"})
	}
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	return req.WithContext(ctx)
}

// setupDeletionHandlerは渡したクエリとジョブ投入器を使うハンドラーを組み立てる。
func setupDeletionHandler(t *testing.T, queries *query.Queries, inserter dispatcher.JobInserter) *topicdeletion.Handler {
	t.Helper()

	cfg := &config.Config{Env: "test", Domain: "localhost"}
	spaceRepo := repository.NewSpaceRepository(queries)
	spaceMemberRepo := repository.NewSpaceMemberRepository(queries)
	topicRepo := repository.NewTopicRepository(queries)
	topicMemberRepo := repository.NewTopicMemberRepository(queries)

	return topicdeletion.NewHandler(
		cfg,
		session.NewFlashManager("", false, true),
		usecase.NewGetTopicDeletionNewUsecase(spaceRepo, spaceMemberRepo, topicRepo, topicMemberRepo),
		usecase.NewDiscardTopicUsecase(spaceRepo, spaceMemberRepo, topicRepo, topicMemberRepo, dispatcher.NewDispatcher(inserter), validator.NewTopicDeletionCreateValidator()),
	)
}

// deletionTopicはスペースとトピック番号1のトピック、渡したロールを持つメンバー1人を用意し、
// メンバーのユーザーとトピックを返す。roleが空のときはビルダーの既定 (管理者) になる。
func deletionTopic(t *testing.T, tx *sql.Tx, identifier string, role model.SpaceRole) (model.UserID, model.SpaceID, model.TopicID) {
	t.Helper()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail(identifier + "@example.com").
		WithAtname(strings.ReplaceAll(identifier, "-", "_")).
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier(identifier).
		WithName("削除のスペース").
		Build()
	memberBuilder := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID)
	if role != "" {
		memberBuilder = memberBuilder.WithRole(role)
	}
	memberBuilder.Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName(deletionTopicName).
		Build()

	return userID, spaceID, topicID
}

func newPath(identifier string) string {
	return "/s/" + identifier + "/topics/1/settings/deletion/new"
}

func createPath(identifier string) string {
	return "/s/" + identifier + "/topics/1/settings/deletion"
}

func TestNew_削除できるメンバーに確認画面を表示する(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	identifier := "topic-deletion-new"
	userID, _, _ := deletionTopic(t, tx, identifier, "")

	req := newDeletionRequest(t, http.MethodGet, newPath(identifier), identifier, "1", userID, nil)
	rr := httptest.NewRecorder()
	setupDeletionHandler(t, queries, &recordingJobInserter{}).New(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("X-Robots-Tag"); got != "noindex" {
		t.Errorf("X-Robots-Tag = %q、期待値 = %q", got, "noindex")
	}

	body := rr.Body.String()
	for _, want := range []string{
		"<title>トピックの削除 | " + deletionTopicName + " | 削除のスペース</title>",
		`action="` + createPath(identifier) + `"`,
		`name="csrf_token"`,
		`<label class="label" for="topic_name">`,
		"トピック名 (" + deletionTopicName + ") を入力してください",
		"ページもすべて削除されます",
		`href="/s/` + identifier + `/topics/1/settings"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに%qが含まれていない", want)
		}
	}
	if focused := autofocusPattern.FindAllString(body, -1); len(focused) != 0 {
		t.Errorf("初回表示でフォーカスする要素 = %v、期待値 = なし", focused)
	}
}

func TestNew_到達できない場合は404が返る(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		identifier  string
		role        model.SpaceRole
		topicNumber string
	}{
		{name: "topic:deleteを持たない編集者", identifier: "topic-deletion-new-editor", role: model.SpaceRoleEditor, topicNumber: "1"},
		{name: "存在しないトピック", identifier: "topic-deletion-new-missing", topicNumber: "999"},
		{name: "数値でないトピック番号", identifier: "topic-deletion-new-nan", topicNumber: "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			queries := testutil.QueriesWithTx(tx)
			userID, _, _ := deletionTopic(t, tx, tt.identifier, tt.role)

			req := newDeletionRequest(t, http.MethodGet, newPath(tt.identifier), tt.identifier, tt.topicNumber, userID, nil)
			rr := httptest.NewRecorder()
			setupDeletionHandler(t, queries, &recordingJobInserter{}).New(rr, req)

			if rr.Code != http.StatusNotFound {
				t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusNotFound)
			}
		})
	}
}

func TestNew_未ログインならログイン画面へリダイレクトする(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	identifier := "topic-deletion-new-anon"
	deletionTopic(t, tx, identifier, "")

	req := newDeletionRequest(t, http.MethodGet, newPath(identifier), identifier, "1", "", nil)
	rr := httptest.NewRecorder()
	setupDeletionHandler(t, queries, &recordingJobInserter{}).New(rr, req)

	if rr.Code != http.StatusFound || rr.Header().Get("Location") != "/sign_in" {
		t.Errorf("ステータスコード = %d・Location = %q、期待値 = 302・/sign_in", rr.Code, rr.Header().Get("Location"))
	}
}

func TestCreate_名前が一致すればトピックを削除してスペースの画面へリダイレクトする(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	identifier := "topic-deletion-create"
	userID, spaceID, topicID := deletionTopic(t, tx, identifier, "")
	inserter := &recordingJobInserter{}

	form := url.Values{"topic_name": {deletionTopicName}}
	req := newDeletionRequest(t, http.MethodPost, createPath(identifier), identifier, "1", userID, form)
	rr := httptest.NewRecorder()
	setupDeletionHandler(t, queries, inserter).Create(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusSeeOther)
	}
	if location := rr.Header().Get("Location"); location != "/s/"+identifier {
		t.Errorf("Location = %q、期待値 = %q", location, "/s/"+identifier)
	}
	if !hasFlashCookie(rr) {
		t.Error("フラッシュが設定されていない")
	}

	// 論理削除したトピックは直ちに見えなくなる
	topic, err := repository.NewTopicRepository(queries).FindBySpaceAndNumber(t.Context(), spaceID, 1)
	if err != nil {
		t.Fatalf("トピックの取得に失敗: %v", err)
	}
	if topic != nil {
		t.Error("削除したトピックがまだ見える")
	}
	if len(inserter.args) != 1 {
		t.Fatalf("投入したジョブの数 = %d、期待値 = 1", len(inserter.args))
	}
	if args, ok := inserter.args[0].(dispatcher.DeleteTopicArgs); !ok || args.TopicID != string(topicID) {
		t.Errorf("投入したジョブ = %+v、トピック%sの物理削除を期待", inserter.args[0], topicID)
	}
}

func TestCreate_名前が一致しなければ確認画面を描画し直す(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		topicName string
		wantMsg   string
	}{
		{name: "名前が違う", topicName: "別のトピック", wantMsg: "トピック名が間違っています"},
		{name: "名前が空", topicName: "", wantMsg: "入力してください"},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			queries := testutil.QueriesWithTx(tx)
			identifier := "topic-deletion-mismatch-" + string(rune('a'+i))
			userID, spaceID, _ := deletionTopic(t, tx, identifier, "")
			inserter := &recordingJobInserter{}

			form := url.Values{"topic_name": {tt.topicName}}
			req := newDeletionRequest(t, http.MethodPost, createPath(identifier), identifier, "1", userID, form)
			rr := httptest.NewRecorder()
			setupDeletionHandler(t, queries, inserter).Create(rr, req)

			if rr.Code != http.StatusUnprocessableEntity {
				t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusUnprocessableEntity)
			}
			body := rr.Body.String()
			for _, want := range []string{tt.wantMsg, `aria-invalid="true"`, `value="` + tt.topicName + `"`} {
				if !strings.Contains(body, want) {
					t.Errorf("レスポンスに%qが含まれていない", want)
				}
			}
			focused := autofocusPattern.FindAllString(body, -1)
			if len(focused) != 1 || !strings.Contains(focused[0], `id="topic_name"`) {
				t.Errorf("フォーカス先 = %v、期待値のID = %q", focused, "topic_name")
			}

			topic, err := repository.NewTopicRepository(queries).FindBySpaceAndNumber(t.Context(), spaceID, 1)
			if err != nil {
				t.Fatalf("トピックの取得に失敗: %v", err)
			}
			if topic == nil {
				t.Error("名前が一致しないのにトピックが削除された")
			}
			if len(inserter.args) != 0 {
				t.Errorf("投入したジョブの数 = %d、期待値 = 0", len(inserter.args))
			}
		})
	}
}

func TestCreate_topic_deleteを持たないメンバーには404が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	identifier := "topic-deletion-create-editor"
	userID, spaceID, _ := deletionTopic(t, tx, identifier, model.SpaceRoleEditor)
	inserter := &recordingJobInserter{}

	form := url.Values{"topic_name": {deletionTopicName}}
	req := newDeletionRequest(t, http.MethodPost, createPath(identifier), identifier, "1", userID, form)
	rr := httptest.NewRecorder()
	setupDeletionHandler(t, queries, inserter).Create(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusNotFound)
	}
	topic, err := repository.NewTopicRepository(queries).FindBySpaceAndNumber(t.Context(), spaceID, 1)
	if err != nil {
		t.Fatalf("トピックの取得に失敗: %v", err)
	}
	if topic == nil {
		t.Error("topic:deleteを持たないメンバーがトピックを削除できた")
	}
	if len(inserter.args) != 0 {
		t.Errorf("投入したジョブの数 = %d、期待値 = 0", len(inserter.args))
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
