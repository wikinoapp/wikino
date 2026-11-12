package usecase

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// searchResultLimitは、検索の結果に出すページの上限。
const searchResultLimit = 25

// GetSearchShowUsecaseは、検索の画面 (GET /search) に出す結果を集める。
type GetSearchShowUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	pageRepo        *repository.PageRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
	validator       *validator.SearchShowValidator
}

// NewGetSearchShowUsecaseはGetSearchShowUsecaseを生成する。
func NewGetSearchShowUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	pageRepo *repository.PageRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	validator *validator.SearchShowValidator,
) *GetSearchShowUsecase {
	return &GetSearchShowUsecase{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		pageRepo:        pageRepo,
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
		validator:       validator,
	}
}

// GetSearchShowInputは検索の入力。Queryは前後の空白を除いたキーワード。
type GetSearchShowInput struct {
	UserID model.UserID
	Query  string
}

// GetSearchShowOutputは検索の結果。
type GetSearchShowOutput struct {
	// Searchedは検索したか (スペースの指定を除いたキーワードがあったか) を表す。検索していないときは
	// 結果が無いことを伝えない
	Searched bool
	// HasSpaceFilterは `space:` でスペースを指定したかを表す
	HasSpaceFilter bool
	Pages          []*model.Page

	// TopicMapは結果の各ページのトピックを、スペースの識別子と名前を添えて引けるようにする
	TopicMap map[model.TopicID]*model.Topic
}

// Executeは、キーワードを検証し、公開トピックのページと、参加しているスペースで閲覧者が開ける
// トピックのページから、タイトルにすべてのキーワードを含むページを返す。
//
// 参加しているスペースのページは、スペースごとに開けるトピックを解決してから絞る。非公開トピックの
// ページは、そのトピックを開けるときだけ結果に含める。
func (uc *GetSearchShowUsecase) Execute(ctx context.Context, input GetSearchShowInput) (*GetSearchShowOutput, error) {
	conditions, err := uc.validator.Validate(ctx, validator.SearchShowValidatorInput{Query: input.Query})
	if err != nil {
		return nil, err
	}

	output := &GetSearchShowOutput{
		Searched:       len(conditions.Keywords) > 0,
		HasSpaceFilter: len(conditions.SpaceIdentifiers) > 0,
		Pages:          []*model.Page{},
		TopicMap:       map[model.TopicID]*model.Topic{},
	}
	if !output.Searched {
		return output, nil
	}

	memberTopicIDs, err := uc.memberTopicIDs(ctx, input.UserID, conditions.SpaceIdentifiers)
	if err != nil {
		return nil, err
	}

	pages, err := uc.pageRepo.SearchByTitle(ctx, repository.SearchByTitleInput{
		Keywords:         conditions.Keywords,
		MemberTopicIDs:   memberTopicIDs,
		SpaceIdentifiers: conditions.SpaceIdentifiers,
		Limit:            searchResultLimit,
	})
	if err != nil {
		return nil, fmt.Errorf("ページの検索に失敗: %w", err)
	}

	topics, err := uc.topicRepo.ListWithSpacesByPages(ctx, pages)
	if err != nil {
		return nil, fmt.Errorf("検索の結果のトピックの取得に失敗: %w", err)
	}
	for _, topic := range topics {
		output.TopicMap[topic.ID] = topic
	}
	output.Pages = pages

	return output, nil
}

// memberTopicIDsは、閲覧者が参加しているスペースで開けるトピックを返す。spaceIdentifiersを
// 渡すと、そのスペースに絞る。参加しているスペースの数だけトピックとトピックメンバーを引くが、
// 1人が参加するスペースの数は限られる。
func (uc *GetSearchShowUsecase) memberTopicIDs(ctx context.Context, userID model.UserID, spaceIdentifiers []model.SpaceIdentifier) ([]model.TopicID, error) {
	spaces, err := uc.spaceRepo.ListActiveByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("参加しているスペースの取得に失敗: %w", err)
	}

	spaceIDs := make([]model.SpaceID, 0, len(spaces))
	for _, space := range spaces {
		if len(spaceIdentifiers) > 0 && !containsSpaceIdentifier(spaceIdentifiers, space.Identifier) {
			continue
		}
		spaceIDs = append(spaceIDs, space.ID)
	}

	spaceMembers, err := uc.spaceMemberRepo.ListActiveByUserAndSpaceIDs(ctx, userID, spaceIDs)
	if err != nil {
		return nil, fmt.Errorf("スペースメンバーの取得に失敗: %w", err)
	}

	var topicIDs []model.TopicID
	for _, spaceMember := range spaceMembers {
		access, err := loadTopicAccess(ctx, uc.topicRepo, uc.topicMemberRepo, spaceMember.SpaceID, spaceMember, func(topicMember *model.TopicMember) policy.Authorizer {
			return newAuthorizer(spaceMember, topicMember)
		})
		if err != nil {
			return nil, err
		}
		for _, topic := range access.visibleTopics() {
			topicIDs = append(topicIDs, topic.ID)
		}
	}
	return topicIDs, nil
}

// containsSpaceIdentifierは、識別子の一覧が指定した識別子を含むかを、大文字と小文字を区別せずに返す。
// スペースの識別子は大文字と小文字を区別しないため (citext)。
func containsSpaceIdentifier(identifiers []model.SpaceIdentifier, target model.SpaceIdentifier) bool {
	return slices.ContainsFunc(identifiers, func(identifier model.SpaceIdentifier) bool {
		return strings.EqualFold(string(identifier), string(target))
	})
}
