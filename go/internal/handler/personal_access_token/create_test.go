package personal_access_token_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// tokenValuePatternは発行直後の画面に出るトークンの値を取り出す
var tokenValuePattern = regexp.MustCompile(`value="(wkp_[A-Za-z0-9_-]+)"`)

func TestCreate_発行したトークンの値をその応答でだけ表示する(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	identifier := "pat-create"
	m := setupPATMember(t, tx, identifier, []model.Scope{model.ScopePersonalAccessTokenWrite}, true)

	form := url.Values{
		"name":            {"自宅のCLI"},
		"scopes":          {"page:read", "topic:read"},
		"expiration_days": {"7"},
	}
	req := newRequest(t, http.MethodPost, "/s/"+identifier+"/settings/personal_access_tokens", identifier, m.userID, form)
	rr := httptest.NewRecorder()
	setupHandler(t, q).Create(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q、期待値 = %q", got, "no-store")
	}

	body := rr.Body.String()
	match := tokenValuePattern.FindStringSubmatch(body)
	if match == nil {
		t.Fatal("レスポンスにトークンの値が含まれていない")
	}
	for _, want := range []string{"トークン (自宅のCLI)", "トピックの読み取り", "ページの読み取り", "二度と表示できない"} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに%qが含まれていない", want)
		}
	}

	token, err := repository.NewPersonalAccessTokenRepository(q).FindByTokenDigest(context.Background(), auth.DigestOpaqueToken(match[1]))
	if err != nil {
		t.Fatalf("FindByTokenDigest()のエラー = %v", err)
	}
	if token == nil {
		t.Fatal("表示したトークンがデータベースに無い")
	}
	if token.SpaceMemberID != m.spaceMemberID || token.Name != "自宅のCLI" {
		t.Errorf("保存されたトークン = %+v", token)
	}
}

func TestCreate_入力が不正なら422で送信内容を戻したフォームを再描画する(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	identifier := "pat-create-invalid"
	m := setupPATMember(t, tx, identifier, []model.Scope{model.ScopePersonalAccessTokenWrite}, true)

	form := url.Values{
		"name":            {"自宅のCLI"},
		"scopes":          {"page:write"},
		"expiration_days": {"0"},
	}
	req := newRequest(t, http.MethodPost, "/s/"+identifier+"/settings/personal_access_tokens", identifier, m.userID, form)
	rr := httptest.NewRecorder()
	setupHandler(t, q).Create(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusUnprocessableEntity)
	}
	if got := rr.Header().Get("Cache-Control"); got != "private, no-cache" {
		t.Errorf("Cache-Control = %q、期待値 = %q", got, "private, no-cache")
	}

	body := rr.Body.String()
	for _, want := range []string{
		"有効期限を選択肢から選んでください",
		`id="expiration_days-error"`,
		`value="自宅のCLI"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに%qが含まれていない", want)
		}
	}
	if tokenValuePattern.MatchString(body) {
		t.Error("拒否された送信のレスポンスにトークンの値が含まれている")
	}
	if !regexp.MustCompile(`id="scopes_page_write"[^>]*checked`).MatchString(body) {
		t.Error("送信したスコープのチェックが戻っていない")
	}

	tokens, err := repository.NewPersonalAccessTokenRepository(q).ListUnrevokedBySpaceMember(context.Background(), m.spaceID, m.spaceMemberID)
	if err != nil {
		t.Fatalf("ListUnrevokedBySpaceMember()のエラー = %v", err)
	}
	if len(tokens) != 0 {
		t.Errorf("作られたトークンの件数 = %d、期待値 = 0", len(tokens))
	}
}

func TestCreate_入力エラーの最初の欄にフォーカスする(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		form        url.Values
		wantInputID string
	}{
		{
			name:        "名前だけが不正",
			form:        url.Values{"name": {""}, "scopes": {"page:read"}, "expiration_days": {"30"}},
			wantInputID: "name",
		},
		{
			name:        "スコープだけが不正",
			form:        url.Values{"name": {"CLI"}, "expiration_days": {"30"}},
			wantInputID: "scopes_topic_read",
		},
		{
			name:        "有効期限だけが不正",
			form:        url.Values{"name": {"CLI"}, "scopes": {"page:read"}, "expiration_days": {"0"}},
			wantInputID: "expiration_days",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			q := testutil.QueriesWithTx(tx)
			identifier := "pat-create-focus-" + strings.ReplaceAll(tt.wantInputID, "_", "-")
			m := setupPATMember(t, tx, identifier, []model.Scope{model.ScopePersonalAccessTokenWrite}, true)

			req := newRequest(t, http.MethodPost, "/s/"+identifier+"/settings/personal_access_tokens", identifier, m.userID, tt.form)
			rr := httptest.NewRecorder()
			setupHandler(t, q).Create(rr, req)
			if rr.Code != http.StatusUnprocessableEntity {
				t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusUnprocessableEntity)
			}

			focused := regexp.MustCompile(`<(?:input|select)\b[^>]*\bautofocus\b[^>]*>`).FindAllString(rr.Body.String(), -1)
			if len(focused) != 1 || !strings.Contains(focused[0], `id="`+tt.wantInputID+`"`) {
				t.Errorf("フォーカス先 = %v、期待値のID = %q", focused, tt.wantInputID)
			}
		})
	}
}

func TestCreate_発行の権限が無ければ404が返る(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	identifier := "pat-create-readonly"
	m := setupPATMember(t, tx, identifier, []model.Scope{model.ScopePersonalAccessTokenRead}, true)

	form := url.Values{
		"name":            {"CLI"},
		"scopes":          {"page:read"},
		"expiration_days": {"30"},
	}
	req := newRequest(t, http.MethodPost, "/s/"+identifier+"/settings/personal_access_tokens", identifier, m.userID, form)
	rr := httptest.NewRecorder()
	setupHandler(t, q).Create(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusNotFound)
	}
}
