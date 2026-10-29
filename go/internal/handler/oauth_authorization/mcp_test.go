package oauth_authorization_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// setupMCPAuthorizationUserは、MCPサーバーのフラグを持つユーザーを、keyに番号を付けた識別子のスペース
// spaces個に編集者として参加させ、ユーザーのIDを返す
func setupMCPAuthorizationUser(t *testing.T, tx *sql.Tx, key string, spaces int) model.UserID {
	t.Helper()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail(key + "@example.com").
		WithAtname(strings.ReplaceAll(key, "-", "_")).
		Build()
	testutil.NewFeatureFlagBuilder(t, tx).WithUserID(userID).WithName(string(model.FeatureFlagRemoteMCPServer)).Build()
	for i := range spaces {
		suffix := string(rune('a' + i))
		spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier(key + "-" + suffix).WithName("スペース" + suffix).Build()
		testutil.NewSpaceMemberBuilder(t, tx).WithSpaceID(spaceID).WithUserID(userID).WithRole(model.SpaceRoleEditor).Build()
	}
	return userID
}

// mcpParamsは、clientIDのアプリのMCPサーバーを宛先にした認可要求のパラメーターを返す
func mcpParams(clientID string) url.Values {
	params := validParams(clientID)
	params.Set("resource", "https://example.com/mcp")
	return params
}

func TestNew_MCPサーバー宛てでスペースを選んでいなければ連携するスペースを選ぶ段を示す(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	userID := setupMCPAuthorizationUser(t, tx, "oauthz-mcp-select", 2)
	testutil.NewOAuthApplicationBuilder(t, tx).
		AsOfficialClient().
		WithName("公式のアプリ").
		WithClientID("oauthz-mcp-select-official").
		WithRedirectURIs([]string{"https://client.example/callback?app=cli"}).
		Build()

	rr := httptest.NewRecorder()
	setupHandler(t, testutil.QueriesWithTx(tx)).New(rr, newRequest(t, http.MethodGet, userID, mcpParams("oauthz-mcp-select-official")))

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータス = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	assertResponseHeaders(t, rr)

	body := rr.Body.String()
	for _, want := range []string{
		"公式のアプリと連携するスペースを選んでください",
		`method="GET"`,
		`name="space_identifier" required type="radio" value="oauthz-mcp-select-a"`,
		`name="space_identifier" required type="radio" value="oauthz-mcp-select-b"`,
		"スペースa",
		"スペースb",
		// 選んだスペースを付けて、同じ認可要求を送り直す
		`name="resource" value="https://example.com/mcp"`,
		`name="code_challenge" value="` + testCodeChallenge + `"`,
		// 選ぶのをやめるときは、拒否としてクライアントへ戻る
		`name="decision" type="submit" value="deny"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("本文に%qが含まれていない", want)
		}
	}
	if strings.Contains(body, `value="approve"`) {
		t.Error("スペースを選ぶ段に許可のボタンが含まれている")
	}
}

func TestNew_選んだスペースを同意の送信へ引き継ぐ(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	userID := setupMCPAuthorizationUser(t, tx, "oauthz-mcp-selected", 2)
	testutil.NewOAuthApplicationBuilder(t, tx).
		AsOfficialClient().
		WithName("公式のアプリ").
		WithClientID("oauthz-mcp-selected-official").
		WithRedirectURIs([]string{"https://client.example/callback?app=cli"}).
		Build()
	params := mcpParams("oauthz-mcp-selected-official")
	params.Set("space_identifier", "oauthz-mcp-selected-b")

	rr := httptest.NewRecorder()
	setupHandler(t, testutil.QueriesWithTx(tx)).New(rr, newRequest(t, http.MethodGet, userID, params))

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータス = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, want := range []string{
		"公式のアプリにアクセスを許可しますか？",
		"スペースb",
		`name="space_identifier" value="oauthz-mcp-selected-b"`,
		`name="decision" type="submit" value="approve"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("本文に%qが含まれていない", want)
		}
	}
}

func TestNew_CIMDのクライアントにクライアントIDのホスト名とループバックだけのリダイレクトURIへの警告を示す(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	userID := setupMCPAuthorizationUser(t, tx, "oauthz-mcp-cimd", 1)
	clientID := "https://claude.ai/oauth/oauthz-mcp-cimd"
	// 期限内のキャッシュを置き、文書を取得しない
	if _, err := repository.NewOAuthApplicationRepository(testutil.QueriesWithTx(tx)).UpsertClientIDMetadataDocument(context.Background(), repository.UpsertClientIDMetadataDocumentInput{
		ClientID:          clientID,
		Name:              "Claude Code",
		RedirectURIs:      []string{"http://localhost/callback"},
		MetadataExpiresAt: time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("CIMDのクライアントの作成に失敗: %v", err)
	}
	params := mcpParams(clientID)
	params.Set("redirect_uri", "http://localhost:53123/callback")

	rr := httptest.NewRecorder()
	setupHandler(t, testutil.QueriesWithTx(tx)).New(rr, newRequest(t, http.MethodGet, userID, params))

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータス = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, want := range []string{
		"Claude Codeにアクセスを許可しますか？",
		"claude.aiが提供するアプリです",
		"移動先",
		"localhost:53123",
		"自分で始めた連携でなければ許可しないでください",
		// 選べるスペースが1つなので、選ぶ段を省いてそのスペースを引き継ぐ
		`name="space_identifier" value="oauthz-mcp-cimd-a"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("本文に%qが含まれていない", want)
		}
	}
}

func TestNew_登録したクライアントにはループバックだけのリダイレクトURIへの警告を示さない(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	f := setupAuthorizationFixture(t, tx, "oauthz-loopback-registered", model.SpaceRoleAdmin, true)
	params := validParams(f.clientID)
	params.Set("redirect_uri", "http://127.0.0.1:53123/callback")

	rr := httptest.NewRecorder()
	setupHandler(t, testutil.QueriesWithTx(tx)).New(rr, newRequest(t, http.MethodGet, f.userID, params))

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータス = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if strings.Contains(body, "自分で始めた連携でなければ許可しないでください") || strings.Contains(body, "が提供するアプリです") {
		t.Error("登録したクライアントの同意画面に、CIMDのクライアントへの表示が含まれている")
	}
}
