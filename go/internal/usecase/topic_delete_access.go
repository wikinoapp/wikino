package usecase

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// fetchTopicDeleteAccessは、削除しようとしているトピックと、それが属するスペースを解決し、
// トピックを削除できない閲覧者を拒否する。確認画面とその送信の両方がここを通るため、両者が誰を
// 通すかについて別々の答えを出すことはない。
func fetchTopicDeleteAccess(
	ctx context.Context,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	spaceIdentifier model.SpaceIdentifier,
	topicNumber int32,
	userID model.UserID,
) (*topicSettingsTarget, error) {
	target, err := fetchTopicSettingsTarget(ctx, spaceRepo, spaceMemberRepo, topicRepo, topicMemberRepo, spaceIdentifier, topicNumber, userID)
	if err != nil {
		return nil, err
	}

	if !target.Authorizer.CanDeleteTopic() {
		return nil, forbiddenError(ctx)
	}

	return target, nil
}
