package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// editDraftPagesLimit is the upper bound on the number of draft pages shown in the page editor's
// draft list column.
//
// [Ja] editDraftPagesLimit はページ編集画面の下書き一覧カラムに表示する下書きページ数の上限。
const editDraftPagesLimit = 20

// editDraftRevisionsLimit is the upper bound on the number of revisions shown in the page editor's
// edit history column.
//
// [Ja] editDraftRevisionsLimit はページ編集画面の編集履歴カラムに表示するリビジョン数の上限。
const editDraftRevisionsLimit = 20

// GetPageDetailUsecase はページ詳細画面のデータ取得ユースケース
type GetPageDetailUsecase struct {
	spaceRepo             *repository.SpaceRepository
	spaceMemberRepo       *repository.SpaceMemberRepository
	pageRepo              *repository.PageRepository
	draftPageRepo         *repository.DraftPageRepository
	draftPageRevisionRepo *repository.DraftPageRevisionRepository
	topicRepo             *repository.TopicRepository
	topicMemberRepo       *repository.TopicMemberRepository
	suggestionPageRepo    *repository.SuggestionPageRepository
	suggestionRepo        *repository.SuggestionRepository
}

// NewGetPageDetailUsecase は GetPageDetailUsecase を生成する
func NewGetPageDetailUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	pageRepo *repository.PageRepository,
	draftPageRepo *repository.DraftPageRepository,
	draftPageRevisionRepo *repository.DraftPageRevisionRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	suggestionPageRepo *repository.SuggestionPageRepository,
	suggestionRepo *repository.SuggestionRepository,
) *GetPageDetailUsecase {
	return &GetPageDetailUsecase{
		spaceRepo:             spaceRepo,
		spaceMemberRepo:       spaceMemberRepo,
		pageRepo:              pageRepo,
		draftPageRepo:         draftPageRepo,
		draftPageRevisionRepo: draftPageRevisionRepo,
		topicRepo:             topicRepo,
		topicMemberRepo:       topicMemberRepo,
		suggestionPageRepo:    suggestionPageRepo,
		suggestionRepo:        suggestionRepo,
	}
}

// GetPageDetailInput はページ詳細取得の入力パラメータ
type GetPageDetailInput struct {
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      int32
	UserID          model.UserID

	// IncludeDraftPages requests the space member's draft list (for the editor's draft list column).
	// The autosave fragment endpoint leaves it false to skip the extra query it does not need.
	//
	// [Ja] IncludeDraftPages はスペースメンバーの下書き一覧 (編集画面の下書き一覧カラム用) の取得を要求する。
	// オートセーブ用フラグメントエンドポイントは不要なクエリを避けるため false のままにする。
	IncludeDraftPages bool

	// IncludeDraftRevisions requests the draft's edit history (revision list and total count, for the
	// editor's edit history column). Callers that do not render the column leave it false to skip
	// the extra queries.
	//
	// [Ja] IncludeDraftRevisions は下書きの編集履歴 (編集画面の編集履歴カラム用のリビジョン一覧と総件数)
	// の取得を要求する。カラムを描画しない呼び出し元は不要なクエリを避けるため false のままにする。
	IncludeDraftRevisions bool
}

// GetPageDetailOutput はページ詳細取得の出力
type GetPageDetailOutput struct {
	Space         *model.Space
	SpaceMember   *model.SpaceMember
	Page          *model.Page
	Topic         *model.Topic
	TopicMember   *model.TopicMember
	DraftPage     *model.DraftPage
	Suggestion    *model.Suggestion
	CanUpdatePage bool

	// DraftPages is the space member's draft list for the editor's draft list column.
	// It is nil unless GetPageDetailInput.IncludeDraftPages is set.
	//
	// [Ja] DraftPages は編集画面の下書き一覧カラム用のスペースメンバーの下書き一覧。
	// GetPageDetailInput.IncludeDraftPages が指定されない限り nil。
	DraftPages []*model.DraftPage

	// DraftPageRevisions is the draft's revision list for the editor's edit history column, newest
	// first and capped at editDraftRevisionsLimit. It is nil unless
	// GetPageDetailInput.IncludeDraftRevisions is set and a draft exists.
	//
	// [Ja] DraftPageRevisions は編集画面の編集履歴カラム用の下書きのリビジョン一覧 (新しい順、
	// 最大 editDraftRevisionsLimit 件)。GetPageDetailInput.IncludeDraftRevisions が指定され、
	// かつ下書きが存在する場合以外は nil。
	DraftPageRevisions []*model.DraftPageRevision

	// DraftPageRevisionTotalCount is the total number of revisions of the draft (not capped by the
	// list limit). Version numbers in the edit history column are derived from this count.
	//
	// [Ja] DraftPageRevisionTotalCount は下書きのリビジョン総件数 (一覧の上限ではキャップしない)。
	// 編集履歴カラムのバージョン番号はこの件数から算出する。
	DraftPageRevisionTotalCount int64
}

// Execute はページ詳細画面に必要なデータを取得する
func (uc *GetPageDetailUsecase) Execute(ctx context.Context, input GetPageDetailInput) (*GetPageDetailOutput, error) {
	space, err := uc.spaceRepo.FindByIdentifier(ctx, input.SpaceIdentifier)
	if err != nil {
		return nil, fmt.Errorf("スペースの取得に失敗: %w", err)
	}
	if space == nil {
		return nil, nil
	}

	spaceMember, err := uc.spaceMemberRepo.FindActiveBySpaceAndUser(ctx, space.ID, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("スペースメンバーの取得に失敗: %w", err)
	}
	if spaceMember == nil {
		return nil, nil
	}

	pg, err := uc.pageRepo.FindBySpaceAndNumber(ctx, space.ID, model.PageNumber(input.PageNumber))
	if err != nil {
		return nil, fmt.Errorf("ページの取得に失敗: %w", err)
	}
	if pg == nil {
		return nil, nil
	}

	topicMember, err := uc.topicMemberRepo.FindBySpaceMemberAndTopic(ctx, space.ID, spaceMember.ID, pg.TopicID)
	if err != nil {
		return nil, fmt.Errorf("トピックメンバーの取得に失敗: %w", err)
	}

	topic, err := uc.topicRepo.FindBySpaceAndID(ctx, space.ID, pg.TopicID)
	if err != nil {
		return nil, fmt.Errorf("トピックの取得に失敗: %w", err)
	}
	if topic == nil {
		return nil, fmt.Errorf("ページのトピックが見つかりません: page_id=%s, topic_id=%s", pg.ID, pg.TopicID)
	}

	draftPage, err := uc.draftPageRepo.FindByPageAndMember(ctx, pg.ID, spaceMember.ID, space.ID)
	if err != nil {
		return nil, fmt.Errorf("下書きの取得に失敗: %w", err)
	}

	// 下書きが編集提案にリンクされている場合、編集提案を取得する
	var suggestion *model.Suggestion
	if draftPage != nil && draftPage.SuggestionPageID != nil {
		sp, err := uc.suggestionPageRepo.FindByID(ctx, *draftPage.SuggestionPageID, space.ID)
		if err != nil {
			return nil, fmt.Errorf("編集提案ページの取得に失敗: %w", err)
		}
		if sp != nil {
			suggestion, err = uc.suggestionRepo.FindByID(ctx, sp.SuggestionID, space.ID)
			if err != nil {
				return nil, fmt.Errorf("編集提案の取得に失敗: %w", err)
			}
		}
	}

	// 認可チェック
	authorizer := newAuthorizer(spaceMember, topicMember)
	canUpdatePage := authorizer.CanUpdatePage()

	// Fetch the space member's own draft list within this space for the editor's draft list column (only when requested).
	// [Ja] 編集画面の下書き一覧カラム用に、同一スペース内の自分の下書き一覧を取得する (要求された場合のみ)。
	var draftPages []*model.DraftPage
	if input.IncludeDraftPages {
		draftPages, err = uc.draftPageRepo.ListBySpaceMember(ctx, spaceMember.ID, space.ID, editDraftPagesLimit)
		if err != nil {
			return nil, fmt.Errorf("下書き一覧の取得に失敗: %w", err)
		}
	}

	// Fetch the draft's edit history for the editor's edit history column (only when requested).
	// Without a draft there are no revisions, so the column data stays empty.
	//
	// [Ja] 編集画面の編集履歴カラム用に、下書きの編集履歴を取得する (要求された場合のみ)。
	// 下書きが無ければリビジョンも存在しないため、カラム用データは空のままにする。
	var draftPageRevisions []*model.DraftPageRevision
	var draftPageRevisionTotalCount int64
	if input.IncludeDraftRevisions && draftPage != nil {
		draftPageRevisions, err = uc.draftPageRevisionRepo.ListByDraftPageID(ctx, draftPage.ID, space.ID, editDraftRevisionsLimit)
		if err != nil {
			return nil, fmt.Errorf("リビジョン一覧の取得に失敗: %w", err)
		}
		draftPageRevisionTotalCount, err = uc.draftPageRevisionRepo.CountByDraftPageID(ctx, draftPage.ID, space.ID)
		if err != nil {
			return nil, fmt.Errorf("リビジョン件数の取得に失敗: %w", err)
		}
	}

	return &GetPageDetailOutput{
		Space:                       space,
		SpaceMember:                 spaceMember,
		Page:                        pg,
		Topic:                       topic,
		TopicMember:                 topicMember,
		DraftPage:                   draftPage,
		Suggestion:                  suggestion,
		CanUpdatePage:               canUpdatePage,
		DraftPages:                  draftPages,
		DraftPageRevisions:          draftPageRevisions,
		DraftPageRevisionTotalCount: draftPageRevisionTotalCount,
	}, nil
}
