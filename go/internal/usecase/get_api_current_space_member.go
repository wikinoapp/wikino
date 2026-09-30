package usecase

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// GetAPICurrentSpaceMemberUsecaseは、公開APIでトークンの持ち主の、束縛先のスペースでのメンバーを取得するユースケース
type GetAPICurrentSpaceMemberUsecase struct{}

// NewGetAPICurrentSpaceMemberUsecaseはGetAPICurrentSpaceMemberUsecaseを生成する
func NewGetAPICurrentSpaceMemberUsecase() *GetAPICurrentSpaceMemberUsecase {
	return &GetAPICurrentSpaceMemberUsecase{}
}

// GetAPICurrentSpaceMemberInputはメンバーの取得に必要な入力パラメータ
type GetAPICurrentSpaceMemberInput struct {
	Principal       *model.APIPrincipal
	SpaceIdentifier model.SpaceIdentifier
}

// GetAPICurrentSpaceMemberOutputはメンバーの取得結果。
// メンバーはアットネームと表示名を持たないため、トークンの持ち主のユーザーもあわせて返す
type GetAPICurrentSpaceMemberOutput struct {
	SpaceMember *model.SpaceMember
	User        *model.User
}

// Executeはパスのスペースがトークンの束縛先であれば、トークンの持ち主のメンバーを返す。
// メンバーとユーザーはトークン認証の時点で有効なことを確かめてあるため、ここではDBを引かない
func (uc *GetAPICurrentSpaceMemberUsecase) Execute(ctx context.Context, input GetAPICurrentSpaceMemberInput) (*GetAPICurrentSpaceMemberOutput, error) {
	if _, err := resolveAPISpace(ctx, input.Principal, input.SpaceIdentifier); err != nil {
		return nil, err
	}
	return &GetAPICurrentSpaceMemberOutput{
		SpaceMember: input.Principal.SpaceMember,
		User:        input.Principal.User,
	}, nil
}
