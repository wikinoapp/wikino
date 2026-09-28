package usecase

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// OAuthAuthorizationParamsはOAuthの認可要求のパラメーター (RFC 6749 §4.1.1、RFC 7636 §4.3、
// RFC 8707 §2)。同意画面の表示と同意の送信は、どちらもこのパラメーターを受け取り、同じ検証を通す。
type OAuthAuthorizationParams struct {
	ClientID            string
	RedirectURI         string
	ResponseType        string
	Scope               string
	State               string
	CodeChallenge       string
	CodeChallengeMethod string
	Resource            string
	// MalformedQueryはGETのクエリに不正なエスケープなどがあることを表す。
	// クライアントとリダイレクトURIを確かめた後に `invalid_request` として返す
	MalformedQuery bool

	// DuplicatedParamsは2回以上送られたパラメーターの名前。認可要求のパラメーターは
	// 1回しか送れない (RFC 6749 §3.1) ため、どの値を採るかを決めずに拒否する
	DuplicatedParams []string
}

// oauthAuthorizationRequestは、検証を通った認可要求。連携先のスペースは、アプリのスペースか、
// 公式クライアントでは `resource` が指すスペースに決まっている
type oauthAuthorizationRequest struct {
	application   *model.OAuthApplication
	space         *model.Space
	redirectURI   string
	scopes        []model.Scope
	state         string
	codeChallenge string
}

// codeChallengePatternはS256のcode_challengeの形式。SHA-256の32バイトをbase64url (パディング
// なし) で表すと43文字になる (RFC 7636 §4.2)
var codeChallengePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)

// resolveOAuthAuthorizationRequestは認可要求を検証し、連携先のスペースを決める。
//
// エラーの返し方はRFC 6749 §4.1.2.1に従う。クライアントIDかリダイレクトURIを確かめられない
// 要求は、攻撃者の指定した先へ利用者を送らないよう、クライアントへ戻さないエラー (RedirectURIが
// 空の*model.OAuthAuthorizationError) にする。それ以外の不正は、リダイレクトURIへ付けて
// クライアントへ戻すエラーにする。
//
// クライアントIDでアプリを引くクエリは `space_id` の規約の例外である。認可要求はスペースを
// 特定する前の入口で、スペースはアプリ (公式クライアントでは `resource`) から決まるため。
func resolveOAuthAuthorizationRequest(
	ctx context.Context,
	appURL string,
	oauthApplicationRepo *repository.OAuthApplicationRepository,
	spaceRepo *repository.SpaceRepository,
	params OAuthAuthorizationParams,
) (*oauthAuthorizationRequest, error) {
	// 1. クライアントとリダイレクトURI (失敗してもクライアントへ戻さない)
	invalidClient := &model.OAuthAuthorizationError{
		Code:    model.OAuthAuthorizationErrorInvalidRequest,
		UserMsg: i18n.T(ctx, "oauth_authorization_error_invalid_client"),
	}
	if params.ClientID == "" || slices.Contains(params.DuplicatedParams, "client_id") {
		return nil, invalidClient
	}
	app, err := oauthApplicationRepo.FindByClientID(ctx, params.ClientID)
	if err != nil {
		return nil, fmt.Errorf("OAuthアプリの取得に失敗: %w", err)
	}
	if app == nil {
		return nil, invalidClient
	}

	// スペースのアプリは、そのスペースが削除されていれば使えない
	var appSpace *model.Space
	if !app.IsOfficialClient() {
		appSpace, err = spaceRepo.FindByID(ctx, *app.SpaceID)
		if err != nil {
			return nil, fmt.Errorf("スペースの取得に失敗: %w", err)
		}
		if appSpace == nil {
			return nil, invalidClient
		}
	}

	// リダイレクトURIは登録が1つでも省略させない。照合の対象を常に要求に載せ、トークン要求で
	// 同じ値を求められるようにするため (OAuth 2.1ドラフトは登録が複数なら必須としている)
	if params.RedirectURI == "" || slices.Contains(params.DuplicatedParams, "redirect_uri") || !app.AcceptsRedirectURI(params.RedirectURI) {
		return nil, &model.OAuthAuthorizationError{
			Code:    model.OAuthAuthorizationErrorInvalidRequest,
			UserMsg: i18n.T(ctx, "oauth_authorization_error_invalid_redirect_uri"),
		}
	}

	// 2. ここからはリダイレクトURIへ付けてクライアントへ戻す
	fail := func(code model.OAuthAuthorizationErrorCode) (*oauthAuthorizationRequest, error) {
		return nil, &model.OAuthAuthorizationError{Code: code, RedirectURI: params.RedirectURI, State: params.State}
	}
	if params.MalformedQuery {
		return fail(model.OAuthAuthorizationErrorInvalidRequest)
	}

	for _, name := range params.DuplicatedParams {
		if name != "resource" {
			return fail(model.OAuthAuthorizationErrorInvalidRequest)
		}
	}

	switch params.ResponseType {
	case model.OAuthResponseTypeCode:
	case "":
		return fail(model.OAuthAuthorizationErrorInvalidRequest)
	default:
		return fail(model.OAuthAuthorizationErrorUnsupportedResponseType)
	}

	// PKCEはS256だけを全クライアントに求める。code_challenge_methodの省略はplainを意味する
	// (RFC 7636 §4.3) ため、省略も拒否する
	if params.CodeChallengeMethod != model.OAuthCodeChallengeMethodS256 || !codeChallengePattern.MatchString(params.CodeChallenge) {
		return fail(model.OAuthAuthorizationErrorInvalidRequest)
	}

	scopes, ok := parseOAuthScopes(params.Scope)
	if !ok {
		return fail(model.OAuthAuthorizationErrorInvalidScope)
	}

	// 3. 連携先のスペース
	if slices.Contains(params.DuplicatedParams, "resource") {
		return fail(model.OAuthAuthorizationErrorInvalidTarget)
	}
	space := appSpace
	switch {
	case params.Resource != "":
		identifier, ok := model.ParseSpaceAPIResourceURL(appURL, params.Resource)
		if !ok {
			return fail(model.OAuthAuthorizationErrorInvalidTarget)
		}
		resourceSpace, err := spaceRepo.FindByIdentifier(ctx, identifier)
		if err != nil {
			return nil, fmt.Errorf("スペースの取得に失敗: %w", err)
		}
		if resourceSpace == nil || !app.IsAvailableIn(resourceSpace.ID) {
			return fail(model.OAuthAuthorizationErrorInvalidTarget)
		}
		space = resourceSpace
	case app.IsOfficialClient():
		// 公式クライアントはスペースに属さないため、`resource` が無ければ連携先を決められない
		return fail(model.OAuthAuthorizationErrorInvalidTarget)
	}

	return &oauthAuthorizationRequest{
		application:   app,
		space:         space,
		redirectURI:   params.RedirectURI,
		scopes:        scopes,
		state:         params.State,
		codeChallenge: params.CodeChallenge,
	}, nil
}

// parseOAuthScopesは空白区切りのスコープ (RFC 6749 §3.3) を解釈する。トークンに付与できる
// スコープ (model.APITokenScopes) だけを受け付け、重複を除いてAPITokenScopesの順に並べる。
// スコープが無い要求は、何を許可するかを利用者に示せないため受け付けない。
func parseOAuthScopes(scope string) ([]model.Scope, bool) {
	requested := strings.Split(scope, " ")
	for _, s := range requested {
		if !slices.Contains(model.APITokenScopes, model.Scope(s)) {
			return nil, false
		}
	}

	scopes := make([]model.Scope, 0, len(model.APITokenScopes))
	for _, s := range model.APITokenScopes {
		if slices.Contains(requested, string(s)) {
			scopes = append(scopes, s)
		}
	}
	return scopes, true
}

// oauthAuthorizationDecisionは、ログイン中のユーザーが連携先のスペースでアプリを許可できるかの判定
type oauthAuthorizationDecision struct {
	spaceMember *model.SpaceMember

	// deniedは許可できない理由。許可できる場合は空
	denied OAuthAuthorizationDeniedReason
}

// OAuthAuthorizationDeniedReasonは、ログイン中のユーザーがアプリを許可できない理由
type OAuthAuthorizationDeniedReason string

const (
	// OAuthAuthorizationDeniedNotMemberは、連携先のスペースの有効なメンバーでないことを表す
	OAuthAuthorizationDeniedNotMember OAuthAuthorizationDeniedReason = "not_member"
	// OAuthAuthorizationDeniedNoPermissionは、メンバーだが `oauth_grant:write` を持たないことを表す
	OAuthAuthorizationDeniedNoPermission OAuthAuthorizationDeniedReason = "no_permission"
)

// decideOAuthAuthorizationは、ユーザーuserIDが連携先のスペースspaceでアプリを許可できるかを判定する
func decideOAuthAuthorization(
	ctx context.Context,
	spaceMemberRepo *repository.SpaceMemberRepository,
	space *model.Space,
	userID model.UserID,
) (*oauthAuthorizationDecision, error) {
	spaceMember, err := spaceMemberRepo.FindActiveBySpaceAndUser(ctx, space.ID, userID)
	if err != nil {
		return nil, fmt.Errorf("スペースメンバーの取得に失敗: %w", err)
	}
	if spaceMember == nil {
		return &oauthAuthorizationDecision{denied: OAuthAuthorizationDeniedNotMember}, nil
	}
	if !newAuthorizer(spaceMember, nil).CanCreateOAuthGrant() {
		return &oauthAuthorizationDecision{spaceMember: spaceMember, denied: OAuthAuthorizationDeniedNoPermission}, nil
	}
	return &oauthAuthorizationDecision{spaceMember: spaceMember}, nil
}

// checkPublicAPIEnabledは、ユーザーuserIDに公開APIのフィーチャーフラグが有効でなければ
// 「見つからない」として答える。公開前の同意画面の存在を知らせないためである
func checkPublicAPIEnabled(ctx context.Context, featureFlagRepo *repository.FeatureFlagRepository, userID model.UserID) error {
	enabled, err := featureFlagRepo.IsEnabled(ctx, userID, model.FeatureFlagPublicAPI)
	if err != nil {
		return fmt.Errorf("フィーチャーフラグの判定に失敗: %w", err)
	}
	if !enabled {
		return &model.AppError{Code: model.AppErrCodeResourceNotFound, UserMsg: i18n.T(ctx, "error_not_found_message")}
	}
	return nil
}
