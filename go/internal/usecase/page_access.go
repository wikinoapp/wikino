package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// pageAccessData はページ操作に必要な共通データ
type pageAccessData struct {
	space       *model.Space
	spaceMember *model.SpaceMember
	page        *model.Page
	topic       *model.Topic
	topicMember *model.TopicMember
}

// pageAccessRepos はページアクセスデータ取得に必要なリポジトリ群
type pageAccessRepos struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	pageRepo        *repository.PageRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
}

// fetchPageAccessData はページ操作に必要な共通データを取得する
func fetchPageAccessData(ctx context.Context, repos pageAccessRepos, spaceIdentifier model.SpaceIdentifier, pageNumber int32, userID model.UserID) (*pageAccessData, error) {
	space, err := repos.spaceRepo.FindByIdentifier(ctx, spaceIdentifier)
	if err != nil {
		return nil, fmt.Errorf("スペースの取得に失敗: %w", err)
	}
	if space == nil {
		return nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	spaceMember, err := repos.spaceMemberRepo.FindActiveBySpaceAndUser(ctx, space.ID, userID)
	if err != nil {
		return nil, fmt.Errorf("スペースメンバーの取得に失敗: %w", err)
	}

	pg, err := repos.pageRepo.FindBySpaceAndNumber(ctx, space.ID, model.PageNumber(pageNumber))
	if err != nil {
		return nil, fmt.Errorf("ページの取得に失敗: %w", err)
	}
	if pg == nil {
		return nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	topic, err := repos.topicRepo.FindBySpaceAndID(ctx, space.ID, pg.TopicID)
	if err != nil {
		return nil, fmt.Errorf("トピックの取得に失敗: %w", err)
	}
	if topic == nil {
		return nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	var topicMember *model.TopicMember
	if spaceMember != nil {
		topicMember, err = repos.topicMemberRepo.FindBySpaceMemberAndTopic(ctx, space.ID, spaceMember.ID, pg.TopicID)
		if err != nil {
			return nil, fmt.Errorf("トピックメンバーの取得に失敗: %w", err)
		}
	}

	return &pageAccessData{
		space:       space,
		spaceMember: spaceMember,
		page:        pg,
		topic:       topic,
		topicMember: topicMember,
	}, nil
}

// authorizePageUpdate はページ更新の認可チェックを行う
func authorizePageUpdate(ctx context.Context, data *pageAccessData) error {
	if data.spaceMember == nil {
		return &model.AppError{
			Code:    model.AppErrCodeForbidden,
			UserMsg: i18n.T(ctx, "error_forbidden"),
		}
	}

	authorizer := newAuthorizer(data.spaceMember, data.topicMember)
	if !authorizer.CanUpdatePage() {
		return &model.AppError{
			Code:    model.AppErrCodeForbidden,
			UserMsg: i18n.T(ctx, "error_forbidden"),
		}
	}

	return nil
}
