package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// publicAPISettingsAccessは、公開APIのためのスペース設定の画面 (個人アクセストークン・
// OAuthアプリ) が扱うスペースと、それを操作するメンバーとその権限を表す。
type publicAPISettingsAccess struct {
	space       *model.Space
	spaceMember *model.SpaceMember
	authorizer  policy.Authorizer
}

// fetchPublicAPISettingsAccessは、公開APIのためのスペース設定の画面 (個人アクセストークン・
// OAuthアプリ) が扱うスペースと、それを操作するユーザーのメンバーシップを解決し、allowedが
// 許さないものとフィーチャーフラグが無効なものを拒否する。各画面の表示と、発行・登録などの
// 操作そのものがここを通るため、誰を通すかについて別々の答えを出すことはない。
//
// 通さない場合はどれも「見つからない」として答える。あるスペースに誰が参加していて誰が
// 何をできるかを、外部から試して確かめられないようにするためである。フラグが無効な場合も、
// 公開前の画面の存在を知らせないよう同じく答える。
func fetchPublicAPISettingsAccess(
	ctx context.Context,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	featureFlagRepo *repository.FeatureFlagRepository,
	spaceIdentifier model.SpaceIdentifier,
	userID model.UserID,
	allowed func(policy.Authorizer) bool,
) (*publicAPISettingsAccess, error) {
	space, err := spaceRepo.FindByIdentifier(ctx, spaceIdentifier)
	if err != nil {
		return nil, fmt.Errorf("スペースの取得に失敗: %w", err)
	}
	if space == nil {
		return nil, &model.AppError{Code: model.AppErrCodeResourceNotFound, UserMsg: i18n.T(ctx, "error_not_found_message")}
	}

	spaceMember, err := spaceMemberRepo.FindActiveBySpaceAndUser(ctx, space.ID, userID)
	if err != nil {
		return nil, fmt.Errorf("スペースメンバーの取得に失敗: %w", err)
	}
	if spaceMember == nil {
		return nil, &model.AppError{Code: model.AppErrCodeResourceNotFound, UserMsg: i18n.T(ctx, "error_not_found_message")}
	}

	authorizer := newAuthorizer(spaceMember, nil)
	if !allowed(authorizer) {
		return nil, &model.AppError{Code: model.AppErrCodeForbidden, UserMsg: i18n.T(ctx, "error_forbidden")}
	}

	enabled, err := featureFlagRepo.IsEnabled(ctx, userID, model.FeatureFlagPublicAPI)
	if err != nil {
		return nil, fmt.Errorf("フィーチャーフラグの判定に失敗: %w", err)
	}
	if !enabled {
		return nil, &model.AppError{Code: model.AppErrCodeResourceNotFound, UserMsg: i18n.T(ctx, "error_not_found_message")}
	}

	return &publicAPISettingsAccess{space: space, spaceMember: spaceMember, authorizer: authorizer}, nil
}
