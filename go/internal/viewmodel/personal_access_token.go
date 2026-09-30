package viewmodel

import (
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// PersonalAccessTokenは個人アクセストークンの画面で表示するトークン。トークンの値そのものは
// 持たず、見分けるための末尾の数文字だけを持つ。
type PersonalAccessToken struct {
	ID             string
	Name           string
	TokenLastChars string
	Scopes         []model.Scope
	ExpiresAt      time.Time
	LastUsedAt     *time.Time
	CreatedAt      time.Time

	// Expiredはnowの時点で有効期限が切れているかを表す
	Expired bool
}

// NewPersonalAccessTokenはモデルからPersonalAccessTokenを生成する。期限切れかどうかは
// nowの時点で読む。
func NewPersonalAccessToken(token *model.PersonalAccessToken, now time.Time) PersonalAccessToken {
	return PersonalAccessToken{
		ID:             string(token.ID),
		Name:           token.Name,
		TokenLastChars: token.TokenLastChars,
		Scopes:         token.Scopes,
		ExpiresAt:      token.ExpiresAt,
		LastUsedAt:     token.LastUsedAt,
		CreatedAt:      token.CreatedAt,
		Expired:        token.IsExpired(now),
	}
}

// NewPersonalAccessTokensはモデルのスライスからPersonalAccessTokenのスライスを生成する。
func NewPersonalAccessTokens(tokens []*model.PersonalAccessToken, now time.Time) []PersonalAccessToken {
	vms := make([]PersonalAccessToken, len(tokens))
	for i, token := range tokens {
		vms[i] = NewPersonalAccessToken(token, now)
	}
	return vms
}
