package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// fetchTopicCreateAccess resolves the space a topic is created in together with the membership of
// the user creating it, and refuses the ones that may not create one. The form and its submission
// both go through it, so the screen and the create cannot come to different answers about who is
// allowed in.
//
// A space the user is not an active member of is answered as not found, so that membership of a
// space is not something an outsider can probe for.
//
// [Ja] fetchTopicCreateAccess は、トピックを作成するスペースと、それを作成するユーザーのメンバー
// シップを解決し、作成を許されないものを拒否する。フォームとその送信の両方がここを通るため、
// 画面と作成処理が、誰を通すかについて別々の答えを出すことはない。
//
// ユーザーがアクティブなメンバーでないスペースは「見つからない」として答える。あるスペースに誰が
// 参加しているかを、外部から試して確かめられないようにするためである。
func fetchTopicCreateAccess(
	ctx context.Context,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	spaceIdentifier model.SpaceIdentifier,
	userID model.UserID,
) (*model.Space, *model.SpaceMember, error) {
	space, err := spaceRepo.FindByIdentifier(ctx, spaceIdentifier)
	if err != nil {
		return nil, nil, fmt.Errorf("スペースの取得に失敗: %w", err)
	}
	if space == nil {
		return nil, nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	spaceMember, err := spaceMemberRepo.FindActiveBySpaceAndUser(ctx, space.ID, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("スペースメンバーの取得に失敗: %w", err)
	}
	if spaceMember == nil {
		return nil, nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	if !newAuthorizer(spaceMember, nil).CanCreateTopic() {
		return nil, nil, &model.AppError{
			Code:    model.AppErrCodeForbidden,
			UserMsg: i18n.T(ctx, "error_forbidden"),
		}
	}

	return space, spaceMember, nil
}
