package oauth_grant_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestIndex_自分が許可した連携の一覧を表示する(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	identifier := "grant-index"
	m := setupGrantMember(t, tx, identifier, []model.Scope{model.ScopeSpaceAdmin}, true)
	otherUserID := testutil.NewUserBuilder(t, tx).WithEmail(identifier + "-other@example.com").WithAtname("grant_index_other").Build()
	otherMemberID := testutil.NewSpaceMemberBuilder(t, tx).WithSpaceID(m.spaceID).WithUserID(otherUserID).Build()

	// 既存の許可を広げると文字列の順に保存されるため、表示ではAPITokenScopesの順に並べ直す
	grantID := m.buildGrant(t, tx, m.spaceMemberID, "自宅のCLI", "grant-index-cli",
		[]model.Scope{model.ScopePageRead, model.ScopeTopicRead})
	revokedAppID := testutil.NewOAuthApplicationBuilder(t, tx).
		WithSpaceID(m.spaceID).WithName("解除したアプリ").WithClientID("grant-index-revoked").Build()
	testutil.NewOAuthGrantBuilder(t, tx).
		WithOAuthApplicationID(revokedAppID).WithSpaceID(m.spaceID).WithSpaceMemberID(m.spaceMemberID).
		WithRevokedAt(time.Now()).Build()
	m.buildGrant(t, tx, otherMemberID, "他人の連携", "grant-index-other", []model.Scope{model.ScopePageRead})

	req := newRequest(t, http.MethodGet, "/s/"+identifier+"/settings/oauth_grants", identifier, m.userID)
	rr := httptest.NewRecorder()
	setupHandler(t, q).Index(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("Cache-Control"); got != "private, no-cache" {
		t.Errorf("Cache-Control = %q、期待値 = %q", got, "private, no-cache")
	}

	body := rr.Body.String()
	for _, want := range []string{
		"<title>連携中のアプリ | 連携のスペース</title>",
		"自宅のCLI",
		"許可した日時",
		`action="/s/` + identifier + `/settings/oauth_grants/` + string(grantID) + `"`,
		`name="_method" value="DELETE"`,
		"連携を解除する",
		// 同じ文言の解除ボタンを見分けられるよう、ボタンの名前にアプリ名を連ねる
		`aria-labelledby="oauth-grant-` + string(grantID) + `-revoke-label oauth-grant-` + string(grantID) + `-name"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに%qが含まれていない", want)
		}
	}
	for _, notWant := range []string{"解除したアプリ", "他人の連携"} {
		if strings.Contains(body, notWant) {
			t.Errorf("レスポンスに%qが含まれている", notWant)
		}
	}

	topicRead := strings.Index(body, "トピックの読み取り")
	pageRead := strings.Index(body, "ページの読み取り")
	if topicRead == -1 || pageRead == -1 || topicRead > pageRead {
		t.Errorf("スコープの位置 (トピックの読み取り = %d、ページの読み取り = %d)、APITokenScopesの順を期待", topicRead, pageRead)
	}
}

func TestIndex_解除の権限だけで一覧と解除ボタンを表示する(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	identifier := "grant-index-delete-only"
	m := setupGrantMember(t, tx, identifier, []model.Scope{model.ScopeOAuthGrantDelete}, true)
	m.buildGrant(t, tx, m.spaceMemberID, "自宅のCLI", "grant-index-delete-only-cli", []model.Scope{model.ScopePageRead})

	req := newRequest(t, http.MethodGet, "/s/"+identifier+"/settings/oauth_grants", identifier, m.userID)
	rr := httptest.NewRecorder()
	setupHandler(t, q).Index(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	if !strings.Contains(rr.Body.String(), `name="_method" value="DELETE"`) {
		t.Error("レスポンスに解除のフォームが含まれていない")
	}
}

func TestIndex_閲覧の権限だけなら解除ボタンを出さない(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	identifier := "grant-index-readonly"
	m := setupGrantMember(t, tx, identifier, []model.Scope{model.ScopeOAuthGrantRead}, true)
	m.buildGrant(t, tx, m.spaceMemberID, "自宅のCLI", "grant-index-readonly-cli", []model.Scope{model.ScopePageRead})

	req := newRequest(t, http.MethodGet, "/s/"+identifier+"/settings/oauth_grants", identifier, m.userID)
	rr := httptest.NewRecorder()
	setupHandler(t, q).Index(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "自宅のCLI") {
		t.Error("レスポンスに連携中のアプリが含まれていない")
	}
	if strings.Contains(body, `name="_method" value="DELETE"`) {
		t.Error("解除の権限が無いメンバーのレスポンスに解除のフォームが含まれている")
	}
}

func TestIndex_連携が無ければその旨を表示する(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	identifier := "grant-index-empty"
	m := setupGrantMember(t, tx, identifier, []model.Scope{model.ScopeOAuthGrantRead}, true)

	req := newRequest(t, http.MethodGet, "/s/"+identifier+"/settings/oauth_grants", identifier, m.userID)
	rr := httptest.NewRecorder()
	setupHandler(t, q).Index(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	if !strings.Contains(rr.Body.String(), "連携中のアプリはありません") {
		t.Error("連携が無いことの表示が含まれていない")
	}
}

func TestIndex_開けない場合は404が返る(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		identifier  string
		scopes      []model.Scope
		flagEnabled bool
	}{
		{name: "oauth_grant:readを持たない", identifier: "grant-index-noscope", scopes: []model.Scope{model.ScopePersonalAccessTokenRead}, flagEnabled: true},
		{name: "フィーチャーフラグが無効", identifier: "grant-index-noflag", scopes: []model.Scope{model.ScopeSpaceAdmin}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			q := testutil.QueriesWithTx(tx)
			m := setupGrantMember(t, tx, tt.identifier, tt.scopes, tt.flagEnabled)

			req := newRequest(t, http.MethodGet, "/s/"+tt.identifier+"/settings/oauth_grants", tt.identifier, m.userID)
			rr := httptest.NewRecorder()
			setupHandler(t, q).Index(rr, req)

			if rr.Code != http.StatusNotFound {
				t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusNotFound)
			}
		})
	}
}

func TestIndex_未ログインならログイン画面へリダイレクトする(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	identifier := "grant-index-anon"
	setupGrantMember(t, tx, identifier, []model.Scope{model.ScopeSpaceAdmin}, true)

	req := newRequest(t, http.MethodGet, "/s/"+identifier+"/settings/oauth_grants", identifier, "")
	rr := httptest.NewRecorder()
	setupHandler(t, q).Index(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusFound)
	}
	if location := rr.Header().Get("Location"); location != "/sign_in" {
		t.Errorf("Location = %q、期待値 = %q", location, "/sign_in")
	}
}
