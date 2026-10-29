package usecase

import (
	"context"
	"database/sql"
	"slices"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// cimdTestRedirectURIは、テストのCIMDの文書が宣言するHTTPSのリダイレクトURI
const cimdTestRedirectURI = "https://claude.ai/api/mcp/auth_callback"

// cimdFixtureは、CIMDのクライアントと連携するスペースのメンバー
type cimdFixture struct {
	patMemberFixture
	key      string
	clientID string
}

// setupCIMDFixtureは、公開APIのフラグを持つスペースkeyのメンバーを作り、mcpFlagEnabledならMCPサーバーの
// フラグも有効にする。クライアントIDは許可リストのホストのURLで、keyごとに一意にする
func setupCIMDFixture(t *testing.T, tx *sql.Tx, key string, mcpFlagEnabled bool) cimdFixture {
	t.Helper()

	f := setupPATMember(t, tx, key, model.SpaceRoleEditor, true)
	if mcpFlagEnabled {
		testutil.NewFeatureFlagBuilder(t, tx).WithUserID(f.userID).WithName(string(model.FeatureFlagRemoteMCPServer)).Build()
	}
	return cimdFixture{patMemberFixture: f, key: key, clientID: "https://claude.ai/oauth/" + key}
}

// documentは、fのクライアントIDの検証を通るCIMDの文書を、名前nameで返す
func (f cimdFixture) document(name string) string {
	return `{"client_id":"` + f.clientID + `","client_name":"` + name + `","redirect_uris":["` + cimdTestRedirectURI + `","http://localhost/callback"],"token_endpoint_auth_method":"none"}`
}

// paramsは、fのスペースに連携する、検証を通る認可要求のパラメーターを返す
func (f cimdFixture) params() OAuthAuthorizationParams {
	params := validOAuthAuthorizationParams(f.clientID)
	params.RedirectURI = cimdTestRedirectURI
	params.Resource = model.SpaceAPIResourceURL(oauthAuthorizationTestConfig.AppURL(), model.SpaceIdentifier(f.key))
	return params
}

// buildCachedClientは、fのクライアントIDのCIMDのクライアントを、名前nameと期限expiresAtで作る
func (f cimdFixture) buildCachedClient(t *testing.T, tx *sql.Tx, name string, expiresAt time.Time) {
	t.Helper()

	_, err := repository.NewOAuthApplicationRepository(testutil.QueriesWithTx(tx)).UpsertClientIDMetadataDocument(context.Background(), repository.UpsertClientIDMetadataDocumentInput{
		ClientID:          f.clientID,
		Name:              name,
		RedirectURIs:      []string{cimdTestRedirectURI},
		MetadataExpiresAt: expiresAt,
	})
	if err != nil {
		t.Fatalf("CIMDのクライアントの作成に失敗: %v", err)
	}
}

// assertFatalOAuthAuthorizationErrorは、errがクライアントへ戻さない認可要求のエラーであることを確かめる
func assertFatalOAuthAuthorizationError(t *testing.T, err error) {
	t.Helper()

	oe := model.AsOAuthAuthorizationError(err)
	if oe == nil {
		t.Fatalf("エラー = %v、期待値 = *model.OAuthAuthorizationError", err)
	}
	if oe.IsRedirectable() {
		t.Errorf("RedirectURI = %q、クライアントへ戻さないエラーを期待", oe.RedirectURI)
	}
}

func TestGetOAuthAuthorizationNewUsecase_Execute_CIMD(t *testing.T) {
	t.Parallel()

	t.Run("文書を取得してCIMDのクライアントを作り、期限内は取得し直さない", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		f := setupCIMDFixture(t, tx, "cimd-fetch", true)
		stub := testutil.NewCIMDStub(map[string]string{f.clientID: f.document("Claude")})
		uc := newGetOAuthAuthorizationNewUsecaseWithCIMD(q, stub.Fetcher())

		output, err := uc.Execute(context.Background(), GetOAuthAuthorizationNewInput{UserID: f.userID, Params: f.params()})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		app := output.Application
		if app.ClientID != f.clientID || app.Name != "Claude" || !app.IsClientIDMetadataDocument() {
			t.Errorf("Application = %+v、期待値 = 名前がClaudeのCIMDのクライアント", app)
		}
		if app.SpaceID != nil || app.ClientType != model.OAuthClientTypePublic {
			t.Errorf("Application = %+v、スペースに属さないpublicクライアントを期待", app)
		}
		if !app.IsMetadataFresh(time.Now()) {
			t.Errorf("MetadataExpiresAt = %v、期限内を期待", app.MetadataExpiresAt)
		}
		if output.Space.ID != f.spaceID {
			t.Errorf("Space.ID = %v、期待値 = %v", output.Space.ID, f.spaceID)
		}

		if _, err := uc.Execute(context.Background(), GetOAuthAuthorizationNewInput{UserID: f.userID, Params: f.params()}); err != nil {
			t.Fatalf("2回目のExecute()のエラー = %v", err)
		}
		if got := stub.Requests(); !slices.Equal(got, []string{f.clientID}) {
			t.Errorf("取得したURL = %v、期待値 = [%s] (期限内は取得し直さない)", got, f.clientID)
		}
	})

	t.Run("文書のlocalhostのリダイレクトURIはポートを付けた要求と一致する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		f := setupCIMDFixture(t, tx, "cimd-localhost", true)
		stub := testutil.NewCIMDStub(map[string]string{f.clientID: f.document("Claude Code")})
		params := f.params()
		params.RedirectURI = "http://localhost:53123/callback"

		output, err := newGetOAuthAuthorizationNewUsecaseWithCIMD(testutil.QueriesWithTx(tx), stub.Fetcher()).Execute(context.Background(), GetOAuthAuthorizationNewInput{UserID: f.userID, Params: params})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output.RedirectURI != params.RedirectURI {
			t.Errorf("RedirectURI = %q、期待値 = %q", output.RedirectURI, params.RedirectURI)
		}
	})

	t.Run("キャッシュの期限が切れていれば取得し直して更新する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		f := setupCIMDFixture(t, tx, "cimd-refetch", true)
		f.buildCachedClient(t, tx, "古い名前", time.Now().Add(-time.Minute))
		stub := testutil.NewCIMDStub(map[string]string{f.clientID: f.document("新しい名前")})

		output, err := newGetOAuthAuthorizationNewUsecaseWithCIMD(testutil.QueriesWithTx(tx), stub.Fetcher()).Execute(context.Background(), GetOAuthAuthorizationNewInput{UserID: f.userID, Params: f.params()})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output.Application.Name != "新しい名前" || !output.Application.IsMetadataFresh(time.Now()) {
			t.Errorf("Application = %+v、取得し直した名前と期限を期待", output.Application)
		}
		if got := len(stub.Requests()); got != 1 {
			t.Errorf("取得の回数 = %d、期待値 = 1", got)
		}
	})

	t.Run("MCPサーバーのフラグが無効なら、文書を取得せずに見つからないとして答える", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		f := setupCIMDFixture(t, tx, "cimd-noflag", false)
		stub := testutil.NewCIMDStub(map[string]string{f.clientID: f.document("Claude")})

		_, err := newGetOAuthAuthorizationNewUsecaseWithCIMD(testutil.QueriesWithTx(tx), stub.Fetcher()).Execute(context.Background(), GetOAuthAuthorizationNewInput{UserID: f.userID, Params: f.params()})
		ae := model.AsAppError(err)
		if ae == nil || ae.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("エラー = %v、期待値 = AppErrCodeResourceNotFound", err)
		}
		if got := stub.Requests(); len(got) != 0 {
			t.Errorf("取得したURL = %v、期待値 = 取得しない", got)
		}
	})
}

func TestGetOAuthAuthorizationNewUsecase_Execute_CIMDのクライアントへ戻さないエラー(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  string
		// setupはスタブの文書とキャッシュを用意し、認可要求のパラメーターを変える
		setup func(t *testing.T, tx *sql.Tx, f cimdFixture, stub *testutil.CIMDStub, params *OAuthAuthorizationParams)
		// wantNoRequestは、文書を取得しないことを期待するか
		wantNoRequest bool
	}{
		{
			name: "許可リストの外のホストのクライアントID",
			key:  "cimd-fatal-host",
			setup: func(_ *testing.T, _ *sql.Tx, f cimdFixture, stub *testutil.CIMDStub, params *OAuthAuthorizationParams) {
				params.ClientID = "https://example.com/oauth/" + f.key
				stub.SetDocument(params.ClientID, f.document("Evil"))
			},
			wantNoRequest: true,
		},
		{
			name: "文書を取得できない",
			key:  "cimd-fatal-404",
			setup: func(*testing.T, *sql.Tx, cimdFixture, *testutil.CIMDStub, *OAuthAuthorizationParams) {
			},
		},
		{
			name: "キャッシュの期限が切れていて、取得し直せない",
			key:  "cimd-fatal-expired",
			setup: func(t *testing.T, tx *sql.Tx, f cimdFixture, _ *testutil.CIMDStub, _ *OAuthAuthorizationParams) {
				f.buildCachedClient(t, tx, "Claude", time.Now().Add(-time.Minute))
			},
		},
		{
			name: "文書のclient_idがURLと一致しない",
			key:  "cimd-fatal-mismatch",
			setup: func(_ *testing.T, _ *sql.Tx, f cimdFixture, stub *testutil.CIMDStub, _ *OAuthAuthorizationParams) {
				other := cimdFixture{clientID: "https://claude.ai/oauth/other"}
				stub.SetDocument(f.clientID, other.document("Claude"))
			},
		},
		{
			name: "redirect_uriが文書のものと一致しない",
			key:  "cimd-fatal-redirect",
			setup: func(_ *testing.T, _ *sql.Tx, f cimdFixture, stub *testutil.CIMDStub, params *OAuthAuthorizationParams) {
				stub.SetDocument(f.clientID, f.document("Claude"))
				params.RedirectURI = "https://claude.ai/other_callback"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			f := setupCIMDFixture(t, tx, tt.key, true)
			stub := testutil.NewCIMDStub(nil)
			params := f.params()
			tt.setup(t, tx, f, stub, &params)

			_, err := newGetOAuthAuthorizationNewUsecaseWithCIMD(testutil.QueriesWithTx(tx), stub.Fetcher()).Execute(context.Background(), GetOAuthAuthorizationNewInput{UserID: f.userID, Params: params})
			assertFatalOAuthAuthorizationError(t, err)
			if got := stub.Requests(); tt.wantNoRequest && len(got) != 0 {
				t.Errorf("取得したURL = %v、期待値 = 取得しない", got)
			}
		})
	}
}

func TestCreateOAuthAuthorizationUsecase_Execute_CIMD(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	var f cimdFixture
	testutil.CommitFixture(t, func(tx *sql.Tx) { f = setupCIMDFixture(t, tx, "cimd-create", true) })
	stub := testutil.NewCIMDStub(map[string]string{f.clientID: f.document("Claude")})

	output, err := newCreateOAuthAuthorizationUsecaseWithCIMD(query.New(db), stub.Fetcher()).Execute(context.Background(), CreateOAuthAuthorizationInput{
		UserID:        f.userID,
		Params:        f.params(),
		Approved:      true,
		GrantedScopes: []string{string(model.ScopeTopicRead)},
	})
	if err != nil {
		t.Fatalf("Execute()のエラー = %v", err)
	}
	if output.Code == "" || output.RedirectURI != cimdTestRedirectURI {
		t.Errorf("output = %+v、リダイレクトURIへ付ける認可コードを期待", output)
	}
}
