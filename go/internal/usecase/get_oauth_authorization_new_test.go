package usecase

import (
	"context"
	"database/sql"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/cimd"
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// oauthAuthorizationTestConfigは認可要求のテストの設定。`resource` はこのオリジンのURLで表す
var oauthAuthorizationTestConfig = &config.Config{Domain: "example.com"}

// oauthAuthorizationTestCodeChallengeはS256のcode_challengeの例 (RFC 7636 Appendix B)
const oauthAuthorizationTestCodeChallenge = "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"

// oauthAuthorizationFixtureは、スペースのOAuthアプリと、そのスペースのメンバーのユーザー
type oauthAuthorizationFixture struct {
	patMemberFixture
	clientID string
}

// setupOAuthAuthorizationFixtureは、スペースkeyのroleのロールを持つメンバーと、そのスペースのOAuthアプリを作る。
// アプリのリダイレクトURIは `https://example.com/callback` とループバックの `http://127.0.0.1/callback`
func setupOAuthAuthorizationFixture(t *testing.T, tx *sql.Tx, key string, role model.SpaceRole, flagEnabled bool) oauthAuthorizationFixture {
	t.Helper()

	f := setupPATMember(t, tx, key, role, flagEnabled)
	clientID := key + "-client"
	testutil.NewOAuthApplicationBuilder(t, tx).
		WithSpaceID(f.spaceID).
		WithName("連携アプリ " + key).
		WithClientID(clientID).
		WithRedirectURIs([]string{"https://example.com/callback", "http://127.0.0.1/callback"}).
		Build()

	return oauthAuthorizationFixture{patMemberFixture: f, clientID: clientID}
}

// validOAuthAuthorizationParamsは、clientIDのアプリへの検証を通る認可要求のパラメーターを返す
func validOAuthAuthorizationParams(clientID string) OAuthAuthorizationParams {
	return OAuthAuthorizationParams{
		ClientID:            clientID,
		RedirectURI:         "https://example.com/callback",
		ResponseType:        "code",
		Scope:               "page:write topic:read",
		State:               "xyz",
		CodeChallenge:       oauthAuthorizationTestCodeChallenge,
		CodeChallengeMethod: "S256",
	}
}

func newGetOAuthAuthorizationNewUsecaseForTest(q *query.Queries) *GetOAuthAuthorizationNewUsecase {
	return newGetOAuthAuthorizationNewUsecaseWithCIMD(q, testutil.NewCIMDStub(nil).Fetcher())
}

// newGetOAuthAuthorizationNewUsecaseWithCIMDは、CIMDの文書をfetcherで取得するユースケースを返す
func newGetOAuthAuthorizationNewUsecaseWithCIMD(q *query.Queries, fetcher *cimd.Fetcher) *GetOAuthAuthorizationNewUsecase {
	return NewGetOAuthAuthorizationNewUsecase(
		oauthAuthorizationTestConfig,
		repository.NewFeatureFlagRepository(q),
		repository.NewOAuthApplicationRepository(q),
		repository.NewSpaceRepository(q),
		repository.NewSpaceMemberRepository(q),
		repository.NewTopicMemberRepository(q),
		fetcher,
	)
}

func TestGetOAuthAuthorizationNewUsecase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("スペースのアプリは、resourceが無ければアプリのスペースに連携する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		f := setupOAuthAuthorizationFixture(t, tx, "goan-space-app", model.SpaceRoleViewer, true)

		output, err := newGetOAuthAuthorizationNewUsecaseForTest(testutil.QueriesWithTx(tx)).Execute(context.Background(), GetOAuthAuthorizationNewInput{
			UserID: f.userID,
			Params: validOAuthAuthorizationParams(f.clientID),
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output.Space.ID != f.spaceID {
			t.Errorf("Space.ID = %v、期待値 = %v", output.Space.ID, f.spaceID)
		}
		if output.Application.ClientID != f.clientID {
			t.Errorf("Application.ClientID = %q、期待値 = %q", output.Application.ClientID, f.clientID)
		}
		// 重複を除き、トークンに付与できるスコープの順に並べる
		if want := []model.Scope{model.ScopeTopicRead, model.ScopePageWrite}; !slices.Equal(output.Scopes, want) {
			t.Errorf("Scopes = %v、期待値 = %v", output.Scopes, want)
		}
		if output.DeniedReason != "" {
			t.Errorf("DeniedReason = %q、期待値 = 空", output.DeniedReason)
		}
	})

	t.Run("スペースのロールでもトピックのロールでも持たないスコープを、今の権限では効かないものとして返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		viewer := setupOAuthAuthorizationFixture(t, tx, "goan-ineffective", model.SpaceRoleViewer, true)

		output, err := newGetOAuthAuthorizationNewUsecaseForTest(q).Execute(context.Background(), GetOAuthAuthorizationNewInput{
			UserID: viewer.userID,
			Params: validOAuthAuthorizationParams(viewer.clientID),
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if want := []model.Scope{model.ScopePageWrite}; !slices.Equal(output.IneffectiveScopes, want) {
			t.Errorf("IneffectiveScopes = %v、期待値 = %v (スペース閲覧者はpage:writeを持たない)", output.IneffectiveScopes, want)
		}

		// トピック編集者のロールがあれば、スペースのロールが閲覧者でもpage:writeは効く
		topicID := testutil.NewTopicBuilder(t, tx).WithSpaceID(viewer.spaceID).WithNumber(1).Build()
		testutil.NewTopicMemberBuilder(t, tx).WithSpaceID(viewer.spaceID).WithTopicID(topicID).WithSpaceMemberID(viewer.spaceMemberID).WithRole(model.TopicRoleEditor).Build()

		output, err = newGetOAuthAuthorizationNewUsecaseForTest(q).Execute(context.Background(), GetOAuthAuthorizationNewInput{
			UserID: viewer.userID,
			Params: validOAuthAuthorizationParams(viewer.clientID),
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if len(output.IneffectiveScopes) != 0 {
			t.Errorf("IneffectiveScopes = %v、期待値 = 空", output.IneffectiveScopes)
		}
	})

	t.Run("スペースのアプリは、resourceがアプリのスペースを指せば受け付ける", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		f := setupOAuthAuthorizationFixture(t, tx, "goan-resource", model.SpaceRoleViewer, true)
		params := validOAuthAuthorizationParams(f.clientID)
		params.Resource = "https://example.com/api/v1/spaces/goan-resource"

		output, err := newGetOAuthAuthorizationNewUsecaseForTest(testutil.QueriesWithTx(tx)).Execute(context.Background(), GetOAuthAuthorizationNewInput{
			UserID: f.userID,
			Params: params,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output.Space.ID != f.spaceID {
			t.Errorf("Space.ID = %v、期待値 = %v", output.Space.ID, f.spaceID)
		}
	})

	t.Run("公式クライアントは、resourceが指すスペースに連携する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		f := setupPATMember(t, tx, "goan-official", model.SpaceRoleViewer, true)
		testutil.NewOAuthApplicationBuilder(t, tx).
			AsOfficialClient().
			WithClientID("goan-official-client").
			Build()
		params := validOAuthAuthorizationParams("goan-official-client")
		params.RedirectURI = "http://127.0.0.1:53123/callback"
		params.Resource = "https://example.com/api/v1/spaces/goan-official"

		output, err := newGetOAuthAuthorizationNewUsecaseForTest(testutil.QueriesWithTx(tx)).Execute(context.Background(), GetOAuthAuthorizationNewInput{
			UserID: f.userID,
			Params: params,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output.Space.ID != f.spaceID {
			t.Errorf("Space.ID = %v、期待値 = %v", output.Space.ID, f.spaceID)
		}
		// ループバックはポート番号を除いて照合し、要求のURIをそのまま使う
		if output.RedirectURI != params.RedirectURI {
			t.Errorf("RedirectURI = %q、期待値 = %q", output.RedirectURI, params.RedirectURI)
		}
	})

	t.Run("連携先のスペースのメンバーでなければ、その理由を返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		f := setupOAuthAuthorizationFixture(t, tx, "goan-not-member", model.SpaceRoleViewer, true)
		outsider := setupPATMember(t, tx, "goan-not-member-outsider", model.SpaceRoleViewer, true)

		output, err := newGetOAuthAuthorizationNewUsecaseForTest(testutil.QueriesWithTx(tx)).Execute(context.Background(), GetOAuthAuthorizationNewInput{
			UserID: outsider.userID,
			Params: validOAuthAuthorizationParams(f.clientID),
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output.DeniedReason != OAuthAuthorizationDeniedNotMember {
			t.Errorf("DeniedReason = %q、期待値 = %q", output.DeniedReason, OAuthAuthorizationDeniedNotMember)
		}
	})

	t.Run("フィーチャーフラグが無効なら見つからないとして答える", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		f := setupOAuthAuthorizationFixture(t, tx, "goan-noflag", model.SpaceRoleViewer, false)

		_, err := newGetOAuthAuthorizationNewUsecaseForTest(testutil.QueriesWithTx(tx)).Execute(context.Background(), GetOAuthAuthorizationNewInput{
			UserID: f.userID,
			Params: validOAuthAuthorizationParams(f.clientID),
		})
		ae := model.AsAppError(err)
		if ae == nil || ae.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("エラー = %v、期待値 = AppErrCodeResourceNotFound", err)
		}
	})
}

func TestGetOAuthAuthorizationNewUsecase_Execute_クライアントへ戻さないエラー(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		key    string
		modify func(p *OAuthAuthorizationParams)
	}{
		{name: "client_idが無い", key: "goan-fatal-no-client", modify: func(p *OAuthAuthorizationParams) { p.ClientID = "" }},
		{name: "client_idが登録されていない", key: "goan-fatal-unknown", modify: func(p *OAuthAuthorizationParams) { p.ClientID = "unknown-client" }},
		{name: "client_idが重複している", key: "goan-fatal-dup-client", modify: func(p *OAuthAuthorizationParams) { p.DuplicatedParams = []string{"client_id"} }},
		{name: "redirect_uriが無い", key: "goan-fatal-no-redirect", modify: func(p *OAuthAuthorizationParams) { p.RedirectURI = "" }},
		{name: "redirect_uriが登録と一致しない", key: "goan-fatal-mismatch", modify: func(p *OAuthAuthorizationParams) { p.RedirectURI = "https://evil.example/callback" }},
		{name: "redirect_uriが末尾だけ違う", key: "goan-fatal-trailing", modify: func(p *OAuthAuthorizationParams) { p.RedirectURI = "https://example.com/callback/" }},
		{name: "redirect_uriが重複している", key: "goan-fatal-dup-redirect", modify: func(p *OAuthAuthorizationParams) { p.DuplicatedParams = []string{"redirect_uri"} }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			f := setupOAuthAuthorizationFixture(t, tx, tt.key, model.SpaceRoleViewer, true)
			params := validOAuthAuthorizationParams(f.clientID)
			tt.modify(&params)

			_, err := newGetOAuthAuthorizationNewUsecaseForTest(testutil.QueriesWithTx(tx)).Execute(context.Background(), GetOAuthAuthorizationNewInput{
				UserID: f.userID,
				Params: params,
			})
			oe := model.AsOAuthAuthorizationError(err)
			if oe == nil {
				t.Fatalf("エラー = %v、期待値 = *model.OAuthAuthorizationError", err)
			}
			if oe.IsRedirectable() {
				t.Errorf("RedirectURI = %q、期待値 = 空 (クライアントへ戻さない)", oe.RedirectURI)
			}
			if oe.UserMsg == "" {
				t.Error("UserMsgが空")
			}
		})
	}

	t.Run("削除されたアプリ", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		f := setupPATMember(t, tx, "goan-fatal-discarded", model.SpaceRoleViewer, true)
		testutil.NewOAuthApplicationBuilder(t, tx).
			WithSpaceID(f.spaceID).
			WithClientID("goan-fatal-discarded-client").
			WithRedirectURIs([]string{"https://example.com/callback"}).
			WithDiscardedAt(time.Now()).
			Build()

		_, err := newGetOAuthAuthorizationNewUsecaseForTest(testutil.QueriesWithTx(tx)).Execute(context.Background(), GetOAuthAuthorizationNewInput{
			UserID: f.userID,
			Params: validOAuthAuthorizationParams("goan-fatal-discarded-client"),
		})
		if oe := model.AsOAuthAuthorizationError(err); oe == nil || oe.IsRedirectable() {
			t.Errorf("エラー = %v、期待値 = クライアントへ戻さない*model.OAuthAuthorizationError", err)
		}
	})
}

func TestGetOAuthAuthorizationNewUsecase_Execute_クライアントへ戻すエラー(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		key      string
		modify   func(p *OAuthAuthorizationParams)
		wantCode model.OAuthAuthorizationErrorCode
	}{
		{name: "response_typeが無い", key: "goan-rt-missing", modify: func(p *OAuthAuthorizationParams) { p.ResponseType = "" }, wantCode: model.OAuthAuthorizationErrorInvalidRequest},
		{name: "response_typeがtoken", key: "goan-rt-token", modify: func(p *OAuthAuthorizationParams) { p.ResponseType = "token" }, wantCode: model.OAuthAuthorizationErrorUnsupportedResponseType},
		{name: "code_challengeが無い", key: "goan-pkce-missing", modify: func(p *OAuthAuthorizationParams) { p.CodeChallenge = "" }, wantCode: model.OAuthAuthorizationErrorInvalidRequest},
		{name: "code_challenge_methodが無い (plain)", key: "goan-pkce-nomethod", modify: func(p *OAuthAuthorizationParams) { p.CodeChallengeMethod = "" }, wantCode: model.OAuthAuthorizationErrorInvalidRequest},
		{name: "code_challenge_methodがplain", key: "goan-pkce-plain", modify: func(p *OAuthAuthorizationParams) { p.CodeChallengeMethod = "plain" }, wantCode: model.OAuthAuthorizationErrorInvalidRequest},
		{name: "code_challengeがS256の形式でない", key: "goan-pkce-short", modify: func(p *OAuthAuthorizationParams) { p.CodeChallenge = "short" }, wantCode: model.OAuthAuthorizationErrorInvalidRequest},
		{name: "stateが重複している", key: "goan-dup-state", modify: func(p *OAuthAuthorizationParams) { p.DuplicatedParams = []string{"state"} }, wantCode: model.OAuthAuthorizationErrorInvalidRequest},
		{name: "scopeが無い", key: "goan-scope-missing", modify: func(p *OAuthAuthorizationParams) { p.Scope = "" }, wantCode: model.OAuthAuthorizationErrorInvalidScope},
		{name: "scopeに不明な値がある", key: "goan-scope-unknown", modify: func(p *OAuthAuthorizationParams) { p.Scope = "page:read unknown:read" }, wantCode: model.OAuthAuthorizationErrorInvalidScope},
		{name: "scopeにトークンへ付与できない値がある", key: "goan-scope-admin", modify: func(p *OAuthAuthorizationParams) { p.Scope = "space:admin" }, wantCode: model.OAuthAuthorizationErrorInvalidScope},
		{name: "scopeにトークン管理のスコープがある", key: "goan-scope-grant", modify: func(p *OAuthAuthorizationParams) { p.Scope = "page:read oauth_grant:write" }, wantCode: model.OAuthAuthorizationErrorInvalidScope},
		{name: "resourceの形式が不正", key: "goan-res-malformed", modify: func(p *OAuthAuthorizationParams) { p.Resource = "https://example.com/s/goan-res-malformed" }, wantCode: model.OAuthAuthorizationErrorInvalidTarget},
		{name: "resourceが別のオリジンを指す", key: "goan-res-origin", modify: func(p *OAuthAuthorizationParams) {
			p.Resource = "https://evil.example/api/v1/spaces/goan-res-origin"
		}, wantCode: model.OAuthAuthorizationErrorInvalidTarget},
		{name: "resourceが存在しないスペースを指す", key: "goan-res-missing", modify: func(p *OAuthAuthorizationParams) {
			p.Resource = "https://example.com/api/v1/spaces/no-such-space"
		}, wantCode: model.OAuthAuthorizationErrorInvalidTarget},
		{name: "device_nameが100文字を超える", key: "goan-device-long", modify: func(p *OAuthAuthorizationParams) { p.DeviceName = strings.Repeat("a", 101) }, wantCode: model.OAuthAuthorizationErrorInvalidRequest},
		{name: "device_nameに制御文字がある", key: "goan-device-control", modify: func(p *OAuthAuthorizationParams) { p.DeviceName = "my\tlaptop" }, wantCode: model.OAuthAuthorizationErrorInvalidRequest},
		{name: "device_nameが重複している", key: "goan-device-dup", modify: func(p *OAuthAuthorizationParams) { p.DuplicatedParams = []string{"device_name"} }, wantCode: model.OAuthAuthorizationErrorInvalidRequest},
		{name: "resourceが重複している", key: "goan-res-dup", modify: func(p *OAuthAuthorizationParams) { p.DuplicatedParams = []string{"resource"} }, wantCode: model.OAuthAuthorizationErrorInvalidTarget},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			f := setupOAuthAuthorizationFixture(t, tx, tt.key, model.SpaceRoleViewer, true)
			params := validOAuthAuthorizationParams(f.clientID)
			tt.modify(&params)

			_, err := newGetOAuthAuthorizationNewUsecaseForTest(testutil.QueriesWithTx(tx)).Execute(context.Background(), GetOAuthAuthorizationNewInput{
				UserID: f.userID,
				Params: params,
			})
			assertRedirectableOAuthAuthorizationError(t, err, tt.wantCode, params)
		})
	}

	t.Run("スペースのアプリで、resourceが別のスペースを指す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		f := setupOAuthAuthorizationFixture(t, tx, "goan-res-other", model.SpaceRoleViewer, true)
		setupPATMember(t, tx, "goan-res-other-space", model.SpaceRoleViewer, true)
		params := validOAuthAuthorizationParams(f.clientID)
		params.Resource = "https://example.com/api/v1/spaces/goan-res-other-space"

		_, err := newGetOAuthAuthorizationNewUsecaseForTest(testutil.QueriesWithTx(tx)).Execute(context.Background(), GetOAuthAuthorizationNewInput{
			UserID: f.userID,
			Params: params,
		})
		assertRedirectableOAuthAuthorizationError(t, err, model.OAuthAuthorizationErrorInvalidTarget, params)
	})

	t.Run("公式クライアントで、resourceが無い", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		f := setupPATMember(t, tx, "goan-official-nores", model.SpaceRoleViewer, true)
		testutil.NewOAuthApplicationBuilder(t, tx).
			AsOfficialClient().
			WithClientID("goan-official-nores-client").
			Build()
		params := validOAuthAuthorizationParams("goan-official-nores-client")
		params.RedirectURI = "http://127.0.0.1:53123/callback"

		_, err := newGetOAuthAuthorizationNewUsecaseForTest(testutil.QueriesWithTx(tx)).Execute(context.Background(), GetOAuthAuthorizationNewInput{
			UserID: f.userID,
			Params: params,
		})
		assertRedirectableOAuthAuthorizationError(t, err, model.OAuthAuthorizationErrorInvalidTarget, params)
	})
}

// assertRedirectableOAuthAuthorizationErrorは、errが認可要求のリダイレクトURIとstateを持つ
// コードwantCodeのエラーであることを確かめる
func assertRedirectableOAuthAuthorizationError(t *testing.T, err error, wantCode model.OAuthAuthorizationErrorCode, params OAuthAuthorizationParams) {
	t.Helper()

	oe := model.AsOAuthAuthorizationError(err)
	if oe == nil {
		t.Fatalf("エラー = %v、期待値 = *model.OAuthAuthorizationError", err)
	}
	if oe.Code != wantCode {
		t.Errorf("Code = %q、期待値 = %q", oe.Code, wantCode)
	}
	if oe.RedirectURI != params.RedirectURI {
		t.Errorf("RedirectURI = %q、期待値 = %q", oe.RedirectURI, params.RedirectURI)
	}
	if oe.State != params.State {
		t.Errorf("State = %q、期待値 = %q", oe.State, params.State)
	}
}

// mcpAuthorizationFixtureは、MCPサーバーを宛先にした認可要求のテストの、公式クライアントと、複数の
// スペースのメンバーのユーザー
type mcpAuthorizationFixture struct {
	userID      model.UserID
	spaceIDs    []model.SpaceID
	identifiers []model.SpaceIdentifier
	clientID    string
}

// setupMCPAuthorizationFixtureは、MCPサーバーのフラグだけを持つユーザーを、keyに番号を付けた識別子の
// スペースspaces個に編集者として参加させ、公式クライアントを作る
func setupMCPAuthorizationFixture(t *testing.T, tx *sql.Tx, key string, spaces int) mcpAuthorizationFixture {
	t.Helper()

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail(key + "@example.com").
		WithAtname(strings.ReplaceAll(key, "-", "_")).
		Build()
	testutil.NewFeatureFlagBuilder(t, tx).WithUserID(userID).WithName(string(model.FeatureFlagRemoteMCPServer)).Build()

	f := mcpAuthorizationFixture{userID: userID, clientID: key + "-official"}
	for i := range spaces {
		identifier := key + "-" + string(rune('a'+i))
		spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier(identifier).Build()
		testutil.NewSpaceMemberBuilder(t, tx).WithSpaceID(spaceID).WithUserID(userID).WithRole(model.SpaceRoleEditor).Build()
		f.spaceIDs = append(f.spaceIDs, spaceID)
		f.identifiers = append(f.identifiers, model.SpaceIdentifier(identifier))
	}
	testutil.NewOAuthApplicationBuilder(t, tx).
		AsOfficialClient().
		WithClientID(f.clientID).
		WithRedirectURIs([]string{"https://example.com/callback"}).
		Build()
	return f
}

// mcpOAuthAuthorizationParamsは、clientIDのアプリのMCPサーバーを宛先にした、検証を通る認可要求のパラメーターを返す
func mcpOAuthAuthorizationParams(clientID string) OAuthAuthorizationParams {
	params := validOAuthAuthorizationParams(clientID)
	params.Resource = model.MCPServerResourceURL(oauthAuthorizationTestConfig.AppURL())
	return params
}

func TestGetOAuthAuthorizationNewUsecase_Execute_MCPServer(t *testing.T) {
	t.Parallel()

	t.Run("スペースに属さないクライアントは、スペースを選ぶ前ならメンバーであるスペースを選択肢として返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		f := setupMCPAuthorizationFixture(t, tx, "goan-mcp-options", 2)
		// メンバーでないスペースは選択肢に出さない
		testutil.NewSpaceBuilder(t, tx).WithIdentifier("goan-mcp-options-other").Build()

		output, err := newGetOAuthAuthorizationNewUsecaseForTest(testutil.QueriesWithTx(tx)).Execute(context.Background(), GetOAuthAuthorizationNewInput{
			UserID: f.userID,
			Params: mcpOAuthAuthorizationParams(f.clientID),
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if !output.NeedsSpaceSelection() {
			t.Fatalf("Space = %+v、スペースを選ぶ段を期待", output.Space)
		}
		got := make([]model.SpaceID, len(output.SpaceOptions))
		for i, option := range output.SpaceOptions {
			got[i] = option.Space.ID
			if option.DeniedReason != "" {
				t.Errorf("%sのDeniedReason = %q、期待値 = 空", option.Space.ID, option.DeniedReason)
			}
		}
		slices.Sort(got)
		want := slices.Sorted(slices.Values(f.spaceIDs))
		if !slices.Equal(got, want) {
			t.Errorf("選択肢のスペース = %v、期待値 = %v", got, want)
		}
	})

	t.Run("メンバーであるスペースが無ければ、選択肢の無い選ぶ段を返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		f := setupMCPAuthorizationFixture(t, tx, "goan-mcp-none", 0)

		output, err := newGetOAuthAuthorizationNewUsecaseForTest(testutil.QueriesWithTx(tx)).Execute(context.Background(), GetOAuthAuthorizationNewInput{
			UserID: f.userID,
			Params: mcpOAuthAuthorizationParams(f.clientID),
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if !output.NeedsSpaceSelection() || len(output.SpaceOptions) != 0 {
			t.Errorf("Space = %+v、SpaceOptions = %+v、期待値 = 選択肢の無い選ぶ段", output.Space, output.SpaceOptions)
		}
	})

	t.Run("選べるスペースが1つなら、選ぶ段を省いてそのスペースの同意画面の内容を返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		f := setupMCPAuthorizationFixture(t, tx, "goan-mcp-single", 1)

		output, err := newGetOAuthAuthorizationNewUsecaseForTest(testutil.QueriesWithTx(tx)).Execute(context.Background(), GetOAuthAuthorizationNewInput{
			UserID: f.userID,
			Params: mcpOAuthAuthorizationParams(f.clientID),
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output.NeedsSpaceSelection() || output.Space.ID != f.spaceIDs[0] || !output.SpaceSelected {
			t.Errorf("Space = %+v、SpaceSelected = %v、期待値 = 選んだものとしてスペース %v", output.Space, output.SpaceSelected, f.spaceIDs[0])
		}
	})

	t.Run("選んだスペースの同意画面の内容を返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		f := setupMCPAuthorizationFixture(t, tx, "goan-mcp-selected", 2)
		params := mcpOAuthAuthorizationParams(f.clientID)
		params.SpaceIdentifier = string(f.identifiers[1])

		output, err := newGetOAuthAuthorizationNewUsecaseForTest(testutil.QueriesWithTx(tx)).Execute(context.Background(), GetOAuthAuthorizationNewInput{
			UserID: f.userID,
			Params: params,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output.Space == nil || output.Space.ID != f.spaceIDs[1] || !output.SpaceSelected {
			t.Errorf("Space = %+v、SpaceSelected = %v、期待値 = 選んだスペース %v", output.Space, output.SpaceSelected, f.spaceIDs[1])
		}
		if output.DeniedReason != "" {
			t.Errorf("DeniedReason = %q、期待値 = 空", output.DeniedReason)
		}
	})

	t.Run("選んだスペースのメンバーでなければ、その理由を返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		f := setupMCPAuthorizationFixture(t, tx, "goan-mcp-notmember", 2)
		testutil.NewSpaceBuilder(t, tx).WithIdentifier("goan-mcp-notmember-other").Build()
		params := mcpOAuthAuthorizationParams(f.clientID)
		params.SpaceIdentifier = "goan-mcp-notmember-other"

		output, err := newGetOAuthAuthorizationNewUsecaseForTest(testutil.QueriesWithTx(tx)).Execute(context.Background(), GetOAuthAuthorizationNewInput{
			UserID: f.userID,
			Params: params,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output.DeniedReason != OAuthAuthorizationDeniedNotMember {
			t.Errorf("DeniedReason = %q、期待値 = %q", output.DeniedReason, OAuthAuthorizationDeniedNotMember)
		}
	})

	t.Run("スペースのアプリは、選ぶ段を挟まずアプリのスペースに連携する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		f := setupOAuthAuthorizationFixture(t, tx, "goan-mcp-space-app", model.SpaceRoleViewer, false)
		testutil.NewFeatureFlagBuilder(t, tx).WithUserID(f.userID).WithName(string(model.FeatureFlagRemoteMCPServer)).Build()

		output, err := newGetOAuthAuthorizationNewUsecaseForTest(testutil.QueriesWithTx(tx)).Execute(context.Background(), GetOAuthAuthorizationNewInput{
			UserID: f.userID,
			Params: mcpOAuthAuthorizationParams(f.clientID),
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output.Space == nil || output.Space.ID != f.spaceID || output.SpaceSelected {
			t.Errorf("Space = %+v、SpaceSelected = %v、期待値 = アプリのスペース %v", output.Space, output.SpaceSelected, f.spaceID)
		}
	})

	t.Run("MCPサーバーのフラグが無効なら、公開APIのフラグがあっても見つからないとして答える", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		f := setupOAuthAuthorizationFixture(t, tx, "goan-mcp-noflag", model.SpaceRoleViewer, true)

		_, err := newGetOAuthAuthorizationNewUsecaseForTest(testutil.QueriesWithTx(tx)).Execute(context.Background(), GetOAuthAuthorizationNewInput{
			UserID: f.userID,
			Params: mcpOAuthAuthorizationParams(f.clientID),
		})
		ae := model.AsAppError(err)
		if ae == nil || ae.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("エラー = %v、期待値 = AppErrCodeResourceNotFound", err)
		}
	})

	t.Run("公開APIを宛先にした要求は、MCPサーバーのフラグだけでは見つからないとして答える", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		f := setupMCPAuthorizationFixture(t, tx, "goan-mcp-public", 1)
		params := validOAuthAuthorizationParams(f.clientID)
		params.Resource = model.SpaceAPIResourceURL(oauthAuthorizationTestConfig.AppURL(), f.identifiers[0])

		_, err := newGetOAuthAuthorizationNewUsecaseForTest(testutil.QueriesWithTx(tx)).Execute(context.Background(), GetOAuthAuthorizationNewInput{
			UserID: f.userID,
			Params: params,
		})
		ae := model.AsAppError(err)
		if ae == nil || ae.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("エラー = %v、期待値 = AppErrCodeResourceNotFound", err)
		}
	})
}

func TestGetOAuthAuthorizationNewUsecase_Execute_MCPServerのクライアントへ戻すエラー(t *testing.T) {
	t.Parallel()

	t.Run("選んだスペースが存在しない", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		f := setupMCPAuthorizationFixture(t, tx, "goan-mcp-missing", 2)
		params := mcpOAuthAuthorizationParams(f.clientID)
		params.SpaceIdentifier = "goan-mcp-no-such-space"

		_, err := newGetOAuthAuthorizationNewUsecaseForTest(testutil.QueriesWithTx(tx)).Execute(context.Background(), GetOAuthAuthorizationNewInput{UserID: f.userID, Params: params})
		assertRedirectableOAuthAuthorizationError(t, err, model.OAuthAuthorizationErrorInvalidTarget, params)
	})

	t.Run("スペースのアプリでspace_identifierがある", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		f := setupOAuthAuthorizationFixture(t, tx, "goan-mcp-space-app-select", model.SpaceRoleViewer, false)
		testutil.NewFeatureFlagBuilder(t, tx).WithUserID(f.userID).WithName(string(model.FeatureFlagRemoteMCPServer)).Build()
		params := mcpOAuthAuthorizationParams(f.clientID)
		params.SpaceIdentifier = "goan-mcp-space-app-select"

		_, err := newGetOAuthAuthorizationNewUsecaseForTest(testutil.QueriesWithTx(tx)).Execute(context.Background(), GetOAuthAuthorizationNewInput{UserID: f.userID, Params: params})
		assertRedirectableOAuthAuthorizationError(t, err, model.OAuthAuthorizationErrorInvalidRequest, params)
	})

	t.Run("公開APIを宛先にした要求でspace_identifierがある", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		f := setupOAuthAuthorizationFixture(t, tx, "goan-public-select", model.SpaceRoleViewer, true)
		params := validOAuthAuthorizationParams(f.clientID)
		params.SpaceIdentifier = "goan-public-select"

		_, err := newGetOAuthAuthorizationNewUsecaseForTest(testutil.QueriesWithTx(tx)).Execute(context.Background(), GetOAuthAuthorizationNewInput{UserID: f.userID, Params: params})
		assertRedirectableOAuthAuthorizationError(t, err, model.OAuthAuthorizationErrorInvalidRequest, params)
	})

	t.Run("space_identifierが重複している", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		f := setupMCPAuthorizationFixture(t, tx, "goan-mcp-dup", 2)
		params := mcpOAuthAuthorizationParams(f.clientID)
		params.SpaceIdentifier = string(f.identifiers[0])
		params.DuplicatedParams = []string{"space_identifier"}

		_, err := newGetOAuthAuthorizationNewUsecaseForTest(testutil.QueriesWithTx(tx)).Execute(context.Background(), GetOAuthAuthorizationNewInput{UserID: f.userID, Params: params})
		assertRedirectableOAuthAuthorizationError(t, err, model.OAuthAuthorizationErrorInvalidRequest, params)
	})
}
