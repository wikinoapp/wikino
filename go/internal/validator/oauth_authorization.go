package validator

import (
	"context"
	"slices"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
)

// OAuthAuthorizationCreateValidatorは、同意画面で利用者が選んだ付与するスコープのバリデーションを行う
type OAuthAuthorizationCreateValidator struct{}

// NewOAuthAuthorizationCreateValidatorはOAuthAuthorizationCreateValidatorを生成する
func NewOAuthAuthorizationCreateValidator() *OAuthAuthorizationCreateValidator {
	return &OAuthAuthorizationCreateValidator{}
}

// OAuthAuthorizationCreateValidatorInputはバリデーションの入力パラメータ。
// RequestedScopesは検証を通った認可要求のスコープ、GrantedScopesは同意画面が送信した値
type OAuthAuthorizationCreateValidatorInput struct {
	RequestedScopes []model.Scope
	GrantedScopes   []string
}

// Validateは、送信されたスコープが要求の部分集合で、1つ以上あることを確かめ、付与するスコープを
// RequestedScopesの順に重複を除いて返す。要求に無いスコープが1つでも含まれていれば送信全体を
// 拒否する。同意画面には要求されたスコープしか出さないため、それ以外が届くのは改ざんされた
// 送信だけであり、黙って取り除くと利用者の意図と違う許可を作ることになる。
func (v *OAuthAuthorizationCreateValidator) Validate(ctx context.Context, input OAuthAuthorizationCreateValidatorInput) ([]model.Scope, error) {
	ve := model.NewValidationError()

	if len(input.GrantedScopes) == 0 {
		ve.AddField("granted_scopes", i18n.T(ctx, "validation_oauth_authorization_scopes_required"))
		return nil, ve
	}

	for _, value := range input.GrantedScopes {
		if !model.HasScope(input.RequestedScopes, model.Scope(value)) {
			ve.AddField("granted_scopes", i18n.T(ctx, "validation_oauth_authorization_scopes_invalid"))
			return nil, ve
		}
	}

	scopes := make([]model.Scope, 0, len(input.RequestedScopes))
	for _, scope := range input.RequestedScopes {
		if slices.Contains(input.GrantedScopes, string(scope)) {
			scopes = append(scopes, scope)
		}
	}
	return scopes, nil
}
