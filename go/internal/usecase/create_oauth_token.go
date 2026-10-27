package usecase

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// codeVerifierPatternはPKCEのcode_verifierの形式 (RFC 7636 §4.1)
var codeVerifierPattern = regexp.MustCompile(`^[A-Za-z0-9._~-]{43,128}$`)

// CreateOAuthTokenUsecaseは、トークン要求 (`POST /oauth/token`) に応じて、認可コードの交換・
// リフレッシュトークンの更新・デバイスコードの交換でアクセストークンとリフレッシュトークンを発行する。
//
// 認可コード・リフレッシュトークン・デバイスコードをダイジェストで引くクエリは `space_id` の規約の
// 例外である。トークン要求はスペースを特定する前の入口で、スペースはコードやトークンから決まるため。
// 以降のアプリ・許可・持ち主の取得は、ここで得た `space_id` を条件に含める。
type CreateOAuthTokenUsecase struct {
	db                           *sql.DB
	cfg                          *config.Config
	oauthApplicationRepo         *repository.OAuthApplicationRepository
	oauthGrantRepo               *repository.OAuthGrantRepository
	oauthAuthorizationCodeRepo   *repository.OAuthAuthorizationCodeRepository
	oauthRefreshTokenRepo        *repository.OAuthRefreshTokenRepository
	oauthDeviceAuthorizationRepo *repository.OAuthDeviceAuthorizationRepository
	owners                       *apiTokenOwnerFinder
}

// NewCreateOAuthTokenUsecaseはCreateOAuthTokenUsecaseを生成する
func NewCreateOAuthTokenUsecase(
	db *sql.DB,
	cfg *config.Config,
	oauthApplicationRepo *repository.OAuthApplicationRepository,
	oauthGrantRepo *repository.OAuthGrantRepository,
	oauthAuthorizationCodeRepo *repository.OAuthAuthorizationCodeRepository,
	oauthRefreshTokenRepo *repository.OAuthRefreshTokenRepository,
	oauthDeviceAuthorizationRepo *repository.OAuthDeviceAuthorizationRepository,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	userRepo *repository.UserRepository,
	featureFlagRepo *repository.FeatureFlagRepository,
) *CreateOAuthTokenUsecase {
	return &CreateOAuthTokenUsecase{
		db:                           db,
		cfg:                          cfg,
		oauthApplicationRepo:         oauthApplicationRepo,
		oauthGrantRepo:               oauthGrantRepo,
		oauthAuthorizationCodeRepo:   oauthAuthorizationCodeRepo,
		oauthRefreshTokenRepo:        oauthRefreshTokenRepo,
		oauthDeviceAuthorizationRepo: oauthDeviceAuthorizationRepo,
		owners:                       newAPITokenOwnerFinder(spaceRepo, spaceMemberRepo, userRepo, featureFlagRepo),
	}
}

// OAuthTokenParamsはトークン要求のパラメーター (RFC 6749 §4.1.3・§6、RFC 7636 §4.5、RFC 8628 §3.4、RFC 8707 §2)
type OAuthTokenParams struct {
	GrantType    string
	Code         string
	RedirectURI  string
	CodeVerifier string
	RefreshToken string
	DeviceCode   string
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
// 使用済みの認可コード・リフレッシュトークン・交換済みのデバイスコードが再び提示された場合は、
// 漏洩したものとして許可に属するアクセストークン・リフレッシュトークンをすべて失効させてから
// `invalid_grant` を返す。
func (uc *CreateOAuthTokenUsecase) Execute(ctx context.Context, params OAuthTokenParams) (*CreateOAuthTokenOutput, error) {
	if len(params.DuplicatedParams) > 0 {
		return nil, oauthTokenError(model.OAuthTokenErrorInvalidRequest)
	}

	switch params.GrantType {
	case model.OAuthGrantTypeAuthorizationCode:
		return uc.exchangeAuthorizationCode(ctx, params)
	case model.OAuthGrantTypeRefreshToken:
		return uc.refresh(ctx, params)
	case model.OAuthGrantTypeDeviceCode:
		return uc.exchangeDeviceCode(ctx, params)
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

	// 許可のスコープは作成後に変わらず、コードは許可の範囲で作る。範囲を超えるトークンを発行しないよう、
	// 不変条件として発行の前にも確かめる
	now := time.Now()
	if code.IsExpired(now) || grant.IsRevoked() || !model.HasAllScopes(grant.Scopes, code.Scopes) {
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
	exchanged, err := uc.exchangeCode(ctx, code, repository.ExchangeOAuthAuthorizationCodeInput{
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
	rotated, err := uc.rotateToken(ctx, refreshToken, repository.RotateOAuthRefreshTokenInput{
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

// exchangeDeviceCodeは、利用者が許可したデバイスコードをトークンと交換する (RFC 8628 §3.4・§3.5)。
// 許可待ちなら `authorization_pending` か `slow_down`、拒否されていれば `access_denied`、
// 期限が切れていれば `expired_token` を返し、クライアントはそれに応じてポーリングを続けるか止める。
//
// 要求の `scope` は使わない。スコープはデバイス認可の要求と利用者の同意で決まっており、
// クライアント (golang.org/x/oauth2など) はポーリングにも要求の時点のスコープを付けて送るため、
// 利用者が同意画面で狭めた範囲と照合すると交換できなくなる
func (uc *CreateOAuthTokenUsecase) exchangeDeviceCode(ctx context.Context, params OAuthTokenParams) (*CreateOAuthTokenOutput, error) {
	// 1. データ取得と検証
	if params.DeviceCode == "" || params.ClientID == "" {
		return nil, oauthTokenError(model.OAuthTokenErrorInvalidRequest)
	}

	device, err := uc.oauthDeviceAuthorizationRepo.FindByDeviceCodeDigest(ctx, auth.DigestOpaqueToken(params.DeviceCode))
	if err != nil {
		return nil, fmt.Errorf("デバイス認可の要求の取得に失敗: %w", err)
	}
	if device == nil {
		return nil, oauthTokenError(model.OAuthTokenErrorInvalidGrant)
	}

	app, err := uc.authenticateApplication(ctx, params, device.SpaceID)
	if err != nil {
		return nil, err
	}
	if app.ID != device.OAuthApplicationID {
		return nil, oauthTokenError(model.OAuthTokenErrorInvalidGrant)
	}

	now := time.Now()
	switch device.Status {
	case model.OAuthDeviceAuthorizationStatusDenied:
		return nil, oauthTokenError(model.OAuthTokenErrorAccessDenied)
	case model.OAuthDeviceAuthorizationStatusExchanged:
		// 交換済みのデバイスコードの再提示は、コードが漏れて別の誰かが先に交換したことを示す。
		// デバイスコードは要求したクライアントしか知らないため、認可コードと同じく許可のトークンを失効させる
		grant, err := uc.findGrant(ctx, app, *device.OAuthGrantID, device.SpaceID)
		if err != nil {
			return nil, err
		}
		return nil, uc.revokeReplayedGrant(ctx, grant)
	}
	if device.IsExpired(now) {
		return nil, oauthTokenError(model.OAuthTokenErrorExpiredToken)
	}
	if device.Status == model.OAuthDeviceAuthorizationStatusPending {
		return nil, uc.pollPendingDeviceAuthorization(ctx, device, now)
	}

	grant, err := uc.findGrant(ctx, app, *device.OAuthGrantID, device.SpaceID)
	if err != nil {
		return nil, err
	}
	// 許可した後に利用者が連携を解除した場合はトークンを作らない。スコープが許可の範囲に収まることも、
	// 認可コードの交換と同じく不変条件として確かめる
	if grant.IsRevoked() || !model.HasAllScopes(grant.Scopes, device.Scopes) {
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
	exchanged, err := uc.exchangeDevice(ctx, device, repository.ExchangeOAuthDeviceAuthorizationInput{
		ID:                    device.ID,
		SpaceID:               device.SpaceID,
		AccessTokenDigest:     auth.DigestOpaqueToken(issuance.accessToken),
		AccessTokenExpiresAt:  now.Add(model.OAuthAccessTokenLifetime),
		RefreshTokenDigest:    auth.DigestOpaqueToken(issuance.refreshToken),
		RefreshTokenExpiresAt: now.Add(model.OAuthRefreshTokenLifetime),
		Now:                   now,
	})
	if err != nil {
		return nil, fmt.Errorf("デバイスコードの交換に失敗: %w", err)
	}
	if exchanged == nil {
		// 取得の後に同じデバイスコードが交換された。同時に届いた2つの要求の一方は漏れたコードである
		return nil, uc.revokeReplayedGrant(ctx, grant)
	}

	return issuance.output(device.Scopes), nil
}

// pollPendingDeviceAuthorizationは、許可待ちの要求へのポーリングを記録し、前回のポーリングから
// 間隔を空けずに届いていれば `slow_down` を、そうでなければ `authorization_pending` を返す。
// 取得の後に利用者が許可・拒否した場合も `authorization_pending` を返し、次のポーリングで結果を伝える
func (uc *CreateOAuthTokenUsecase) pollPendingDeviceAuthorization(ctx context.Context, device *model.OAuthDeviceAuthorization, now time.Time) error {
	tooFast, err := uc.oauthDeviceAuthorizationRepo.RecordPoll(ctx, device.ID, device.SpaceID, model.OAuthDeviceAuthorizationSlowDownIncrement, now)
	if err != nil {
		return fmt.Errorf("デバイス認可の要求へのポーリングの記録に失敗: %w", err)
	}
	if tooFast {
		return oauthTokenError(model.OAuthTokenErrorSlowDown)
	}
	return oauthTokenError(model.OAuthTokenErrorAuthorizationPending)
}

// authenticateClientは、クライアントを認証し (RFC 6749 §2.3.1)、認可コードやリフレッシュトークンが
// 属する許可がそのクライアントのものであることを確かめて、許可を返す。
//
// 認証できないクライアントには `invalid_client`、別のクライアントに発行されたコードやトークンには
// `invalid_grant` を返す。
func (uc *CreateOAuthTokenUsecase) authenticateClient(ctx context.Context, params OAuthTokenParams, grantID model.OAuthGrantID, spaceID model.SpaceID) (*model.OAuthGrant, error) {
	app, err := uc.authenticateApplication(ctx, params, spaceID)
	if err != nil {
		return nil, err
	}
	return uc.findGrant(ctx, app, grantID, spaceID)
}

// authenticateApplicationは、クライアントを認証し (RFC 6749 §2.3.1)、アプリを返す。
// アプリは、コードやトークンのスペースで使えるものに限って引く (スペースのアプリか公式クライアント)。
// 認証できないクライアントには `invalid_client` を返す
func (uc *CreateOAuthTokenUsecase) authenticateApplication(ctx context.Context, params OAuthTokenParams, spaceID model.SpaceID) (*model.OAuthApplication, error) {
	app, err := uc.oauthApplicationRepo.FindByClientIDAndSpaceID(ctx, params.ClientID, spaceID)
	if err != nil {
		return nil, fmt.Errorf("OAuthアプリの取得に失敗: %w", err)
	}
	if app == nil || !app.IsAvailableIn(spaceID) || !verifyOAuthClientSecret(app, params.ClientSecret, params.ClientSecretProvided) {
		return nil, oauthTokenError(model.OAuthTokenErrorInvalidClient)
	}
	return app, nil
}

// findGrantは、コードやトークンが属する許可を取得し、認証したクライアントのものであることを確かめる。
// 別のクライアントの許可には `invalid_grant` を返す
func (uc *CreateOAuthTokenUsecase) findGrant(ctx context.Context, app *model.OAuthApplication, grantID model.OAuthGrantID, spaceID model.SpaceID) (*model.OAuthGrant, error) {
	grant, err := uc.oauthGrantRepo.FindByID(ctx, grantID, spaceID)
	if err != nil {
		return nil, fmt.Errorf("OAuthアプリの許可の取得に失敗: %w", err)
	}
	if grant == nil || grant.OAuthApplicationID != app.ID {
		return nil, oauthTokenError(model.OAuthTokenErrorInvalidGrant)
	}
	return grant, nil
}

// checkGrantUsableは、許可の持ち主が許可の属するスペースで今もトークンを使えることと、要求の
// `resource` が許可の属するスペースを指すことを確かめる。
//
// 持ち主の有効性はトークンを使うリクエストのたびにも確かめるが、使えないトークンを発行しない
// よう、発行の時点でも確かめる。
func (uc *CreateOAuthTokenUsecase) checkGrantUsable(ctx context.Context, grant *model.OAuthGrant, resource string) error {
	owner, err := uc.owners.find(ctx, grant.SpaceID, grant.SpaceMemberID)
	if err != nil {
		return err
	}
	if owner == nil || !policy.NewMemberPolicy(owner.SpaceMember.Role.Scopes(), nil).CanCreateOAuthGrant() {
		return oauthTokenError(model.OAuthTokenErrorInvalidGrant)
	}

	if resource != "" {
		// 認可要求のスペースの検索 (citextの列) と揃え、識別子の大文字と小文字は区別しない
		identifier, ok := model.ParseSpaceAPIResourceURL(uc.cfg.AppURL(), resource)
		if !ok || !strings.EqualFold(string(identifier), string(owner.Space.Identifier)) {
			return oauthTokenError(model.OAuthTokenErrorInvalidTarget)
		}
	}
	return nil
}

// revokeReplayedGrantは、使用済みの認可コード・リフレッシュトークン・デバイスコードが再び提示された
// 許可と、許可に属するアクセストークン・リフレッシュトークンをすべて失効させ、invalid_grantを返す。
// 許可は認可ごとに作るため、失効は同じアプリのほかの端末の許可に及ばない。利用者が再び認可すると
// 新しい許可ができるため、失効させた許可を残す使い道は無い
func (uc *CreateOAuthTokenUsecase) revokeReplayedGrant(ctx context.Context, grant *model.OAuthGrant) error {
	if _, err := uc.oauthGrantRepo.Revoke(ctx, grant.ID, grant.SpaceID, grant.SpaceMemberID); err != nil {
		return fmt.Errorf("OAuthアプリの許可の失効に失敗: %w", err)
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

// exchangeCodeは、許可のロック後に許可が失効していないことを確かめ、認可コードを交換する。
// ロックで許可の失効と直列化し、失効が先に終わった場合は、コードの再利用として扱わず拒否する
func (uc *CreateOAuthTokenUsecase) exchangeCode(ctx context.Context, code *model.OAuthAuthorizationCode, input repository.ExchangeOAuthAuthorizationCodeInput) (*model.OAuthAuthorizationCode, error) {
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクション開始に失敗: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	grantRepo := uc.oauthGrantRepo.WithTx(tx)
	grant, err := grantRepo.LockByID(ctx, code.OAuthGrantID, code.SpaceID)
	if err != nil {
		return nil, fmt.Errorf("OAuthアプリの許可のロックに失敗: %w", err)
	}
	if grant == nil || grant.IsRevoked() || !model.HasAllScopes(grant.Scopes, code.Scopes) {
		return nil, oauthTokenError(model.OAuthTokenErrorInvalidGrant)
	}
	exchanged, err := uc.oauthAuthorizationCodeRepo.WithTx(tx).Exchange(ctx, input)
	if err != nil {
		return nil, err
	}
	if exchanged != nil {
		if err := grantRepo.SetLastUsedAt(ctx, grant.ID, grant.SpaceID, input.Now); err != nil {
			return nil, fmt.Errorf("OAuthアプリの許可の最終利用日時の更新に失敗: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗: %w", err)
	}
	return exchanged, nil
}

// rotateTokenは、許可の行を先にロックしてからトークンを更新する。
// ロックで許可の失効と直列化し、失効した許可のトークンを再び発行しない
func (uc *CreateOAuthTokenUsecase) rotateToken(ctx context.Context, token *model.OAuthRefreshToken, input repository.RotateOAuthRefreshTokenInput) (*model.OAuthRefreshToken, error) {
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクション開始に失敗: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	grantRepo := uc.oauthGrantRepo.WithTx(tx)
	grant, err := grantRepo.LockByID(ctx, token.OAuthGrantID, token.SpaceID)
	if err != nil {
		return nil, fmt.Errorf("OAuthアプリの許可のロックに失敗: %w", err)
	}
	if grant == nil || grant.IsRevoked() || !model.HasAllScopes(grant.Scopes, token.Scopes) {
		return nil, oauthTokenError(model.OAuthTokenErrorInvalidGrant)
	}
	rotated, err := uc.oauthRefreshTokenRepo.WithTx(tx).Rotate(ctx, input)
	if err != nil {
		return nil, err
	}
	if rotated != nil {
		if err := grantRepo.SetLastUsedAt(ctx, grant.ID, grant.SpaceID, input.Now); err != nil {
			return nil, fmt.Errorf("OAuthアプリの許可の最終利用日時の更新に失敗: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗: %w", err)
	}
	return rotated, nil
}

// exchangeDeviceは、要求と許可をロックし、最新の状態・スコープ・期限を確かめてデバイスコードを交換する。
// 認可コードの交換 (exchangeCode) と同じく、許可の失効が先に終わった場合は拒否する。
//
// ロックは要求 → 許可の順に取る。同意 (UpdateOAuthDeviceAuthorizationUsecase.approveWithGrant) も
// 要求を先にロックするため、同じ要求への許可と交換は要求のロックで直列化される
func (uc *CreateOAuthTokenUsecase) exchangeDevice(ctx context.Context, device *model.OAuthDeviceAuthorization, input repository.ExchangeOAuthDeviceAuthorizationInput) (*model.OAuthDeviceAuthorization, error) {
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクション開始に失敗: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	deviceRepo := uc.oauthDeviceAuthorizationRepo.WithTx(tx)
	locked, err := deviceRepo.LockByID(ctx, device.ID, device.SpaceID)
	if err != nil {
		return nil, fmt.Errorf("デバイス認可の要求のロックに失敗: %w", err)
	}
	if locked == nil || locked.Status != model.OAuthDeviceAuthorizationStatusApproved {
		// 交換済みの再提示は、期限の前後にかかわらず呼び出し側で失効させる。
		return nil, nil
	}
	grantRepo := uc.oauthGrantRepo.WithTx(tx)
	grant, err := grantRepo.LockByID(ctx, *device.OAuthGrantID, device.SpaceID)
	if err != nil {
		return nil, fmt.Errorf("OAuthアプリの許可のロックに失敗: %w", err)
	}
	if grant == nil || grant.IsRevoked() || !model.HasAllScopes(grant.Scopes, device.Scopes) {
		return nil, oauthTokenError(model.OAuthTokenErrorInvalidGrant)
	}
	// 要求と許可のどちらのロック待機中にも期限は進むため、両方の取得後に確認する。
	if locked.IsExpired(time.Now()) {
		return nil, oauthTokenError(model.OAuthTokenErrorExpiredToken)
	}
	exchanged, err := deviceRepo.Exchange(ctx, input)
	if err != nil {
		return nil, err
	}
	if exchanged != nil {
		if err := grantRepo.SetLastUsedAt(ctx, grant.ID, grant.SpaceID, input.Now); err != nil {
			return nil, fmt.Errorf("OAuthアプリの許可の最終利用日時の更新に失敗: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗: %w", err)
	}
	return exchanged, nil
}
