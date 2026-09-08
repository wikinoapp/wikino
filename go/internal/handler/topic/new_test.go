package topic_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// newTopicFormRequest builds a request for the topic creation screens, carrying the chi URL
// parameter, the CSRF token, the signed-in user and the locale the screen is rendered in. A zero
// userID leaves the user off, which is how a test reaches the signed-out path.
//
// [Ja] newTopicFormRequest はトピック作成の画面向けに、chi の URL パラメータ・CSRF トークン・
// ログイン中のユーザー・画面を描画するロケールを載せたリクエストを組み立てる。userID が空のときは
// ユーザーを載せず、テストが未ログインの経路へ到達できるようにする。
func newTopicFormRequest(t *testing.T, method, path, spaceIdentifier string, userID model.UserID, form map[string]string) *http.Request {
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

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("space_identifier", spaceIdentifier)

	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")
	if userID != "" {
		ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID, Atname: "topic-user"})
	}
	ctx = i18n.SetLocale(ctx, i18n.LangJa)

	return req.WithContext(ctx)
}

func topicFormInputTag(t *testing.T, body, id string) string {
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

// topicFormFieldsetTag returns the opening tag of the only fieldset the topic creation form has,
// which groups the visibility options.
//
// [Ja] topicFormFieldsetTag はトピック作成フォームが持つ唯一の fieldset の開始タグを返す。この
// fieldset は公開設定の選択肢をまとめている。
func topicFormFieldsetTag(t *testing.T, body string) string {
	t.Helper()

	start := strings.Index(body, "<fieldset")
	if start < 0 {
		t.Fatal("response does not contain a fieldset")
	}
	endOffset := strings.Index(body[start:], ">")
	if endOffset < 0 {
		t.Fatal("response does not contain a complete fieldset tag")
	}

	return body[start : start+endOffset+1]
}

func TestNew_公開設定は初期状態で未選択になる(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	userID := topicSpace(t, tx, "topic-new-no-visibility", nil)

	req := newTopicFormRequest(t, http.MethodGet, "/s/topic-new-no-visibility/topics/new", "topic-new-no-visibility", userID, nil)
	rr := httptest.NewRecorder()
	setupHandler(t, queries).New(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, id := range []string{"visibility_public", "visibility_private"} {
		if tag := topicFormInputTag(t, body, id); strings.Contains(tag, "checked") {
			t.Errorf("input %q tag contains checked: %s", id, tag)
		}
	}
}

func TestNew_ヘルプリンク名が各言語で行き先を説明する(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	userID := topicSpace(t, tx, "topic-new-help-link", nil)

	tests := []struct {
		name   string
		locale string
		want   []string
	}{
		{
			name:   "日本語",
			locale: i18n.LangJa,
			want:   []string{">トピックについてのヘルプ</a>", ">非公開トピックについてのヘルプ</a>"},
		},
		{
			name:   "英語",
			locale: i18n.LangEn,
			want:   []string{">Topic help</a>", ">Private topic help</a>"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newTopicFormRequest(t, http.MethodGet, "/s/topic-new-help-link/topics/new", "topic-new-help-link", userID, nil)
			req = req.WithContext(i18n.SetLocale(req.Context(), tt.locale))
			rr := httptest.NewRecorder()
			setupHandler(t, queries).New(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("status code = %d, want %d", rr.Code, http.StatusOK)
			}
			body := rr.Body.String()
			for _, want := range tt.want {
				if !strings.Contains(body, want) {
					t.Errorf("response does not contain %q", want)
				}
			}
		})
	}
}

// topicSpace seeds a space with one member and returns what the tests address them by.
//
// [Ja] topicSpace はメンバーが 1 人いるスペースを用意し、テストがそれらを指すための値を返す。
func topicSpace(t *testing.T, tx *sql.Tx, identifier string, scopes []model.Scope) model.UserID {
	t.Helper()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail(identifier + "@example.com").
		WithAtname(identifier).
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

	return userID
}

func TestNew_フォームが表示される(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := topicSpace(t, tx, "topic-new-ok", nil)

	req := newTopicFormRequest(t, http.MethodGet, "/s/topic-new-ok/topics/new", "topic-new-ok", userID, nil)
	rr := httptest.NewRecorder()
	setupHandler(t, queries).New(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	for _, want := range []string{
		`action="/s/topic-new-ok/topics"`,
		`name="name"`,
		`name="description"`,
		`name="visibility"`,
		`value="private"`,
		"test-csrf-token",
		"新規トピック",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("response does not contain %q", want)
		}
	}
}

func TestNew_HEADでも200が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := topicSpace(t, tx, "topic-new-head", nil)

	req := newTopicFormRequest(t, http.MethodHead, "/s/topic-new-head/topics/new", "topic-new-head", userID, nil)
	rr := httptest.NewRecorder()
	setupHandler(t, queries).New(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestNew_トピック作成権限がないメンバーには404が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := topicSpace(t, tx, "topic-new-reader", []model.Scope{model.ScopePageRead})

	req := newTopicFormRequest(t, http.MethodGet, "/s/topic-new-reader/topics/new", "topic-new-reader", userID, nil)
	rr := httptest.NewRecorder()
	setupHandler(t, queries).New(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestNew_スペースのメンバーでなければ404が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	topicSpace(t, tx, "topic-new-space", nil)
	outsiderID := testutil.NewUserBuilder(t, tx).
		WithEmail("topic-new-outsider@example.com").
		WithAtname("topic_new_outsider").
		Build()

	req := newTopicFormRequest(t, http.MethodGet, "/s/topic-new-space/topics/new", "topic-new-space", outsiderID, nil)
	rr := httptest.NewRecorder()
	setupHandler(t, queries).New(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestNew_未ログインならログイン画面へリダイレクトする(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	topicSpace(t, tx, "topic-new-anon", nil)

	req := newTopicFormRequest(t, http.MethodGet, "/s/topic-new-anon/topics/new", "topic-new-anon", "", nil)
	rr := httptest.NewRecorder()
	setupHandler(t, queries).New(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusFound)
	}
	if location := rr.Header().Get("Location"); location != "/sign_in" {
		t.Errorf("Location = %q, want %q", location, "/sign_in")
	}
}
