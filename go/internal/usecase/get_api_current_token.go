package usecase

import (
	"context"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// GetAPICurrentTokenUsecaseは、公開APIでリクエストに使ったトークン自身の情報を取得するユースケース
type GetAPICurrentTokenUsecase struct{}

// NewGetAPICurrentTokenUsecaseはGetAPICurrentTokenUsecaseを生成する
func NewGetAPICurrentTokenUsecase() *GetAPICurrentTokenUsecase {
	return &GetAPICurrentTokenUsecase{}
}

// GetAPICurrentTokenInputはトークンの情報の取得に必要な入力パラメータ
type GetAPICurrentTokenInput struct {
	Principal       *model.APIPrincipal
	SpaceIdentifier model.SpaceIdentifier
}

// GetAPICurrentTokenOutputはトークンの情報の取得結果。
// トークンの値・ダイジェスト・名前は返さない
type GetAPICurrentTokenOutput struct {
	Kind model.APITokenKind
	// Scopesはトークンのスコープを含意展開し、APITokenScopesの順に並べたもの
	Scopes    []model.Scope
	ExpiresAt time.Time
}

// Executeはパスのスペースにトークンが属していれば、トークンの種類・スコープ・有効期限を返す。
// どれもトークン認証の時点で取得してあるため、ここではDBを引かない
func (uc *GetAPICurrentTokenUsecase) Execute(ctx context.Context, input GetAPICurrentTokenInput) (*GetAPICurrentTokenOutput, error) {
	if _, err := resolveAPISpace(ctx, input.Principal, input.SpaceIdentifier); err != nil {
		return nil, err
	}
	return &GetAPICurrentTokenOutput{
		Kind:      input.Principal.TokenKind,
		Scopes:    model.SortAPITokenScopes(input.Principal.Scopes),
		ExpiresAt: input.Principal.TokenExpiresAt,
	}, nil
}
