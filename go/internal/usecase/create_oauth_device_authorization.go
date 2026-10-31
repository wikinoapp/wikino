package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// oauthUserCodeUniqueIndexはユーザーコードのダイジェストの一意制約の名前
const oauthUserCodeUniqueIndex = "idx_oauth_device_authorizations_user_code_digest"

// oauthUserCodeMaxAttemptsは、ユーザーコードが既存の要求と衝突したときに作り直す回数の上限。
// 8文字のコードが続けて衝突することはまず無く、上限は無限に繰り返さないためのものである
const oauthUserCodeMaxAttempts = 3

// CreateOAuthDeviceAuthorizationUsecaseは、デバイス認可の要求 (`POST /oauth/device_authorization`、
// RFC 8628 §3.1) に応じて、デバイスコードとユーザーコードを発行する。
//
// デバイス認可を使えるのはpublicクライアントだけである。confidentialクライアントはリダイレクトを
// 受けるサーバーを持ち、認可コードのフローを使えるため。
// クライアントIDでアプリを引くクエリは `space_id` の規約の例外である (findOAuthClientを参照)。
type CreateOAuthDeviceAuthorizationUsecase struct {
	cfg                          *config.Config
	oauthApplicationRepo         *repository.OAuthApplicationRepository
	spaceRepo                    *repository.SpaceRepository
	oauthDeviceAuthorizationRepo *repository.OAuthDeviceAuthorizationRepository

	// generateUserCodeはユーザーコードを生成する。テストで衝突を再現するために差し替えられるようにしている
	generateUserCode func() (string, error)
}

// NewCreateOAuthDeviceAuthorizationUsecaseはCreateOAuthDeviceAuthorizationUsecaseを生成する
func NewCreateOAuthDeviceAuthorizationUsecase(
	cfg *config.Config,
	oauthApplicationRepo *repository.OAuthApplicationRepository,
	spaceRepo *repository.SpaceRepository,
	oauthDeviceAuthorizationRepo *repository.OAuthDeviceAuthorizationRepository,
) *CreateOAuthDeviceAuthorizationUsecase {
	return &CreateOAuthDeviceAuthorizationUsecase{
		cfg:                          cfg,
		oauthApplicationRepo:         oauthApplicationRepo,
		spaceRepo:                    spaceRepo,
		oauthDeviceAuthorizationRepo: oauthDeviceAuthorizationRepo,
		generateUserCode:             auth.GenerateOAuthUserCode,
	}
}

// OAuthDeviceAuthorizationParamsはデバイス認可の要求のパラメーター (RFC 8628 §3.1、RFC 8707 §2)
type OAuthDeviceAuthorizationParams struct {
	ClientID string
	Scope    string
	Resource string

	// DeviceNameは独自のパラメーター `device_name` の値で、利用者が許可したときに許可へ写す。
	// 同意画面には出さない (他人の端末から送られた値で利用者を誤認させうるため。RFC 8628 §5.4)
	DeviceName string

	// ClientAuthenticatedは、クライアントが `Authorization` ヘッダーか `client_secret` で認証を
	// 試みたことを表す。publicクライアントは認証しないため、試みた要求は拒否する
	ClientAuthenticated bool

	// DuplicatedParamsは2回以上送られたパラメーターの名前。パラメーターは1回しか送れない
	// (RFC 6749 §3.2) ため、どの値を採るかを決めずに拒否する
	DuplicatedParams []string
}

// CreateOAuthDeviceAuthorizationOutputはデバイス認可の応答 (RFC 8628 §3.2) の値。
// デバイスコードとユーザーコードの値そのものはここでしか得られず、データベースには
// ダイジェストしか残らない
type CreateOAuthDeviceAuthorizationOutput struct {
	DeviceCode string
	// UserCodeは利用者に示す形 (`BCDF-GHJK`) のユーザーコード
	UserCode                string
	VerificationURI         string
	VerificationURIComplete string
	ExpiresIn               time.Duration
	Interval                time.Duration
}

// Executeはデバイス認可の要求を検証し、許可待ちの要求を作る。要求を受け付けられない場合は
// *model.OAuthTokenErrorを返す (エラーの形はトークンエンドポイントと同じ。RFC 8628 §3.2)。
func (uc *CreateOAuthDeviceAuthorizationUsecase) Execute(ctx context.Context, params OAuthDeviceAuthorizationParams) (*CreateOAuthDeviceAuthorizationOutput, error) {
	// 1. 要求の検証
	if len(params.DuplicatedParams) > 0 || params.ClientID == "" {
		return nil, oauthTokenError(model.OAuthTokenErrorInvalidRequest)
	}

	app, appSpace, err := findOAuthClient(ctx, uc.oauthApplicationRepo, uc.spaceRepo, params.ClientID)
	if err != nil {
		return nil, err
	}
	// CIMDのクライアントは、利用者が認可要求で同意画面を開いたときに文書を取得して確かめる。
	// デバイス認可の要求では文書を確かめられないため受け付けない
	if app == nil || app.IsClientIDMetadataDocument() {
		return nil, oauthTokenError(model.OAuthTokenErrorInvalidClient)
	}
	if app.IsConfidential() {
		return nil, oauthTokenError(model.OAuthTokenErrorUnauthorizedClient)
	}
	// publicクライアントがシークレットを送った場合は、トークンエンドポイントと同じくクライアントの
	// 設定の誤りとして認証に失敗させる
	if params.ClientAuthenticated {
		return nil, oauthTokenError(model.OAuthTokenErrorInvalidClient)
	}

	scopes, ok := parseOAuthScopes(params.Scope)
	if !ok {
		return nil, oauthTokenError(model.OAuthTokenErrorInvalidScope)
	}

	deviceName, ok := validator.ParseOAuthDeviceName(params.DeviceName)
	if !ok {
		return nil, oauthTokenError(model.OAuthTokenErrorInvalidRequest)
	}

	space, err := resolveOAuthTargetSpace(ctx, uc.cfg.AppURL(), uc.spaceRepo, app, appSpace, params.Resource)
	if err != nil {
		return nil, err
	}
	if space == nil {
		return nil, oauthTokenError(model.OAuthTokenErrorInvalidTarget)
	}

	// 2. 永続化
	return uc.createDeviceAuthorization(ctx, app, space, scopes, deviceName)
}

// createDeviceAuthorizationは、デバイスコードとユーザーコードを生成し、許可待ちの要求を作る。
// ユーザーコードが既存の要求と衝突したら、コードを作り直して再び試みる
func (uc *CreateOAuthDeviceAuthorizationUsecase) createDeviceAuthorization(
	ctx context.Context,
	app *model.OAuthApplication,
	space *model.Space,
	scopes []model.Scope,
	deviceName *string,
) (*CreateOAuthDeviceAuthorizationOutput, error) {
	deviceCode, err := auth.GenerateOpaqueToken(auth.OAuthDeviceCodePrefix)
	if err != nil {
		return nil, fmt.Errorf("デバイスコードの生成に失敗: %w", err)
	}

	for range oauthUserCodeMaxAttempts {
		userCode, err := uc.generateUserCode()
		if err != nil {
			return nil, fmt.Errorf("ユーザーコードの生成に失敗: %w", err)
		}

		_, err = uc.oauthDeviceAuthorizationRepo.Create(ctx, repository.CreateOAuthDeviceAuthorizationInput{
			OAuthApplicationID: app.ID,
			SpaceID:            space.ID,
			DeviceCodeDigest:   auth.DigestOpaqueToken(deviceCode),
			UserCodeDigest:     auth.DigestOpaqueToken(userCode),
			Scopes:             scopes,
			Interval:           model.OAuthDeviceAuthorizationInterval,
			ExpiresAt:          time.Now().Add(model.OAuthDeviceAuthorizationLifetime),
			DeviceName:         deviceName,
		})
		if isUniqueViolationOn(err, oauthUserCodeUniqueIndex) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("デバイス認可の要求の作成に失敗: %w", err)
		}

		formatted := auth.FormatOAuthUserCode(userCode)
		verificationURI := uc.cfg.AppURL() + model.OAuthDeviceVerificationPath
		return &CreateOAuthDeviceAuthorizationOutput{
			DeviceCode:              deviceCode,
			UserCode:                formatted,
			VerificationURI:         verificationURI,
			VerificationURIComplete: verificationURI + "?user_code=" + formatted,
			ExpiresIn:               model.OAuthDeviceAuthorizationLifetime,
			Interval:                model.OAuthDeviceAuthorizationInterval,
		}, nil
	}
	return nil, fmt.Errorf("ユーザーコードが%d回続けて既存の要求と衝突した", oauthUserCodeMaxAttempts)
}
