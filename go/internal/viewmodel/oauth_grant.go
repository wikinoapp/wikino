package viewmodel

import (
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// OAuthGrantは連携中のアプリの画面で表示する許可。
type OAuthGrant struct {
	ID              string
	ApplicationName string

	// Scopesは許可のスコープで、model.APITokenScopesの順に並ぶ
	Scopes    []model.Scope
	CreatedAt time.Time
}

// NewOAuthGrantsは許可のスライスからOAuthGrantのスライスを生成する。アプリの名前はapps
// (アプリのID → アプリ) から引き、アプリが見つからない許可は出さない。
//
// 許可のスコープは保存順が揃っていない (新規作成では要求の順、既存の許可の拡張では文字列の順)
// ため、model.APITokenScopesの順に並べ直す。
func NewOAuthGrants(grants []*model.OAuthGrant, apps map[model.OAuthApplicationID]*model.OAuthApplication) []OAuthGrant {
	vms := make([]OAuthGrant, 0, len(grants))
	for _, grant := range grants {
		app, ok := apps[grant.OAuthApplicationID]
		if !ok {
			continue
		}
		vms = append(vms, OAuthGrant{
			ID:              string(grant.ID),
			ApplicationName: app.Name,
			Scopes:          model.SortAPITokenScopes(grant.Scopes),
			CreatedAt:       grant.CreatedAt,
		})
	}
	return vms
}
