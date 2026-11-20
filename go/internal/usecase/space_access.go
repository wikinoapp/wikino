package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// fetchSpaceUpdateAccessは、一般設定を変更するスペースを解決し、変更を許されない閲覧者を拒否する。
// 画面とその送信の両方がここを通るため、2つが誰を通すかについて別々の答えを出すことはない。
func fetchSpaceUpdateAccess(
	ctx context.Context,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	spaceIdentifier model.SpaceIdentifier,
	userID model.UserID,
) (*model.Space, error) {
	return fetchSpaceAccess(ctx, spaceRepo, spaceMemberRepo, spaceIdentifier, userID, policy.Authorizer.CanUpdateSpace)
}

// fetchSpaceDeleteAccessは、削除するスペースを解決し、削除を許されない閲覧者を拒否する。
// 確認画面とその送信の両方がここを通る。
func fetchSpaceDeleteAccess(
	ctx context.Context,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	spaceIdentifier model.SpaceIdentifier,
	userID model.UserID,
) (*model.Space, error) {
	return fetchSpaceAccess(ctx, spaceRepo, spaceMemberRepo, spaceIdentifier, userID, policy.Authorizer.CanDeleteSpace)
}

// fetchSpaceAccessは、スペースとそのアクティブなメンバーを解決し、canを満たさない閲覧者を拒否する。
//
// ユーザーがアクティブなメンバーでないスペースは「見つからない」として答える。あるスペースに誰が
// 参加しているかを、外部から試して確かめられないようにするためである。
func fetchSpaceAccess(
	ctx context.Context,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	spaceIdentifier model.SpaceIdentifier,
	userID model.UserID,
	can func(policy.Authorizer) bool,
) (*model.Space, error) {
	space, err := spaceRepo.FindByIdentifier(ctx, spaceIdentifier)
	if err != nil {
		return nil, fmt.Errorf("スペースの取得に失敗: %w", err)
	}
	if space == nil {
		return nil, notFoundError(ctx)
	}

	spaceMember, err := spaceMemberRepo.FindActiveBySpaceAndUser(ctx, space.ID, userID)
	if err != nil {
		return nil, fmt.Errorf("スペースメンバーの取得に失敗: %w", err)
	}
	if spaceMember == nil {
		return nil, notFoundError(ctx)
	}

	if !can(newAuthorizer(spaceMember, nil)) {
		return nil, forbiddenError(ctx)
	}

	return space, nil
}
