package topic_settings_general_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/config"
	settingsgeneral "github.com/wikinoapp/wikino/go/internal/handler/topic_settings_general"
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

// settingsGeneralPath is the path of the screen these tests address.
//
// [Ja] settingsGeneralPath はこれらのテストが対象にする画面のパス。
func settingsGeneralPath(identifier, topicNumber string) string {
	return "/s/" + identifier + "/topics/" + topicNumber + "/settings/general"
}

// newSettingsGeneralRequest builds a request for the general settings screen, carrying the chi URL
// parameters, the CSRF token, the signed-in user and the locale the screen is rendered in. A zero
// userID leaves the user off, which is how a test reaches the signed-out path.
//
// [Ja] newSettingsGeneralRequest は一般設定の画面向けに、chi の URL パラメータ・CSRF トークン・
// ログイン中のユーザー・画面を描画するロケールを載せたリクエストを組み立てる。userID が空のときは
// ユーザーを載せず、テストが未ログインの経路へ到達できるようにする。
func newSettingsGeneralRequest(t *testing.T, method, identifier, topicNumber string, userID model.UserID, form map[string]string) *http.Request {
	t.Helper()

	path := settingsGeneralPath(identifier, topicNumber)

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

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("space_identifier", identifier)
	rctx.URLParams.Add("topic_number", topicNumber)

	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	if userID != "" {
		ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID, Atname: "topic-user"})
	}
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	return req.WithContext(ctx)
}

// setupSettingsGeneralHandler builds a handler against the given pool. Neither the screen nor the
// save opens a transaction of its own, so the tests may hold their fixtures in one.
//
// [Ja] setupSettingsGeneralHandler は渡したプールを使うハンドラーを組み立てる。画面も保存処理も
// 自身でトランザクションを開かないため、テストはフィクスチャをトランザクションに閉じ込めてよい。
func setupSettingsGeneralHandler(t *testing.T, queries *query.Queries) *settingsgeneral.Handler {
	t.Helper()

	cfg := &config.Config{Env: "test", Domain: "localhost"}
	spaceRepo := repository.NewSpaceRepository(queries)
	spaceMemberRepo := repository.NewSpaceMemberRepository(queries)
	topicRepo := repository.NewTopicRepository(queries)
	topicMemberRepo := repository.NewTopicMemberRepository(queries)

	return settingsgeneral.NewHandler(
		cfg,
		session.NewFlashManager("", false, true),
		usecase.NewGetTopicSettingsGeneralUsecase(spaceRepo, spaceMemberRepo, topicRepo, topicMemberRepo),
		usecase.NewUpdateTopicUsecase(
			spaceRepo,
			spaceMemberRepo,
			topicRepo,
			topicMemberRepo,
			validator.NewTopicUpdateValidator(topicRepo),
		),
	)
}

// settingsGeneralSpace seeds a space holding one private topic, with one member of the given
// scopes, and returns what the tests address them by.
//
// [Ja] settingsGeneralSpace は非公開トピックを 1 つ持つスペースと、渡したスコープを持つメンバー
// 1 人を用意し、テストがそれらを指すための値を返す。
func settingsGeneralSpace(t *testing.T, tx *sql.Tx, identifier string, scopes []model.Scope) (model.UserID, model.SpaceID, model.TopicID) {
	t.Helper()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail(identifier + "@example.com").
		WithAtname(strings.ReplaceAll(identifier, "-", "_")).
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier(identifier).
		Build()
	memberBuilder := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID)
	if scopes != nil {
		memberBuilder = memberBuilder.WithScopes(scopes)
	}
	memberBuilder.Build()

	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("日報").
		WithDescription("毎日の記録").
		WithVisibility(1).
		Build()

	return userID, spaceID, topicID
}

func TestShow_保存済みの値が入ったフォームが表示される(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	identifier := "topic-general-show"
	userID, _, _ := settingsGeneralSpace(t, tx, identifier, nil)

	req := newSettingsGeneralRequest(t, http.MethodGet, identifier, "1", userID, nil)
	rr := httptest.NewRecorder()
	setupSettingsGeneralHandler(t, queries).Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	for _, want := range []string{
		`action="` + settingsGeneralPath(identifier, "1") + `"`,
		`name="_method" value="PATCH"`,
		`value="日報"`,
		`value="毎日の記録"`,
		"test-csrf-token",
		"基本情報",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("response does not contain %q", want)
		}
	}

	if tag := settingsGeneralInputTag(t, body, "visibility_private"); !strings.Contains(tag, "checked") {
		t.Errorf("input %q tag does not contain checked: %s", "visibility_private", tag)
	}
	if tag := settingsGeneralInputTag(t, body, "visibility_public"); strings.Contains(tag, "checked") {
		t.Errorf("input %q tag contains checked: %s", "visibility_public", tag)
	}
}

func TestShow_HEADでも200が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	identifier := "topic-general-head"
	userID, _, _ := settingsGeneralSpace(t, tx, identifier, nil)

	req := newSettingsGeneralRequest(t, http.MethodHead, identifier, "1", userID, nil)
	rr := httptest.NewRecorder()
	setupSettingsGeneralHandler(t, queries).Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestShow_到達できない場合は404が返る(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		identifier  string
		scopes      []model.Scope
		topicNumber string
		outsider    bool
	}{
		{
			name:        "トピック更新権限がないメンバー",
			identifier:  "topic-general-reader",
			scopes:      []model.Scope{model.ScopePageRead},
			topicNumber: "1",
		},
		{
			name:        "スペースのメンバーではないユーザー",
			identifier:  "topic-general-outsider",
			topicNumber: "1",
			outsider:    true,
		},
		{
			name:        "存在しないトピック番号",
			identifier:  "topic-general-notopic",
			topicNumber: "999",
		},
		{
			name:        "数値でないトピック番号",
			identifier:  "topic-general-badnumber",
			topicNumber: "abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			queries := testutil.QueriesWithTx(tx)
			userID, _, _ := settingsGeneralSpace(t, tx, tt.identifier, tt.scopes)
			if tt.outsider {
				userID = testutil.NewUserBuilder(t, tx).
					WithEmail(tt.identifier + "-other@example.com").
					WithAtname(strings.ReplaceAll(tt.identifier, "-", "_") + "_other").
					Build()
			}

			req := newSettingsGeneralRequest(t, http.MethodGet, tt.identifier, tt.topicNumber, userID, nil)
			rr := httptest.NewRecorder()
			setupSettingsGeneralHandler(t, queries).Show(rr, req)

			if rr.Code != http.StatusNotFound {
				t.Errorf("status code = %d, want %d", rr.Code, http.StatusNotFound)
			}
		})
	}
}

func TestShow_未ログインならログイン画面へリダイレクトする(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	identifier := "topic-general-anon"
	settingsGeneralSpace(t, tx, identifier, nil)

	req := newSettingsGeneralRequest(t, http.MethodGet, identifier, "1", "", nil)
	rr := httptest.NewRecorder()
	setupSettingsGeneralHandler(t, queries).Show(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusFound)
	}
	if location := rr.Header().Get("Location"); location != "/sign_in" {
		t.Errorf("Location = %q, want %q", location, "/sign_in")
	}
}

// settingsGeneralInputTag returns the opening tag of the input carrying the given id.
//
// [Ja] settingsGeneralInputTag は指定した id を持つ input の開始タグを返す。
func settingsGeneralInputTag(t *testing.T, body, id string) string {
	t.Helper()

	idIndex := strings.Index(body, `id="`+id+`"`)
	if idIndex < 0 {
		t.Fatalf("response does not contain input id %q", id)
	}
	start := strings.LastIndex(body[:idIndex], "<input")
	endOffset := strings.Index(body[idIndex:], ">")
	if start < 0 || endOffset < 0 {
		t.Fatalf("response does not contain a complete input tag for id %q", id)
	}

	return body[start : idIndex+endOffset+1]
}
