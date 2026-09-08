package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// fetchExportAccess resolves the space an export belongs to together with the membership of the
// user acting on it, and refuses the ones that may not export it. Every entry into the export
// feature goes through it, so the four screens and the start of an export cannot come to different
// answers about who is allowed in.
//
// A space the user is not an active member of is answered as not found, so that membership of a
// space is not something an outsider can probe for.
//
// [Ja] fetchExportAccess は、エクスポートが属するスペースと、それを操作するユーザーのメンバー
// シップを解決し、エクスポートを許されないものを拒否する。エクスポート機能への入口はすべてここを
// 通るため、4 つの画面とエクスポートの開始が、誰を通すかについて別々の答えを出すことはない。
//
// ユーザーがアクティブなメンバーでないスペースは「見つからない」として答える。あるスペースに誰が
// 参加しているかを、外部から試して確かめられないようにするためである。
func fetchExportAccess(
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
		return nil, nil, &model.AppError{Code: model.AppErrCodeResourceNotFound, UserMsg: i18n.T(ctx, "error_not_found_message")}
	}

	spaceMember, err := spaceMemberRepo.FindActiveBySpaceAndUser(ctx, space.ID, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("スペースメンバーの取得に失敗: %w", err)
	}
	if spaceMember == nil {
		return nil, nil, &model.AppError{Code: model.AppErrCodeResourceNotFound, UserMsg: i18n.T(ctx, "error_not_found_message")}
	}

	if !newAuthorizer(spaceMember, nil).CanExportSpace() {
		return nil, nil, &model.AppError{
			Code:    model.AppErrCodeForbidden,
			UserMsg: i18n.T(ctx, "error_forbidden"),
		}
	}

	return space, spaceMember, nil
}
