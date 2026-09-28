package usecase

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"regexp"
	"slices"
	"time"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// codeVerifierPatternはPKCEのcode_verifierの形式 (RFC 7636 §4.1)
var codeVerifierPattern = regexp.MustCompile(`^[A-Za-z0-9._~-]{43,128}$`)

// CreateOAuthTokenUsecaseは、トークン要求 (`POST /oauth/token`) に応じて、認可コードの交換か
// リフレッシュトークンの更新でアクセストークンとリフレッシュトークンを発行する。
//
// 認可コード・リフレッシュトークンをダイジェストで引くクエリは `space_id` の規約の例外である。
// トークン要求はスペースを特定する前の入口で、スペースはコードやトークンから決まるため。
// 以降のアプリ・許可・持ち主の取得は、ここで得た `space_id` を条件に含める。
type CreateOAuthTokenUsecase struct {
	cfg                        *config.Config
	oauthApplicationRepo       *repository.OAuthApplicationRepository
	oauthGrantRepo             *repository.OAuthGrantRepository
	oauthAuthorizationCodeRepo *repository.OAuthAuthorizationCodeRepository
	oauthAccessTokenRepo       *repository.OAuthAccessTokenRepository
	oauthRefreshTokenRepo      *repository.OAuthRefreshTokenRepository
	owners                     *apiTokenOwnerFinder
}

// NewCreateOAuthTokenUsecaseはCreateOAuthTokenUsecaseを生成する
func NewCreateOAuthTokenUsecase(
	cfg *config.Config,
	oauthApplicationRepo *repository.OAuthApplicationRepository,
	oauthGrantRepo *repository.OAuthGrantRepository,
	oauthAuthorizationCodeRepo *repository.OAuthAuthorizationCodeRepository,
	oauthAccessTokenRepo *repository.OAuthAccessTokenRepository,
	oauthRefreshTokenRepo *repository.OAuthRefreshTokenRepository,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	userRepo *repository.UserRepository,
	featureFlagRepo *repository.FeatureFlagRepository,
) *CreateOAuthTokenUsecase {
	return &CreateOAuthTokenUsecase{
		cfg:                        cfg,
		oauthApplicationRepo:       oauthApplicationRepo,
		oauthGrantRepo:             oauthGrantRepo,
		oauthAuthorizationCodeRepo: oauthAuthorizationCodeRepo,
		oauthAccessTokenRepo:       oauthAccessTokenRepo,
		oauthRefreshTokenRepo:      oauthRefreshTokenRepo,
		owners:                     newAPITokenOwnerFinder(spaceRepo, spaceMemberRepo, userRepo, featureFlagRepo),
	}
}

// OAuthTokenParamsはトークン要求のパラメーター (RFC 6749 §4.1.3・§6、RFC 7636 §4.5、RFC 8707 §2)
type OAuthTokenParams struct {
	GrantType    string
	Code         string
	RedirectURI  string
	CodeVerifier string
	RefreshToken string
	Scope        string
	Resource     string

	// ClientIDとClientSecretは、`Authorization: Basic` (client_secret_basic) か本文
	// (client_secret_post) から取り出したクライアントの資格情報。ハンドラーが2つの方式の
	// 同時の使用を拒否し、どちらか一方の値を渡す。publicクライアントはシークレットを送らない
	ClientID     string
	ClientSecret string
	// ClientSecretProvidedは値が空でも資格情報を送った場合にtrue。publicクライアントでは拒否する
	ClientSecretProvided bool

	// DuplicatedParamsは2回以上送られたパラメーターの名前。パラメーターは1回しか送れない
	// (RFC 6749 §3.2) ため、どの値を採るかを決めずに拒否する
	DuplicatedParams []string
}

// CreateOAuthTokenOutputはトークンレスポンス (RFC 6749 §5.1) の値。トークンの値そのものは
// ここでしか得られず、データベースにはダイジェストしか残らない
type CreateOAuthTokenOutput struct {
	AccessToken  string
	ExpiresIn    time.Duration
	RefreshToken string
	Scopes       []model.Scope
}

// oauthTokenIssuanceは発行するトークンの値
type oauthTokenIssuance struct {
	accessToken  string
	refreshToken string
}

// Executeはトークン要求を処理してトークンを発行する。要求を受け付けられない場合は
// *model.OAuthTokenErrorを返す。
//
// 使用済みの認可コード・リフレッシュトークンが再び提示された場合は、漏洩したものとして
// 許可に属するアクセストークン・リフレッシュトークンをすべて失効させてから `invalid_grant` を返す。
func (uc *CreateOAuthTokenUsecase) Execute(ctx context.Context, params OAuthTokenParams) (*CreateOAuthTokenOutput, error) {
	if len(params.DuplicatedParams) > 0 {
		return nil, oauthTokenError(model.OAuthTokenErrorInvalidRequest)
	}

	switch params.GrantType {
	case model.OAuthGrantTypeAuthorizationCode:
		return uc.exchangeAuthorizationCode(ctx, params)
	case model.OAuthGrantTypeRefreshToken:
		return uc.refresh(ctx, params)
	case "":
		return nil, oauthTokenError(model.OAuthTokenErrorInvalidRequest)
	default:
		return nil, oauthTokenError(model.OAuthTokenErrorUnsupportedGrantType)
	}
}

// exchangeAuthorizationCodeは認可コードをトークンと交換する (RFC 6749 §4.1.3)
func (uc *CreateOAuthTokenUsecase) exchangeAuthorizationCode(ctx context.Context, params OAuthTokenParams) (*CreateOAuthTokenOutput, error) {
	// 1. データ取得と検証
	if params.Code == "" || params.RedirectURI == "" || params.CodeVerifier == "" || params.ClientID == "" {
		return nil, oauthTokenError(model.OAuthTokenErrorInvalidRequest)
	}
	if !codeVerifierPattern.MatchString(params.CodeVerifier) {
		return nil, oauthTokenError(model.OAuthTokenErrorInvalidRequest)
	}

	code, err := uc.oauthAuthorizationCodeRepo.FindByCodeDigest(ctx, auth.DigestOpaqueToken(params.Code))
	if err != nil {
		return nil, fmt.Errorf("認可コードの取得に失敗: %w", err)
	}
	if code == nil {
		return nil, oauthTokenError(model.OAuthTokenErrorInvalidGrant)
	}

	grant, err := uc.authenticateClient(ctx, params, code.OAuthGrantID, code.SpaceID)
	if err != nil {
		return nil, err
	}

	// 認可要求を送った本人であること (リダイレクトURI・PKCE) を、再利用の検知より前に確かめる。
	// publicクライアントの認証はclient_idの一致だけのため、先に失効させると、使用済みのコードを
	// 拾っただけの第三者が許可のトークンをすべて失効させられる。
	// code_challengeの無いコードは作らないが、ダウングレード (PKCEを外した交換) を
	// 受け付けないよう、ここでも空のcode_challengeを一致とみなさない
	if params.RedirectURI != code.RedirectURI || code.CodeChallenge == "" || !verifyPKCES256(params.CodeVerifier, code.CodeChallenge) {
		return nil, oauthTokenError(model.OAuthTokenErrorInvalidGrant)
	}

	// 使用済みのコードの再提示は、コードが漏れて別の誰かが先に交換したことを示す (RFC 6749 §4.1.2)
	if code.IsUsed() {
		return nil, uc.revokeReplayedGrant(ctx, grant)
	}

	now := time.Now()
	if code.IsExpired(now) || grant.IsRevoked() {
		return nil, oauthTokenError(model.OAuthTokenErrorInvalidGrant)
	}

	// 2. 認可チェック
	if err := uc.checkGrantUsable(ctx, grant, params.Resource); err != nil {
		return nil, err
	}

	// 3. 永続化
	issuance, err := newOAuthTokenIssuance()
	if err != nil {
		return nil, err
	}
	exchanged, err := uc.oauthAuthorizationCodeRepo.Exchange(ctx, repository.ExchangeOAuthAuthorizationCodeInput{
		ID:                    code.ID,
		SpaceID:               code.SpaceID,
		AccessTokenDigest:     auth.DigestOpaqueToken(issuance.accessToken),
		AccessTokenExpiresAt:  now.Add(model.OAuthAccessTokenLifetime),
		RefreshTokenDigest:    auth.DigestOpaqueToken(issuance.refreshToken),
		RefreshTokenExpiresAt: now.Add(model.OAuthRefreshTokenLifetime),
		Now:                   now,
	})
	if err != nil {
		return nil, fmt.Errorf("認可コードの交換に失敗: %w", err)
	}
	if exchanged == nil {
		// 取得の後に同じコードが交換された。同時に届いた2つの要求の一方は漏れたコードである
		return nil, uc.revokeReplayedGrant(ctx, grant)
	}

	return issuance.output(code.Scopes), nil
}

// refreshはリフレッシュトークンをローテーションし、新しいアクセストークンを発行する (RFC 6749 §6)
func (uc *CreateOAuthTokenUsecase) refresh(ctx context.Context, params OAuthTokenParams) (*CreateOAuthTokenOutput, error) {
	// 1. データ取得と検証
	if params.RefreshToken == "" || params.ClientID == "" {
		return nil, oauthTokenError(model.OAuthTokenErrorInvalidRequest)
	}

	refreshToken, err := uc.oauthRefreshTokenRepo.FindByTokenDigest(ctx, auth.DigestOpaqueToken(params.RefreshToken))
	if err != nil {
		return nil, fmt.Errorf("リフレッシュトークンの取得に失敗: %w", err)
	}
	if refreshToken == nil {
		return nil, oauthTokenError(model.OAuthTokenErrorInvalidGrant)
	}

	grant, err := uc.authenticateClient(ctx, params, refreshToken.OAuthGrantID, refreshToken.SpaceID)
	if err != nil {
		return nil, err
	}

	// ローテーション済みのトークンの再提示は、トークンが漏れたことを示す (RFC 9700 §4.14.2)
	if refreshToken.IsUsed() {
		return nil, uc.revokeReplayedGrant(ctx, grant)
	}

	now := time.Now()
	if !refreshToken.IsActive(now) || grant.IsRevoked() {
		return nil, oauthTokenError(model.OAuthTokenErrorInvalidGrant)
	}

	// 要求のスコープで元の範囲を狭められるが、広げることはできない (RFC 6749 §6)。
	// 狭めるのは今回のアクセストークンだけで、リフレッシュトークンは元の範囲を持ち続ける
	scopes := refreshToken.Scopes
	if params.Scope != "" {
		requested, ok := parseOAuthScopes(params.Scope)
		if !ok {
			return nil, oauthTokenError(model.OAuthTokenErrorInvalidScope)
		}
		for _, s := range requested {
			if !slices.Contains(refreshToken.Scopes, s) {
				return nil, oauthTokenError(model.OAuthTokenErrorInvalidScope)
			}
		}
		scopes = requested
	}

	// 2. 認可チェック
	if err := uc.checkGrantUsable(ctx, grant, params.Resource); err != nil {
		return nil, err
	}

	// 3. 永続化
	issuance, err := newOAuthTokenIssuance()
	if err != nil {
		return nil, err
	}
	rotated, err := uc.oauthRefreshTokenRepo.Rotate(ctx, repository.RotateOAuthRefreshTokenInput{
		ID:                   refreshToken.ID,
		SpaceID:              refreshToken.SpaceID,
		TokenDigest:          auth.DigestOpaqueToken(issuance.refreshToken),
		ExpiresAt:            now.Add(model.OAuthRefreshTokenLifetime),
		AccessTokenDigest:    auth.DigestOpaqueToken(issuance.accessToken),
		AccessTokenScopes:    scopes,
		AccessTokenExpiresAt: now.Add(model.OAuthAccessTokenLifetime),
		Now:                  now,
	})
	if err != nil {
		return nil, fmt.Errorf("リフレッシュトークンのローテーションに失敗: %w", err)
	}
	if rotated == nil {
		// 取得の後に同じトークンがローテーションされた。同時に届いた2つの要求の一方は漏れたトークンである
		return nil, uc.revokeReplayedGrant(ctx, grant)
	}

	return issuance.output(scopes), nil
}

// authenticateClientは、クライアントを認証し (RFC 6749 §2.3.1)、認可コードやリフレッシュトークンが
// 属する許可がそのクライアントのものであることを確かめて、許可を返す。
//
// アプリは、コードやトークンのスペースで使えるものに限って引く (スペースのアプリか公式クライアント)。
// 認証できないクライアントには `invalid_client`、別のクライアントに発行されたコードやトークンには
// `invalid_grant` を返す。
func (uc *CreateOAuthTokenUsecase) authenticateClient(ctx context.Context, params OAuthTokenParams, grantID model.OAuthGrantID, spaceID model.SpaceID) (*model.OAuthGrant, error) {
	app, err := uc.oauthApplicationRepo.FindByClientIDAndSpaceID(ctx, params.ClientID, spaceID)
	if err != nil {
		return nil, fmt.Errorf("OAuthアプリの取得に失敗: %w", err)
	}
	if app == nil || !app.IsAvailableIn(spaceID) || !verifyOAuthClientSecret(app, params.ClientSecret, params.ClientSecretProvided) {
		return nil, oauthTokenError(model.OAuthTokenErrorInvalidClient)
	}

	grant, err := uc.oauthGrantRepo.FindByID(ctx, grantID, spaceID)
	if err != nil {
		return nil, fmt.Errorf("OAuthアプリの許可の取得に失敗: %w", err)
	}
	if grant == nil || grant.OAuthApplicationID != app.ID {
		return nil, oauthTokenError(model.OAuthTokenErrorInvalidGrant)
	}
	return grant, nil
}

// checkGrantUsableは、許可の持ち主が束縛先のスペースで今もトークンを使えることと、要求の
// `resource` が束縛先のスペースを指すことを確かめる。
//
// 持ち主の有効性はトークンを使うリクエストのたびにも確かめるが、使えないトークンを発行しない
// よう、発行の時点でも確かめる。
func (uc *CreateOAuthTokenUsecase) checkGrantUsable(ctx context.Context, grant *model.OAuthGrant, resource string) error {
	owner, err := uc.owners.find(ctx, grant.SpaceID, grant.SpaceMemberID)
	if err != nil {
		return err
	}
	if owner == nil || !policy.NewMemberPolicy(owner.SpaceMember.Scopes, nil).CanCreateOAuthGrant() {
		return oauthTokenError(model.OAuthTokenErrorInvalidGrant)
	}

	if resource != "" {
		identifier, ok := model.ParseSpaceAPIResourceURL(uc.cfg.AppURL(), resource)
		if !ok || identifier != owner.Space.Identifier {
			return oauthTokenError(model.OAuthTokenErrorInvalidTarget)
		}
	}
	return nil
}

// revokeReplayedGrantは、使用済みの認可コードやリフレッシュトークンが再び提示された許可の
// アクセストークンとリフレッシュトークンをすべて失効させ、invalid_grantを返す。
// 許可そのものは失効させず、利用者は再び認可すれば連携を続けられる
func (uc *CreateOAuthTokenUsecase) revokeReplayedGrant(ctx context.Context, grant *model.OAuthGrant) error {
	now := time.Now()
	// 長く使えるリフレッシュトークンを先に失効させる
	if err := uc.oauthRefreshTokenRepo.RevokeByOAuthGrant(ctx, grant.ID, grant.SpaceID, now); err != nil {
		return fmt.Errorf("リフレッシュトークンの失効に失敗: %w", err)
	}
	if err := uc.oauthAccessTokenRepo.RevokeByOAuthGrant(ctx, grant.ID, grant.SpaceID, now); err != nil {
		return fmt.Errorf("アクセストークンの失効に失敗: %w", err)
	}
	return oauthTokenError(model.OAuthTokenErrorInvalidGrant)
}

// newOAuthTokenIssuanceは発行するアクセストークンとリフレッシュトークンの値を生成する
func newOAuthTokenIssuance() (*oauthTokenIssuance, error) {
	accessToken, err := auth.GenerateOpaqueToken(auth.OAuthAccessTokenPrefix)
	if err != nil {
		return nil, fmt.Errorf("アクセストークンの生成に失敗: %w", err)
	}
	refreshToken, err := auth.GenerateOpaqueToken(auth.OAuthRefreshTokenPrefix)
	if err != nil {
		return nil, fmt.Errorf("リフレッシュトークンの生成に失敗: %w", err)
	}
	return &oauthTokenIssuance{accessToken: accessToken, refreshToken: refreshToken}, nil
}

// outputは発行したトークンをトークンレスポンスの値にする。scopesはアクセストークンのスコープ
func (i *oauthTokenIssuance) output(scopes []model.Scope) *CreateOAuthTokenOutput {
	return &CreateOAuthTokenOutput{
		AccessToken:  i.accessToken,
		ExpiresIn:    model.OAuthAccessTokenLifetime,
		RefreshToken: i.refreshToken,
		Scopes:       scopes,
	}
}

// verifyOAuthClientSecretは、クライアントの送ったシークレットが登録したものと一致するかを返す。
// confidentialクライアントはシークレットを必須とし、publicクライアントはシークレットを
// 送ってはならない (送られたら、クライアントの設定の誤りとして認証に失敗させる)
func verifyOAuthClientSecret(app *model.OAuthApplication, secret string, provided bool) bool {
	if !app.IsConfidential() {
		return !provided && secret == ""
	}
	if secret == "" || app.ClientSecretDigest == nil {
		return false
	}
	digest := auth.DigestOpaqueToken(secret)
	return subtle.ConstantTimeCompare([]byte(digest), []byte(*app.ClientSecretDigest)) == 1
}

// verifyPKCES256は、code_verifierのS256 (SHA-256をbase64urlで表したもの) がcode_challengeと
// 一致するかを返す (RFC 7636 §4.6)
func verifyPKCES256(verifier, challenge string) bool {
	hash := sha256.Sum256([]byte(verifier))
	computed := base64.RawURLEncoding.EncodeToString(hash[:])
	return subtle.ConstantTimeCompare([]byte(computed), []byte(challenge)) == 1
}

// oauthTokenErrorはトークン要求のエラーを返す
func oauthTokenError(code model.OAuthTokenErrorCode) error {
	return &model.OAuthTokenError{Code: code}
}
