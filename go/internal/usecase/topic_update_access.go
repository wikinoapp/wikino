package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// topicUpdateAccessはトピックの一般設定の表示と保存に必要なものを保持する。
type topicUpdateAccess struct {
	Space *model.Space
	Topic *model.Topic
}

// fetchTopicUpdateAccessは、一般設定を開こうとしているトピックと、それが属するスペースを
// 解決し、変更を許されない閲覧者を拒否する。画面とその送信の両方がここを通るため、両者が誰を通すか
// について別々の答えを出すことはない。
//
// 閲覧者が変更できないトピックは「見つからない」として答える。Rails版も同じで、トピックが存在
// するが手が届かないと部外者に伝えることは、何も伝えないより多くを語ってしまう。
func fetchTopicUpdateAccess(
	ctx context.Context,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	spaceIdentifier model.SpaceIdentifier,
	topicNumber int32,
	userID model.UserID,
) (*topicUpdateAccess, error) {
	space, err := spaceRepo.FindByIdentifier(ctx, spaceIdentifier)
	if err != nil {
		return nil, fmt.Errorf("スペースの取得に失敗: %w", err)
	}
	if space == nil {
		return nil, notFoundError(ctx)
	}

	topic, err := topicRepo.FindBySpaceAndNumber(ctx, space.ID, topicNumber)
	if err != nil {
		return nil, fmt.Errorf("トピックの取得に失敗: %w", err)
	}
	if topic == nil {
		return nil, notFoundError(ctx)
	}

	spaceMember, err := spaceMemberRepo.FindActiveBySpaceAndUser(ctx, space.ID, userID)
	if err != nil {
		return nil, fmt.Errorf("スペースメンバーの取得に失敗: %w", err)
	}
	if spaceMember == nil {
		return nil, notFoundError(ctx)
	}

	topicMember, err := topicMemberRepo.FindBySpaceMemberAndTopic(ctx, space.ID, spaceMember.ID, topic.ID)
	if err != nil {
		return nil, fmt.Errorf("トピックメンバーの取得に失敗: %w", err)
	}

	if !newAuthorizer(spaceMember, topicMember).CanUpdateTopic() {
		return nil, &model.AppError{
			Code:    model.AppErrCodeForbidden,
			UserMsg: i18n.T(ctx, "error_forbidden"),
		}
	}

	return &topicUpdateAccess{Space: space, Topic: topic}, nil
}

// notFoundErrorはトピック設定が、求められたものに手が届かないときに返すエラーを組み立てる。
func notFoundError(ctx context.Context) error {
	return &model.AppError{
		Code:    model.AppErrCodeResourceNotFound,
		UserMsg: i18n.T(ctx, "error_not_found_message"),
	}
}
