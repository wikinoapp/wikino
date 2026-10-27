package usecase

import (
	"context"
	"database/sql"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// oauthDeviceUserCodePatternは利用者に示すユーザーコードの形 (20種の子音を4文字ずつ区切ったもの)
var oauthDeviceUserCodePattern = regexp.MustCompile(`^[BCDFGHJKLMNPQRSTVWXZ]{4}-[BCDFGHJKLMNPQRSTVWXZ]{4}$`)

// oauthDeviceFixtureは、スペースのメンバーと、デバイス認可を要求するOAuthアプリ
type oauthDeviceFixture struct {
	patMemberFixture
	identifier model.SpaceIdentifier
	appID      model.OAuthApplicationID
	clientID   string
}

// setupOAuthDeviceFixtureは、スペースkeyのメンバーとOAuthアプリを作る。アプリは既定でスペースの
// publicクライアントで、officialClientなら公式クライアント、confidentialならconfidentialクライアントにする
func setupOAuthDeviceFixture(t *testing.T, tx *sql.Tx, key string, officialClient, confidential bool) oauthDeviceFixture {
	t.Helper()

	f := setupPATMember(t, tx, key, model.SpaceRoleEditor, true)

	clientID := key + "-client"
	appBuilder := testutil.NewOAuthApplicationBuilder(t, tx).
		WithClientID(clientID).
		WithRedirectURIs([]string{"http://127.0.0.1/callback"})
	if officialClient {
		appBuilder = appBuilder.AsOfficialClient()
	} else {
		appBuilder = appBuilder.WithSpaceID(f.spaceID)
	}
	if confidential {
		appBuilder = appBuilder.WithConfidentialClientSecretDigest(auth.DigestOpaqueToken(oauthTokenTestClientSecret))
	}

	return oauthDeviceFixture{
		patMemberFixture: f,
		identifier:       model.SpaceIdentifier(key),
		appID:            appBuilder.Build(),
		clientID:         clientID,
	}
}

// resourceはフィクスチャのスペースを指す `resource` の値を返す
func (f oauthDeviceFixture) resource() string {
	return model.SpaceAPIResourceURL(oauthAuthorizationTestConfig.AppURL(), f.identifier)
}

func newCreateOAuthDeviceAuthorizationUsecaseForTest(q *query.Queries) *CreateOAuthDeviceAuthorizationUsecase {
	return NewCreateOAuthDeviceAuthorizationUsecase(
		oauthAuthorizationTestConfig,
		repository.NewOAuthApplicationRepository(q),
		repository.NewSpaceRepository(q),
		repository.NewOAuthDeviceAuthorizationRepository(q),
	)
}

// assertOAuthDeviceAuthorizationErrorは、要求が拒否されてerrがcodeのOAuthTokenErrorであることを確かめる
func assertOAuthDeviceAuthorizationError(t *testing.T, output *CreateOAuthDeviceAuthorizationOutput, err error, code model.OAuthTokenErrorCode) {
	t.Helper()

	if output != nil {
		t.Errorf("output = %+v、期待値 = nil", output)
	}
	oe := model.AsOAuthTokenError(err)
	if oe == nil {
		t.Fatalf("エラー = %v、期待値 = %s のOAuthTokenError", err, code)
	}
	if oe.Code != code {
		t.Errorf("Code = %s、期待値 = %s", oe.Code, code)
	}
}

func TestCreateOAuthDeviceAuthorizationUsecase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("スペースのpublicクライアントの要求で、許可待ちの要求を作りコードを返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		ctx := context.Background()
		f := setupOAuthDeviceFixture(t, tx, "codev-space", false, false)

		before := time.Now()
		output, err := newCreateOAuthDeviceAuthorizationUsecaseForTest(q).Execute(ctx, OAuthDeviceAuthorizationParams{
			ClientID:   f.clientID,
			Scope:      "page:write topic:read",
			DeviceName: "my-laptop",
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		if output.DeviceCode == "" {
			t.Error("DeviceCodeが空")
		}
		if !oauthDeviceUserCodePattern.MatchString(output.UserCode) {
			t.Errorf("UserCode = %q、期待値は `BCDF-GHJK` の形", output.UserCode)
		}
		wantURI := oauthAuthorizationTestConfig.AppURL() + "/oauth/device"
		if output.VerificationURI != wantURI {
			t.Errorf("VerificationURI = %q、期待値 = %q", output.VerificationURI, wantURI)
		}
		if want := wantURI + "?user_code=" + output.UserCode; output.VerificationURIComplete != want {
			t.Errorf("VerificationURIComplete = %q、期待値 = %q", output.VerificationURIComplete, want)
		}
		if output.ExpiresIn != 10*time.Minute {
			t.Errorf("ExpiresIn = %v、期待値 = %v", output.ExpiresIn, 10*time.Minute)
		}
		if output.Interval != 5*time.Second {
			t.Errorf("Interval = %v、期待値 = %v", output.Interval, 5*time.Second)
		}

		repo := repository.NewOAuthDeviceAuthorizationRepository(q)
		stored, err := repo.FindByDeviceCodeDigest(ctx, auth.DigestOpaqueToken(output.DeviceCode))
		if err != nil {
			t.Fatalf("要求の取得に失敗: %v", err)
		}
		if stored == nil {
			t.Fatal("デバイスコードのダイジェストで要求が見つからない")
		}
		if stored.OAuthApplicationID != f.appID {
			t.Errorf("OAuthApplicationID = %v、期待値 = %v", stored.OAuthApplicationID, f.appID)
		}
		if stored.SpaceID != f.spaceID {
			t.Errorf("SpaceID = %v、期待値 = %v", stored.SpaceID, f.spaceID)
		}
		if want := []model.Scope{model.ScopeTopicRead, model.ScopePageWrite}; !slices.Equal(stored.Scopes, want) {
			t.Errorf("Scopes = %v、期待値 = %v", stored.Scopes, want)
		}
		if stored.Status != model.OAuthDeviceAuthorizationStatusPending {
			t.Errorf("Status = %v、期待値 = %v", stored.Status, model.OAuthDeviceAuthorizationStatusPending)
		}
		if stored.Interval != 5*time.Second {
			t.Errorf("Interval = %v、期待値 = %v", stored.Interval, 5*time.Second)
		}
		if stored.DeviceName == nil || *stored.DeviceName != "my-laptop" {
			t.Errorf("DeviceName = %v、期待値 = %q", stored.DeviceName, "my-laptop")
		}
		if stored.ExpiresAt.Before(before.Add(10*time.Minute)) || stored.ExpiresAt.After(time.Now().Add(10*time.Minute)) {
			t.Errorf("ExpiresAt = %v、期待値は要求から10分後", stored.ExpiresAt)
		}

		// ダイジェストだけを保存し、コードの値そのものは残さない
		if stored.DeviceCodeDigest == output.DeviceCode {
			t.Error("デバイスコードの値そのものが保存されている")
		}
		userCode := strings.ReplaceAll(output.UserCode, "-", "")
		if stored.UserCodeDigest != auth.DigestOpaqueToken(userCode) {
			t.Errorf("UserCodeDigest = %q、期待値は区切りを除いたユーザーコードのダイジェスト", stored.UserCodeDigest)
		}
		var rawCount int
		if err := tx.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM oauth_device_authorizations
			 WHERE device_code_digest IN ($1, $2, $3) OR user_code_digest IN ($1, $2, $3)`,
			output.DeviceCode, output.UserCode, userCode,
		).Scan(&rawCount); err != nil {
			t.Fatalf("保存された値の確認に失敗: %v", err)
		}
		if rawCount != 0 {
			t.Errorf("コードの値そのものを持つ行の数 = %d、期待値 = 0", rawCount)
		}
	})

	t.Run("公式クライアントは `resource` のスペースを連携先にする", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		q := testutil.QueriesWithTx(tx)
		ctx := context.Background()
		f := setupOAuthDeviceFixture(t, tx, "codev-official", true, false)

		output, err := newCreateOAuthDeviceAuthorizationUsecaseForTest(q).Execute(ctx, OAuthDeviceAuthorizationParams{
			ClientID: f.clientID,
			Scope:    "page:read",
			Resource: f.resource(),
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		stored, err := repository.NewOAuthDeviceAuthorizationRepository(q).FindByDeviceCodeDigest(ctx, auth.DigestOpaqueToken(output.DeviceCode))
		if err != nil {
			t.Fatalf("要求の取得に失敗: %v", err)
		}
		if stored == nil || stored.SpaceID != f.spaceID {
			t.Errorf("要求 = %+v、期待値はスペース %v の要求", stored, f.spaceID)
		}
	})

	t.Run("スペースのアプリは自分のスペースを指す `resource` を受け付ける", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		f := setupOAuthDeviceFixture(t, tx, "codev-own-resource", false, false)

		_, err := newCreateOAuthDeviceAuthorizationUsecaseForTest(testutil.QueriesWithTx(tx)).Execute(context.Background(), OAuthDeviceAuthorizationParams{
			ClientID: f.clientID,
			Scope:    "page:read",
			Resource: f.resource(),
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
	})

	t.Run("受け付けない要求", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		uc := newCreateOAuthDeviceAuthorizationUsecaseForTest(testutil.QueriesWithTx(tx))
		space := setupOAuthDeviceFixture(t, tx, "codev-reject", false, false)
		official := setupOAuthDeviceFixture(t, tx, "codev-reject-official", true, false)
		confidential := setupOAuthDeviceFixture(t, tx, "codev-reject-conf", false, true)
		other := setupOAuthDeviceFixture(t, tx, "codev-reject-other", false, false)

		tests := []struct {
			name   string
			params OAuthDeviceAuthorizationParams
			want   model.OAuthTokenErrorCode
		}{
			{
				name:   "client_idが無い",
				params: OAuthDeviceAuthorizationParams{Scope: "page:read"},
				want:   model.OAuthTokenErrorInvalidRequest,
			},
			{
				name:   "パラメーターの重複",
				params: OAuthDeviceAuthorizationParams{ClientID: space.clientID, Scope: "page:read", DuplicatedParams: []string{"scope"}},
				want:   model.OAuthTokenErrorInvalidRequest,
			},
			{
				name:   "100文字を超える `device_name`",
				params: OAuthDeviceAuthorizationParams{ClientID: space.clientID, Scope: "page:read", DeviceName: strings.Repeat("a", 101)},
				want:   model.OAuthTokenErrorInvalidRequest,
			},
			{
				name:   "制御文字を含む `device_name`",
				params: OAuthDeviceAuthorizationParams{ClientID: space.clientID, Scope: "page:read", DeviceName: "my\nlaptop"},
				want:   model.OAuthTokenErrorInvalidRequest,
			},
			{
				name:   "登録の無いclient_id",
				params: OAuthDeviceAuthorizationParams{ClientID: "codev-unknown-client", Scope: "page:read"},
				want:   model.OAuthTokenErrorInvalidClient,
			},
			{
				name:   "confidentialクライアント",
				params: OAuthDeviceAuthorizationParams{ClientID: confidential.clientID, Scope: "page:read"},
				want:   model.OAuthTokenErrorUnauthorizedClient,
			},
			{
				name:   "クライアントの認証を試みたpublicクライアント",
				params: OAuthDeviceAuthorizationParams{ClientID: space.clientID, Scope: "page:read", ClientAuthenticated: true},
				want:   model.OAuthTokenErrorInvalidClient,
			},
			{
				name:   "スコープが無い",
				params: OAuthDeviceAuthorizationParams{ClientID: space.clientID},
				want:   model.OAuthTokenErrorInvalidScope,
			},
			{
				name:   "不明なスコープ",
				params: OAuthDeviceAuthorizationParams{ClientID: space.clientID, Scope: "page:read unknown:read"},
				want:   model.OAuthTokenErrorInvalidScope,
			},
			{
				name:   "トークンに付与できないスコープ",
				params: OAuthDeviceAuthorizationParams{ClientID: space.clientID, Scope: "oauth_application:write"},
				want:   model.OAuthTokenErrorInvalidScope,
			},
			{
				name:   "公式クライアントで `resource` が無い",
				params: OAuthDeviceAuthorizationParams{ClientID: official.clientID, Scope: "page:read"},
				want:   model.OAuthTokenErrorInvalidTarget,
			},
			{
				name:   "形式が不正な `resource`",
				params: OAuthDeviceAuthorizationParams{ClientID: official.clientID, Scope: "page:read", Resource: official.resource() + "/"},
				want:   model.OAuthTokenErrorInvalidTarget,
			},
			{
				name:   "存在しないスペースを指す `resource`",
				params: OAuthDeviceAuthorizationParams{ClientID: official.clientID, Scope: "page:read", Resource: model.SpaceAPIResourceURL(oauthAuthorizationTestConfig.AppURL(), "codev-missing")},
				want:   model.OAuthTokenErrorInvalidTarget,
			},
			{
				name:   "スペースのアプリでほかのスペースを指す `resource`",
				params: OAuthDeviceAuthorizationParams{ClientID: space.clientID, Scope: "page:read", Resource: other.resource()},
				want:   model.OAuthTokenErrorInvalidTarget,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				output, err := uc.Execute(context.Background(), tt.params)
				assertOAuthDeviceAuthorizationError(t, output, err, tt.want)
			})
		}

		var count int
		if err := tx.QueryRowContext(context.Background(),
			`SELECT COUNT(*) FROM oauth_device_authorizations WHERE space_id IN ($1, $2, $3, $4)`,
			space.spaceID, official.spaceID, confidential.spaceID, other.spaceID,
		).Scan(&count); err != nil {
			t.Fatalf("要求の数の確認に失敗: %v", err)
		}
		if count != 0 {
			t.Errorf("受け付けない要求で作られた要求の数 = %d、期待値 = 0", count)
		}
	})
}

// TestCreateOAuthDeviceAuthorizationUsecase_Execute_UserCodeConflictは、生成したユーザーコードが既存の
// 要求と衝突したときの作り直しを確かめる。一意制約違反でトランザクションが中断すると作り直せないため、
// 前提データをコミットし、UseCaseはトランザクションで包まない接続で動かす。
func TestCreateOAuthDeviceAuthorizationUsecase_Execute_UserCodeConflict(t *testing.T) {
	t.Parallel()

	// setupConflictは、ユーザーコードexistingCodeの要求を持つスペースのpublicクライアントを作ってコミットする
	setupConflict := func(t *testing.T, key, existingCode string) oauthDeviceFixture {
		t.Helper()

		tx, err := testutil.GetTestDB().Begin()
		if err != nil {
			t.Fatalf("トランザクションの開始に失敗: %v", err)
		}
		defer func() { _ = tx.Rollback() }()
		f := setupOAuthDeviceFixture(t, tx, key, false, false)
		testutil.NewOAuthDeviceAuthorizationBuilder(t, tx).
			WithOAuthApplicationID(f.appID).
			WithSpaceID(f.spaceID).
			WithDeviceCodeDigest(key + "-device-code-digest").
			WithUserCodeDigest(auth.DigestOpaqueToken(existingCode)).
			Build()
		if err := tx.Commit(); err != nil {
			t.Fatalf("前提データのコミットに失敗: %v", err)
		}
		return f
	}

	// newUsecaseは、codesを順に返す生成関数を差し替えたUseCaseと、生成関数を呼んだ回数を返す
	newUsecase := func(codes ...string) (*CreateOAuthDeviceAuthorizationUsecase, *int) {
		uc := newCreateOAuthDeviceAuthorizationUsecaseForTest(query.New(testutil.GetTestDB()))
		calls := 0
		uc.generateUserCode = func() (string, error) {
			code := codes[min(calls, len(codes)-1)]
			calls++
			return code, nil
		}
		return uc, &calls
	}

	t.Run("衝突したら作り直したユーザーコードで要求を作る", func(t *testing.T) {
		t.Parallel()

		f := setupConflict(t, "codev-conflict-retry", "BCDFGHJK")
		uc, calls := newUsecase("BCDFGHJK", "CDFGHJKL")

		output, err := uc.Execute(context.Background(), OAuthDeviceAuthorizationParams{ClientID: f.clientID, Scope: "page:read"})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if output.UserCode != "CDFG-HJKL" {
			t.Errorf("UserCode = %q、期待値 = %q", output.UserCode, "CDFG-HJKL")
		}
		if *calls != 2 {
			t.Errorf("ユーザーコードを生成した回数 = %d、期待値 = 2", *calls)
		}

		stored, err := repository.NewOAuthDeviceAuthorizationRepository(query.New(testutil.GetTestDB())).
			FindByDeviceCodeDigest(context.Background(), auth.DigestOpaqueToken(output.DeviceCode))
		if err != nil {
			t.Fatalf("要求の取得に失敗: %v", err)
		}
		if stored == nil {
			t.Fatal("デバイスコードのダイジェストで要求が見つからない")
		}
		if want := auth.DigestOpaqueToken("CDFGHJKL"); stored.UserCodeDigest != want {
			t.Errorf("UserCodeDigest = %q、期待値は作り直したユーザーコードのダイジェスト", stored.UserCodeDigest)
		}
	})

	t.Run("上限まで衝突し続けたらシステムエラーを返す", func(t *testing.T) {
		t.Parallel()

		f := setupConflict(t, "codev-conflict-exhaust", "DFGHJKLM")
		uc, calls := newUsecase("DFGHJKLM")

		output, err := uc.Execute(context.Background(), OAuthDeviceAuthorizationParams{ClientID: f.clientID, Scope: "page:read"})
		if output != nil {
			t.Errorf("output = %+v、期待値 = nil", output)
		}
		if err == nil {
			t.Fatal("エラーを期待したが、nilだった")
		}
		if oe := model.AsOAuthTokenError(err); oe != nil {
			t.Errorf("エラー = %s のOAuthTokenError、期待値はシステムエラー", oe.Code)
		}
		if *calls != oauthUserCodeMaxAttempts {
			t.Errorf("ユーザーコードを生成した回数 = %d、期待値 = %d", *calls, oauthUserCodeMaxAttempts)
		}

		var count int
		if err := testutil.GetTestDB().QueryRowContext(context.Background(),
			`SELECT COUNT(*) FROM oauth_device_authorizations WHERE space_id = $1`, f.spaceID,
		).Scan(&count); err != nil {
			t.Fatalf("要求の数の確認に失敗: %v", err)
		}
		if count != 1 {
			t.Errorf("スペースの要求の数 = %d、期待値は既存の1件だけ", count)
		}
	})
}
