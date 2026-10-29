package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// AuthenticateAPITokenUsecaseは公開API・MCPサーバーのトークンを照合し、呼び出し主体を返すユースケース。
// トークン自体の有効性 (期限・失効・宛先) に加え、持ち主がトークンを使い続けてよいか
// (退会・フィーチャーフラグ・メンバーシップ・メンバー権限) をリクエストのたびに確かめる
type AuthenticateAPITokenUsecase struct {
	personalAccessTokenRepo *repository.PersonalAccessTokenRepository
	oauthAccessTokenRepo    *repository.OAuthAccessTokenRepository
	oauthGrantRepo          *repository.OAuthGrantRepository
	owners                  *apiTokenOwnerFinder
}

// NewAuthenticateAPITokenUsecaseはAuthenticateAPITokenUsecaseを生成する
func NewAuthenticateAPITokenUsecase(
	personalAccessTokenRepo *repository.PersonalAccessTokenRepository,
	oauthAccessTokenRepo *repository.OAuthAccessTokenRepository,
	oauthGrantRepo *repository.OAuthGrantRepository,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	userRepo *repository.UserRepository,
	featureFlagRepo *repository.FeatureFlagRepository,
) *AuthenticateAPITokenUsecase {
	return &AuthenticateAPITokenUsecase{
		personalAccessTokenRepo: personalAccessTokenRepo,
		oauthAccessTokenRepo:    oauthAccessTokenRepo,
		oauthGrantRepo:          oauthGrantRepo,
		owners:                  newAPITokenOwnerFinder(spaceRepo, spaceMemberRepo, userRepo, featureFlagRepo),
	}
}

// Executeは、audienceを宛先とするリクエストのトークンを照合して呼び出し主体を返す。トークンを
// 受け付けられない場合は (nil, nil) を返す。トークンの種類は接頭辞で見分ける。
//
// 公開APIは、公開API宛てのOAuthのアクセストークンと個人用アクセストークンを受け付ける。
// MCPサーバーは、MCPサーバー宛てのOAuthのアクセストークンだけを受け付ける (個人用アクセストークンは
// 宛先を持たず、公開APIのためのものであるため)
func (uc *AuthenticateAPITokenUsecase) Execute(ctx context.Context, token string, audience model.OAuthAudience) (*model.APIPrincipal, error) {
	switch {
	case strings.HasPrefix(token, string(auth.PersonalAccessTokenPrefix)):
		if audience != model.OAuthAudiencePublicAPI {
			return nil, nil
		}
		return uc.authenticatePersonalAccessToken(ctx, token)
	case strings.HasPrefix(token, string(auth.OAuthAccessTokenPrefix)):
		return uc.authenticateOAuthAccessToken(ctx, token, audience)
	default:
		return nil, nil
	}
}

// authenticatePersonalAccessTokenは個人用アクセストークンを照合する
func (uc *AuthenticateAPITokenUsecase) authenticatePersonalAccessToken(ctx context.Context, token string) (*model.APIPrincipal, error) {
	now := time.Now()

	pat, err := uc.personalAccessTokenRepo.FindByTokenDigest(ctx, auth.DigestOpaqueToken(token))
	if err != nil {
		return nil, fmt.Errorf("個人用アクセストークンの取得に失敗: %w", err)
	}
	if pat == nil || !pat.IsActive(now) {
		return nil, nil
	}

	principal, err := uc.owners.find(ctx, pat.SpaceID, pat.SpaceMemberID, model.OAuthAudiencePublicAPI)
	if err != nil {
		return nil, err
	}
	if principal == nil || !policy.NewMemberPolicy(principal.SpaceMember.Role.Scopes(), nil).CanCreatePersonalAccessToken() {
		return nil, nil
	}

	// 最終使用日時は利用状況の表示にだけ使うため、書き込みに失敗してもリクエストは止めない
	if err := uc.personalAccessTokenRepo.UpdateLastUsedAt(ctx, pat.ID, pat.SpaceID, now); err != nil {
		slog.ErrorContext(ctx, "個人用アクセストークンの最終使用日時の更新に失敗しました", "error", err, "personal_access_token_id", pat.ID)
	}

	principal.TokenKind = model.APITokenKindPersonalAccessToken
	principal.TokenExpiresAt = pat.ExpiresAt
	principal.Scopes = policy.ExpandAPITokenScopes(pat.Scopes)
	return principal, nil
}

// authenticateOAuthAccessTokenはOAuthのアクセストークンを照合する。宛先がaudienceと違うか、
// 許可が失効していれば (連携の解除・アプリの削除) 受け付けない
func (uc *AuthenticateAPITokenUsecase) authenticateOAuthAccessToken(ctx context.Context, token string, audience model.OAuthAudience) (*model.APIPrincipal, error) {
	now := time.Now()

	accessToken, err := uc.oauthAccessTokenRepo.FindByTokenDigest(ctx, auth.DigestOpaqueToken(token))
	if err != nil {
		return nil, fmt.Errorf("OAuthのアクセストークンの取得に失敗: %w", err)
	}
	if accessToken == nil || !accessToken.IsActive(now) || accessToken.Audience != audience {
		return nil, nil
	}

	grant, err := uc.oauthGrantRepo.FindByID(ctx, accessToken.OAuthGrantID, accessToken.SpaceID)
	if err != nil {
		return nil, fmt.Errorf("OAuthアプリの許可の取得に失敗: %w", err)
	}
	if grant == nil || grant.IsRevoked() {
		return nil, nil
	}

	principal, err := uc.owners.find(ctx, grant.SpaceID, grant.SpaceMemberID, audience)
	if err != nil {
		return nil, err
	}
	if principal == nil || !policy.NewMemberPolicy(principal.SpaceMember.Role.Scopes(), nil).CanCreateOAuthGrant() {
		return nil, nil
	}

	// 最終利用日時は連携の一覧で端末を見分けるためにだけ使うため、書き込みに失敗してもリクエストは止めない
	if err := uc.oauthGrantRepo.UpdateLastUsedAt(ctx, grant.ID, grant.SpaceID, now); err != nil {
		slog.ErrorContext(ctx, "OAuthアプリの許可の最終利用日時の更新に失敗しました", "error", err, "oauth_grant_id", grant.ID)
	}

	principal.TokenKind = model.APITokenKindOAuthAccessToken
	principal.OAuthApplicationID = &grant.OAuthApplicationID
	principal.TokenExpiresAt = accessToken.ExpiresAt
	principal.Scopes = policy.ExpandAPITokenScopes(accessToken.Scopes)
	return principal, nil
}

// apiTokenOwnerFinderは、公開APIのトークンが属するスペースと持ち主を取得する。トークンの照合と、
// OAuthのトークン要求で持ち主がトークンを使い続けてよいかを確かめるときに使う
type apiTokenOwnerFinder struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	userRepo        *repository.UserRepository
	featureFlagRepo *repository.FeatureFlagRepository
}

func newAPITokenOwnerFinder(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	userRepo *repository.UserRepository,
	featureFlagRepo *repository.FeatureFlagRepository,
) *apiTokenOwnerFinder {
	return &apiTokenOwnerFinder{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		userRepo:        userRepo,
		featureFlagRepo: featureFlagRepo,
	}
}

// findはトークンが属するスペースと持ち主を、トークンの種類とスコープを除いた呼び出し主体として
// 返す。スペースが削除済み、持ち主がスペースの有効なメンバーでない、ユーザーが退会済み、
// トークンの宛先のフィーチャーフラグが無効のいずれかであれば (nil, nil) を返す。トークンを
// 使い続けるためのメンバー権限 (トークンの種類ごとに異なる) は呼び出し側が確かめる
func (f *apiTokenOwnerFinder) find(ctx context.Context, spaceID model.SpaceID, spaceMemberID model.SpaceMemberID, audience model.OAuthAudience) (*model.APIPrincipal, error) {
	space, err := f.spaceRepo.FindByID(ctx, spaceID)
	if err != nil {
		return nil, fmt.Errorf("スペースの取得に失敗: %w", err)
	}
	if space == nil {
		return nil, nil
	}

	members, err := f.spaceMemberRepo.FindByIDs(ctx, []model.SpaceMemberID{spaceMemberID}, spaceID)
	if err != nil {
		return nil, fmt.Errorf("スペースメンバーの取得に失敗: %w", err)
	}
	if len(members) == 0 || !members[0].Active {
		return nil, nil
	}
	spaceMember := members[0]

	user, err := f.userRepo.FindByID(ctx, spaceMember.UserID)
	if err != nil {
		return nil, fmt.Errorf("ユーザーの取得に失敗: %w", err)
	}
	if user == nil || user.DiscardedAt != nil {
		return nil, nil
	}

	enabled, err := f.featureFlagRepo.IsEnabled(ctx, user.ID, audienceFeatureFlag(audience))
	if err != nil {
		return nil, fmt.Errorf("フィーチャーフラグの判定に失敗: %w", err)
	}
	if !enabled {
		return nil, nil
	}

	return &model.APIPrincipal{
		User:        user,
		Space:       space,
		SpaceMember: spaceMember,
	}, nil
}

// audienceFeatureFlagは、トークンの宛先を使うのに要るフィーチャーフラグを返す。MCPサーバーを
// 公開APIと切り離して公開できるよう、MCPサーバー宛てには公開APIのフラグの代わりに専用のフラグを求める
func audienceFeatureFlag(audience model.OAuthAudience) model.FeatureFlagName {
	if audience == model.OAuthAudienceMCPServer {
		return model.FeatureFlagRemoteMCPServer
	}
	return model.FeatureFlagPublicAPI
}
