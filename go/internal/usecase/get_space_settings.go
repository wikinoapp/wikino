package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetSpaceSettingsUsecaseはスペース設定のトップが表示するものを集める。
type GetSpaceSettingsUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	featureFlagRepo *repository.FeatureFlagRepository
}

// NewGetSpaceSettingsUsecaseはGetSpaceSettingsUsecaseを生成する。
func NewGetSpaceSettingsUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	featureFlagRepo *repository.FeatureFlagRepository,
) *GetSpaceSettingsUsecase {
	return &GetSpaceSettingsUsecase{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		featureFlagRepo: featureFlagRepo,
	}
}

// GetSpaceSettingsInputは画面を描画するための入力パラメータ。
type GetSpaceSettingsInput struct {
	SpaceIdentifier model.SpaceIdentifier
	UserID          model.UserID
}

// GetSpaceSettingsOutputはスペースと、閲覧者が開けるスペース設定の項目を返す。
type GetSpaceSettingsOutput struct {
	Space *model.Space
	Items SpaceSettingsItems
}

// Executeはスペースと、閲覧者が開けるスペース設定の項目を解決する。
//
// スペースのアクティブなメンバーでない場合と、開ける項目が1つも無い場合は、どちらも
// 「見つからない」として答える。あるスペースに誰が参加していて誰が何を変更できるかを、
// 外部から試して確かめられないようにするためである。
func (uc *GetSpaceSettingsUsecase) Execute(ctx context.Context, input GetSpaceSettingsInput) (*GetSpaceSettingsOutput, error) {
	space, err := uc.spaceRepo.FindByIdentifier(ctx, input.SpaceIdentifier)
	if err != nil {
		return nil, fmt.Errorf("スペースの取得に失敗: %w", err)
	}
	if space == nil {
		return nil, &model.AppError{Code: model.AppErrCodeResourceNotFound, UserMsg: i18n.T(ctx, "error_not_found_message")}
	}

	spaceMember, err := uc.spaceMemberRepo.FindActiveBySpaceAndUser(ctx, space.ID, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("スペースメンバーの取得に失敗: %w", err)
	}
	if spaceMember == nil {
		return nil, &model.AppError{Code: model.AppErrCodeResourceNotFound, UserMsg: i18n.T(ctx, "error_not_found_message")}
	}

	items, err := resolveSpaceSettingsItems(ctx, uc.featureFlagRepo, spaceMember)
	if err != nil {
		return nil, err
	}
	if !items.CanShow() {
		return nil, &model.AppError{Code: model.AppErrCodeForbidden, UserMsg: i18n.T(ctx, "error_forbidden")}
	}

	return &GetSpaceSettingsOutput{Space: space, Items: items}, nil
}
