package viewmodel

import (
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// OAuthApplicationはOAuthアプリの画面で表示するアプリ。クライアントシークレットの値は
// 持たない (データベースにはダイジェストしか無い)。
type OAuthApplication struct {
	ID           string
	Name         string
	ClientID     string
	Confidential bool
	RedirectURIs []string
	CreatedAt    time.Time

	// CreatorNameは作成したメンバーの表示名。作成したメンバーがいなくなっていれば空になる
	CreatorName string
}

// NewOAuthApplicationはモデルからOAuthApplicationを生成する。作成したメンバーの表示名は
// creators (スペースメンバーのID → ユーザー) から引く。
func NewOAuthApplication(app *model.OAuthApplication, creators map[model.SpaceMemberID]*model.User) OAuthApplication {
	var creatorName string
	if app.CreatedSpaceMemberID != nil {
		if user, ok := creators[*app.CreatedSpaceMemberID]; ok {
			creatorName = userDisplayName(user)
		}
	}

	return OAuthApplication{
		ID:           string(app.ID),
		Name:         app.Name,
		ClientID:     app.ClientID,
		Confidential: app.IsConfidential(),
		RedirectURIs: app.RedirectURIs,
		CreatedAt:    app.CreatedAt,
		CreatorName:  creatorName,
	}
}

// NewOAuthApplicationsはモデルのスライスからOAuthApplicationのスライスを生成する。
func NewOAuthApplications(apps []*model.OAuthApplication, creators map[model.SpaceMemberID]*model.User) []OAuthApplication {
	vms := make([]OAuthApplication, len(apps))
	for i, app := range apps {
		vms[i] = NewOAuthApplication(app, creators)
	}
	return vms
}
