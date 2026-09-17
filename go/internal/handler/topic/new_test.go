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

// newTopicFormRequestはトピック作成の画面向けに、chiのURLパラメータ・CSRFトークン・
// ログイン中のユーザー・画面を描画するロケールを載せたリクエストを組み立てる。userIDが空のときは
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
		t.Fatalf("レスポンスにid%qの入力欄が含まれていない", id)
	}
	start := strings.LastIndex(body[:idIndex], "<input")
	endOffset := strings.Index(body[idIndex:], ">")
	if start < 0 || endOffset < 0 {
		t.Fatalf("レスポンスにid%qの完全なinputタグが含まれていない", id)
	}

	return body[start : idIndex+endOffset+1]
}

// topicFormFieldsetTagはトピック作成フォームが持つ唯一のfieldsetの開始タグを返す。この
// fieldsetは公開設定の選択肢をまとめている。
func topicFormFieldsetTag(t *testing.T, body string) string {
	t.Helper()

	start := strings.Index(body, "<fieldset")
	if start < 0 {
		t.Fatal("レスポンスにfieldsetが含まれていない")
	}
	endOffset := strings.Index(body[start:], ">")
	if endOffset < 0 {
		t.Fatal("レスポンスに完全なfieldsetタグが含まれていない")
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
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, id := range []string{"visibility_public", "visibility_private"} {
		if tag := topicFormInputTag(t, body, id); strings.Contains(tag, "checked") {
			t.Errorf("入力%qのタグにcheckedが含まれている: %s", id, tag)
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
				t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
			}
			body := rr.Body.String()
			for _, want := range tt.want {
				if !strings.Contains(body, want) {
					t.Errorf("レスポンスに%qが含まれていない", want)
				}
			}
		})
	}
}

// 見出し下のサブタイトルは自身の文字色を持つため、その中のヘルプリンクは自分で色を塗らず
// link-inherit-foregroundでその色を受け取る。フォーム内の注意書きは別の色の枠にあり
// link-foregroundのままなので、両者が混ざらないようここで併せて確認する。クラス名は
// テンプレートではなくロケール文字列側にあるため、両ロケールを確認する。片方だけ元に戻されても
// 気づけなくなるからである。
func TestNew_サブタイトルのヘルプリンクが本文色を継承する(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	userID := topicSpace(t, tx, "topic-new-link-color", nil)

	tests := []struct {
		name   string
		locale string
	}{
		{
			name:   "日本語",
			locale: i18n.LangJa,
		},
		{
			name:   "英語",
			locale: i18n.LangEn,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newTopicFormRequest(t, http.MethodGet, "/s/topic-new-link-color/topics/new", "topic-new-link-color", userID, nil)
			req = req.WithContext(i18n.SetLocale(req.Context(), tt.locale))
			rr := httptest.NewRecorder()
			setupHandler(t, queries).New(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
			}

			body := rr.Body.String()
			for _, want := range []string{
				`<a class="link-inherit-foreground" href="https://wikino.app/s/wikino/pages/11"`,
				`<a class="link-inherit-foreground" href="https://wikino.app/s/wikino/pages/52"`,
				`<a class="link-foreground" href="https://wikino.app/s/wikino/pages/38"`,
			} {
				if !strings.Contains(body, want) {
					t.Errorf("レスポンスに%qが含まれていない", want)
				}
			}
		})
	}
}

// 公開設定のラベルと最初の選択肢の間隔は、basecoatの .fieldsetがlegendに付ける
// margin-bottomだけが作っている。legendはフレックスの子にならないため、グループに指定したgapは
// legendまで届かない。したがってbasecoatがそのmarginを0に戻すdata-variant="label" はlegend
// に付けないままにする必要がある。どちらが欠けても密着した見た目に戻るため、両方を確認する。
func TestNew_公開設定のラベルと選択肢の間に余白が入る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	userID := topicSpace(t, tx, "topic-new-fieldset", nil)

	req := newTopicFormRequest(t, http.MethodGet, "/s/topic-new-fieldset/topics/new", "topic-new-fieldset", userID, nil)
	rr := httptest.NewRecorder()
	setupHandler(t, queries).New(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if tag := topicFormFieldsetTag(t, body); !strings.Contains(tag, `class="fieldset gap-3"`) {
		t.Errorf("fieldsetタグにfieldsetクラスが付いていない: %s", tag)
	}
	if !strings.Contains(body, `<legend class="label">`) {
		t.Error("レスポンスにdata-variantを持たないlegendが含まれていない")
	}
}

// topicSpaceはメンバーが1人いるスペースを用意し、テストがそれらを指すための値を返す。
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
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
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
			t.Errorf("レスポンスに%qが含まれていない", want)
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
		t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
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
		t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusNotFound)
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
		t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusNotFound)
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
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusFound)
	}
	if location := rr.Header().Get("Location"); location != "/sign_in" {
		t.Errorf("Location = %q、期待値 = %q", location, "/sign_in")
	}
}

// 経路はこの画面自身で終わるため、末尾の項目はスペースへのリンクではなくaria-currentを持つ
// ラベルになる。同じラベルは見出しとページタイトルにも出るため、パンくず内に絞って検証する。
func TestNew_パンくずが現在地の項目で終わる(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)
	userID := topicSpace(t, tx, "topic-new-crumb", nil)

	req := newTopicFormRequest(t, http.MethodGet, "/s/topic-new-crumb/topics/new", "topic-new-crumb", userID, nil)
	rr := httptest.NewRecorder()
	setupHandler(t, queries).New(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}

	breadcrumb := topicFormBreadcrumb(t, rr.Body.String())
	for _, want := range []string{
		`href="/home"`,
		`href="/s/topic-new-crumb"`,
		`aria-current="page"`,
		"新規トピック",
	} {
		if !strings.Contains(breadcrumb, want) {
			t.Errorf("パンくずに%qが含まれていない", want)
		}
	}
	if strings.Contains(breadcrumb, `href="/s/topic-new-crumb/topics/new"`) {
		t.Error("現在のトピック作成のパンくずの項目がリンクになっている")
	}
}

// topicFormBreadcrumbはパンくずのナビゲーション部分だけのマークアップを返す。経路についての
// 検証が、画面の他の場所に出た同じ文字列で満たされてしまうのを防ぐ。
func topicFormBreadcrumb(t *testing.T, body string) string {
	t.Helper()

	start := strings.Index(body, `<nav aria-label="パンくずリスト"`)
	if start == -1 {
		t.Fatal("レスポンスにパンくずのナビゲーションが含まれていない")
	}
	endOffset := strings.Index(body[start:], "</nav>")
	if endOffset == -1 {
		t.Fatal("パンくずのナビゲーションに閉じタグが無い")
	}

	return body[start : start+endOffset]
}
