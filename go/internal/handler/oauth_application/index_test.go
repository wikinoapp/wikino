package oauth_application_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestIndex_スペースのアプリの一覧を作成者とともに表示する(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	identifier := "oauth-app-index"
	m := setupAppMember(t, tx, identifier, "閲覧するメンバー", model.SpaceRoleAdmin, true)
	other := setupAppMember(t, tx, "oauth-app-index-other", "別のスペースのメンバー", model.SpaceRoleAdmin, true)

	creatorUserID := testutil.NewUserBuilder(t, tx).
		WithEmail("oauth-app-index-creator@example.com").
		WithAtname("oauth_app_index_creator").
		WithName("作成したメンバー").
		Build()
	creatorMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(m.spaceID).
		WithUserID(creatorUserID).
		Build()
	appID := testutil.NewOAuthApplicationBuilder(t, tx).
		WithSpaceID(m.spaceID).
		WithCreatedSpaceMemberID(creatorMemberID).
		WithName("連携するWebアプリ").
		WithClientID("oauth-app-index-client").
		WithConfidentialClientSecretDigest("oauth_app_index_secret").
		Build()
	testutil.NewOAuthApplicationBuilder(t, tx).
		WithSpaceID(m.spaceID).
		WithName("作成者のいないCLI").
		WithClientID("oauth-app-index-orphan-client").
		Build()
	testutil.NewOAuthApplicationBuilder(t, tx).
		WithSpaceID(m.spaceID).
		WithName("削除したアプリ").
		WithClientID("oauth-app-index-discarded-client").
		WithDiscardedAt(time.Now()).
		Build()
	testutil.NewOAuthApplicationBuilder(t, tx).
		WithSpaceID(other.spaceID).
		WithName("別のスペースのアプリ").
		WithClientID("oauth-app-index-other-client").
		Build()

	req := newRequest(t, http.MethodGet, "/s/"+identifier+"/settings/oauth_applications", identifier, m.userID, nil)
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
		"<title>OAuthアプリ | アプリのスペース</title>",
		"連携するWebアプリ",
		"サーバーで動くアプリ (confidential)",
		"作成したメンバー",
		"作成者のいないCLI",
		"CLI・デスクトップアプリ (public)",
		"退出したメンバー",
		`href="/s/` + identifier + `/settings/oauth_applications/` + string(appID) + `"`,
		`href="/s/` + identifier + `/settings/oauth_applications/new"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに%qが含まれていない", want)
		}
	}
	for _, notWant := range []string{"削除したアプリ", "別のスペースのアプリ"} {
		if strings.Contains(body, notWant) {
			t.Errorf("レスポンスに%qが含まれている", notWant)
		}
	}
}

func TestIndex_開けない場合は404が返る(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		identifier  string
		role        model.SpaceRole
		flagEnabled bool
	}{
		{name: "oauth_application:readを持たない編集者", identifier: "oauth-app-index-noscope", role: model.SpaceRoleEditor, flagEnabled: true},
		{name: "フィーチャーフラグが無効", identifier: "oauth-app-index-noflag", role: model.SpaceRoleAdmin},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			q := testutil.QueriesWithTx(tx)
			m := setupAppMember(t, tx, tt.identifier, "閲覧するメンバー", tt.role, tt.flagEnabled)

			req := newRequest(t, http.MethodGet, "/s/"+tt.identifier+"/settings/oauth_applications", tt.identifier, m.userID, nil)
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

	req := newRequest(t, http.MethodGet, "/s/oauth-app-index-anon/settings/oauth_applications", "oauth-app-index-anon", "", nil)
	rr := httptest.NewRecorder()
	setupHandler(t, q).Index(rr, req)

	if rr.Code != http.StatusFound || rr.Header().Get("Location") != "/sign_in" {
		t.Errorf("ステータスコード = %d、Location = %q、期待値 = 302、/sign_in", rr.Code, rr.Header().Get("Location"))
	}
}
