package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// RevokeOAuthTokenUsecaseは、トークンの失効の要求 (`POST /oauth/revoke`、RFC 7009) に応じて、
// クライアントに発行したアクセストークンかリフレッシュトークンを失効させる。
//
// トークンをダイジェストで引くクエリは `space_id` の規約の例外である (トークン要求と同じく、
// スペースはトークンから決まる)。アプリを `client_id` だけで引くクエリも例外で、
// クライアントの認証時点ではスペースを決める手がかりが要求に無いためである。
type RevokeOAuthTokenUsecase struct {
	oauthApplicationRepo  *repository.OAuthApplicationRepository
	oauthGrantRepo        *repository.OAuthGrantRepository
	oauthAccessTokenRepo  *repository.OAuthAccessTokenRepository
	oauthRefreshTokenRepo *repository.OAuthRefreshTokenRepository
}

// NewRevokeOAuthTokenUsecaseはRevokeOAuthTokenUsecaseを生成する
func NewRevokeOAuthTokenUsecase(
	oauthApplicationRepo *repository.OAuthApplicationRepository,
	oauthGrantRepo *repository.OAuthGrantRepository,
	oauthAccessTokenRepo *repository.OAuthAccessTokenRepository,
	oauthRefreshTokenRepo *repository.OAuthRefreshTokenRepository,
) *RevokeOAuthTokenUsecase {
	return &RevokeOAuthTokenUsecase{
		oauthApplicationRepo:  oauthApplicationRepo,
		oauthGrantRepo:        oauthGrantRepo,
		oauthAccessTokenRepo:  oauthAccessTokenRepo,
		oauthRefreshTokenRepo: oauthRefreshTokenRepo,
	}
}

// OAuthTokenRevocationParamsはトークンの失効の要求のパラメーター (RFC 7009 §2.1)。
//
// `token_type_hint` は受け取らない。トークンの種類は接頭辞で決まり、ヒントは探す順を
// 早めるためのものにすぎない (知らない値のヒントも無視してよい) ためである。
type OAuthTokenRevocationParams struct {
	Token string

	// ClientID・ClientSecret・ClientSecretProvidedは、トークン要求と同じくハンドラーが
	// `Authorization: Basic` か本文から取り出したクライアントの資格情報
	ClientID             string
	ClientSecret         string
	ClientSecretProvided bool

	// DuplicatedParamsは2回以上送られたパラメーターの名前
	DuplicatedParams []string
}

// Executeは、クライアントを認証し、トークンがそのクライアントに発行されたものなら失効させる。
// 要求を受け付けられない場合は*model.OAuthTokenErrorを返す。
//
// 無効なトークン (存在しない・期限切れ・失効済み・使用済み、個人アクセストークンなど
// OAuthのトークンでないもの) の失効は、何もせずに成功として答える (RFC 7009 §2.2)。
// クライアントが失効を確かめる目的は、トークンがもう使えないことにあるためである。
//
// アクセストークンを失効したときは、そのトークンだけを失効させる。リフレッシュトークンを
// 失効したときは、そのトークンと同じ許可のアクセストークンをすべて失効させる。許可そのものと、
// 同じ許可のほかのリフレッシュトークン (同じアプリを別の端末で使っているもの) は残す。
func (uc *RevokeOAuthTokenUsecase) Execute(ctx context.Context, params OAuthTokenRevocationParams) error {
	if len(params.DuplicatedParams) > 0 || params.Token == "" || params.ClientID == "" {
		return oauthTokenError(model.OAuthTokenErrorInvalidRequest)
	}

	// 1. トークンの有無にかかわらず、先にクライアントを認証する
	app, err := uc.oauthApplicationRepo.FindByClientID(ctx, params.ClientID)
	if err != nil {
		return fmt.Errorf("OAuthアプリの取得に失敗: %w", err)
	}
	if app == nil || !verifyOAuthClientSecret(app, params.ClientSecret, params.ClientSecretProvided) {
		return oauthTokenError(model.OAuthTokenErrorInvalidClient)
	}

	// 2. トークンがそのクライアントに発行されたものか確かめる
	target, err := uc.findToken(ctx, params.Token)
	if err != nil {
		return err
	}
	if target == nil {
		return nil
	}
	if !app.IsAvailableIn(target.spaceID) {
		return oauthTokenError(model.OAuthTokenErrorUnauthorizedClient)
	}

	grant, err := uc.oauthGrantRepo.FindByID(ctx, target.grantID, target.spaceID)
	if err != nil {
		return fmt.Errorf("OAuthアプリの許可の取得に失敗: %w", err)
	}
	if grant == nil || grant.OAuthApplicationID != app.ID {
		return oauthTokenError(model.OAuthTokenErrorUnauthorizedClient)
	}

	// 3. 永続化
	now := time.Now()
	if target.accessTokenID != "" {
		if err := uc.oauthAccessTokenRepo.Revoke(ctx, target.accessTokenID, target.spaceID, now); err != nil {
			return fmt.Errorf("アクセストークンの失効に失敗: %w", err)
		}
		return nil
	}
	if err := uc.oauthRefreshTokenRepo.Revoke(ctx, target.refreshTokenID, target.spaceID, now); err != nil {
		return fmt.Errorf("リフレッシュトークンの失効に失敗: %w", err)
	}
	return nil
}

// oauthRevocationTargetは失効を求められたトークン。accessTokenIDとrefreshTokenIDのどちらか一方を持つ
type oauthRevocationTarget struct {
	accessTokenID  model.OAuthAccessTokenID
	refreshTokenID model.OAuthRefreshTokenID
	grantID        model.OAuthGrantID
	spaceID        model.SpaceID
}

// findTokenは接頭辞で種類を見分けてトークンを引く。OAuthのトークンでないものや、見つからない
// トークンにはnilを返す。期限切れ・失効済み・使用済みのトークンも返し、失効のクエリがそれらを
// 変えないことに任せる
func (uc *RevokeOAuthTokenUsecase) findToken(ctx context.Context, token string) (*oauthRevocationTarget, error) {
	digest := auth.DigestOpaqueToken(token)

	switch {
	case strings.HasPrefix(token, string(auth.OAuthAccessTokenPrefix)):
		accessToken, err := uc.oauthAccessTokenRepo.FindByTokenDigest(ctx, digest)
		if err != nil {
			return nil, fmt.Errorf("アクセストークンの取得に失敗: %w", err)
		}
		if accessToken == nil {
			return nil, nil
		}
		return &oauthRevocationTarget{accessTokenID: accessToken.ID, grantID: accessToken.OAuthGrantID, spaceID: accessToken.SpaceID}, nil
	case strings.HasPrefix(token, string(auth.OAuthRefreshTokenPrefix)):
		refreshToken, err := uc.oauthRefreshTokenRepo.FindByTokenDigest(ctx, digest)
		if err != nil {
			return nil, fmt.Errorf("リフレッシュトークンの取得に失敗: %w", err)
		}
		if refreshToken == nil {
			return nil, nil
		}
		return &oauthRevocationTarget{refreshTokenID: refreshToken.ID, grantID: refreshToken.OAuthGrantID, spaceID: refreshToken.SpaceID}, nil
	default:
		return nil, nil
	}
}
