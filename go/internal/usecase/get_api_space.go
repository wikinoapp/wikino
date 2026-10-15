package usecase

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
)

// GetAPISpaceUsecaseは、公開APIでトークンが属するスペースを取得するユースケース
type GetAPISpaceUsecase struct{}

// NewGetAPISpaceUsecaseはGetAPISpaceUsecaseを生成する
func NewGetAPISpaceUsecase() *GetAPISpaceUsecase {
	return &GetAPISpaceUsecase{}
}

// GetAPISpaceInputはスペースの取得に必要な入力パラメータ
type GetAPISpaceInput struct {
	Principal       *model.APIPrincipal
	SpaceIdentifier model.SpaceIdentifier
}

// GetAPISpaceOutputはスペースの取得結果
type GetAPISpaceOutput struct {
	Space *model.Space
}

// Executeはパスのスペースにトークンが属していればそのスペースを返す
func (uc *GetAPISpaceUsecase) Execute(ctx context.Context, input GetAPISpaceInput) (*GetAPISpaceOutput, error) {
	space, err := resolveAPISpace(ctx, input.Principal, input.SpaceIdentifier)
	if err != nil {
		return nil, err
	}
	return &GetAPISpaceOutput{Space: space}, nil
}

// resolveAPISpaceは公開APIのパスで指定されたスペースを、トークンが属するスペースとして解決する。
// トークンが属さないスペースは、存在を秘匿するため未存在として扱う。
// トークンが属するスペースはトークン認証の時点で削除済みでないことを確かめてあるため、ここではDBを引かない
func resolveAPISpace(ctx context.Context, principal *model.APIPrincipal, identifier model.SpaceIdentifier) (*model.Space, error) {
	if !principal.IsBoundTo(identifier) {
		return nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}
	return principal.Space, nil
}
