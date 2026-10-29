package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// apiSettingsAccessは、公開APIとMCPのためのスペース設定の画面が扱うスペースと、
// それを操作するメンバーとその権限を表す。
type apiSettingsAccess struct {
	space       *model.Space
	spaceMember *model.SpaceMember
	authorizer  policy.Authorizer
}

// fetchPublicAPISettingsAccessは、個人用アクセストークンとOAuthアプリの管理へのアクセスを判定する。
// 公開APIのフィーチャーフラグを求め、MCPサーバーだけを使う利用者には管理を許さない。
func fetchPublicAPISettingsAccess(
	ctx context.Context,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	featureFlagRepo *repository.FeatureFlagRepository,
	spaceIdentifier model.SpaceIdentifier,
	userID model.UserID,
	allowed func(policy.Authorizer) bool,
) (*apiSettingsAccess, error) {
	return fetchAPISettingsAccess(ctx, spaceRepo, spaceMemberRepo, featureFlagRepo, spaceIdentifier, userID, allowed, model.FeatureFlagPublicAPI)
}

// fetchOAuthGrantSettingsAccessは、利用者自身の連携の一覧と解除へのアクセスを判定する。
// 公開APIとMCPのどちらかを使う利用者が、自分の連携を管理できるようにする。
func fetchOAuthGrantSettingsAccess(
	ctx context.Context,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	featureFlagRepo *repository.FeatureFlagRepository,
	spaceIdentifier model.SpaceIdentifier,
	userID model.UserID,
	allowed func(policy.Authorizer) bool,
) (*apiSettingsAccess, error) {
	return fetchAPISettingsAccess(ctx, spaceRepo, spaceMemberRepo, featureFlagRepo, spaceIdentifier, userID, allowed,
		model.FeatureFlagPublicAPI, model.FeatureFlagRemoteMCPServer)
}

// fetchAPISettingsAccessは、スペースと有効なメンバーシップを解決し、allowedの権限と、flagsのうち
// 少なくとも1つが有効であることを確かめる。各画面の表示と操作に共通の判定を使うための処理である。
// メンバーでない場合とフラグが無効な場合は「見つからない」、権限が不足する場合は「権限がない」とする。
// メンバーでない場合は、あるスペースに誰が参加しているかを外部から試して確かめられないように、
// フラグが無効な場合は、公開前の画面の存在を知らせないようにするためである。
func fetchAPISettingsAccess(
	ctx context.Context,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	featureFlagRepo *repository.FeatureFlagRepository,
	spaceIdentifier model.SpaceIdentifier,
	userID model.UserID,
	allowed func(policy.Authorizer) bool,
	flags ...model.FeatureFlagName,
) (*apiSettingsAccess, error) {
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

	for _, flag := range flags {
		enabled, err := featureFlagRepo.IsEnabled(ctx, userID, flag)
		if err != nil {
			return nil, fmt.Errorf("フィーチャーフラグの判定に失敗: %w", err)
		}
		if enabled {
			return &apiSettingsAccess{space: space, spaceMember: spaceMember, authorizer: authorizer}, nil
		}
	}

	return nil, &model.AppError{Code: model.AppErrCodeResourceNotFound, UserMsg: i18n.T(ctx, "error_not_found_message")}
}
