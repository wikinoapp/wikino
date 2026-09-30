package personal_access_token_test

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestNew_発行フォームを表示する(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	identifier := "pat-new"
	m := setupPATMember(t, tx, identifier, []model.Scope{model.ScopePersonalAccessTokenWrite}, true)

	req := newRequest(t, http.MethodGet, "/s/"+identifier+"/settings/personal_access_tokens/new", identifier, m.userID, nil)
	rr := httptest.NewRecorder()
	setupHandler(t, q).New(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("Cache-Control"); got != "private, no-cache" {
		t.Errorf("Cache-Control = %q、期待値 = %q", got, "private, no-cache")
	}

	body := rr.Body.String()
	for _, want := range []string{
		`action="/s/` + identifier + `/settings/personal_access_tokens"`,
		`name="csrf_token"`,
		`name="name"`,
		// 付与できるスコープだけを選択肢に出す。
		`value="topic:read"`,
		`value="page:read"`,
		`value="page:write"`,
		// 有効期限は選択肢から選び、既定は30日にする。
		`<option value="7">`,
		`<option value="30" selected>`,
		`<option value="90">`,
		`<option value="365">`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに%qが含まれていない", want)
		}
	}
	for _, notWant := range []string{`value="space:admin"`, `value="personal_access_token:write"`} {
		if strings.Contains(body, notWant) {
			t.Errorf("レスポンスに%qが含まれている", notWant)
		}
	}

	// 必須項目のラベルの中に必須のバッジを出す
	for _, tc := range []struct {
		open  string
		label string
		close string
	}{
		{open: `<label class="label" for="name">`, label: "名前", close: "</label>"},
		{open: `<legend class="label">`, label: "スコープ", close: "</legend>"},
		{open: `<label class="label" for="expiration_days">`, label: "有効期限", close: "</label>"},
	} {
		pattern := regexp.QuoteMeta(tc.open) + `\s*` + regexp.QuoteMeta(tc.label) + `\s*<span class="badge" data-variant="outline">\s*必須\s*</span>\s*` + regexp.QuoteMeta(tc.close)
		if !regexp.MustCompile(pattern).MatchString(body) {
			t.Errorf("%sのラベルの中に必須のバッジが無い", tc.label)
		}
	}
	// バッジを出した項目は、入力欄にもrequiredを付けて支援技術とブラウザに必須を伝える。
	// スコープはチェックボックスのためrequiredを付けない (すべてのチェックを求めてしまう)。
	for _, id := range []string{"name", "expiration_days"} {
		if !regexp.MustCompile(`id="` + id + `"[^>]*required`).MatchString(body) {
			t.Errorf("%sにrequiredが付いていない", id)
		}
	}

	crumb := breadcrumb(t, body)
	if !strings.Contains(crumb, `href="/s/`+identifier+`/settings/personal_access_tokens"`) {
		t.Error("パンくずに一覧へのリンクが含まれていない")
	}
}

func TestNew_発行の権限が無ければ404が返る(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		identifier  string
		scopes      []model.Scope
		flagEnabled bool
	}{
		{name: "personal_access_token:readだけを持つ", identifier: "pat-new-readonly", scopes: []model.Scope{model.ScopePersonalAccessTokenRead}, flagEnabled: true},
		{name: "フィーチャーフラグが無効", identifier: "pat-new-noflag", scopes: []model.Scope{model.ScopeSpaceAdmin}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			q := testutil.QueriesWithTx(tx)
			m := setupPATMember(t, tx, tt.identifier, tt.scopes, tt.flagEnabled)

			req := newRequest(t, http.MethodGet, "/s/"+tt.identifier+"/settings/personal_access_tokens/new", tt.identifier, m.userID, nil)
			rr := httptest.NewRecorder()
			setupHandler(t, q).New(rr, req)

			if rr.Code != http.StatusNotFound {
				t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusNotFound)
			}
		})
	}
}
