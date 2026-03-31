package usecase

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// StartSuggestionPageEditUsecase は編集提案ページの編集開始ユースケース
type StartSuggestionPageEditUsecase struct {
	db                 *sql.DB
	spaceRepo          *repository.SpaceRepository
	spaceMemberRepo    *repository.SpaceMemberRepository
	topicMemberRepo    *repository.TopicMemberRepository
	suggestionRepo     *repository.SuggestionRepository
	suggestionPageRepo *repository.SuggestionPageRepository
	draftPageRepo      *repository.DraftPageRepository
	pageRepo           *repository.PageRepository
}

// NewStartSuggestionPageEditUsecase は StartSuggestionPageEditUsecase を生成する
func NewStartSuggestionPageEditUsecase(
	db *sql.DB,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	suggestionRepo *repository.SuggestionRepository,
	suggestionPageRepo *repository.SuggestionPageRepository,
	draftPageRepo *repository.DraftPageRepository,
	pageRepo *repository.PageRepository,
) *StartSuggestionPageEditUsecase {
	return &StartSuggestionPageEditUsecase{
		db:                 db,
		spaceRepo:          spaceRepo,
		spaceMemberRepo:    spaceMemberRepo,
		topicMemberRepo:    topicMemberRepo,
		suggestionRepo:     suggestionRepo,
		suggestionPageRepo: suggestionPageRepo,
		draftPageRepo:      draftPageRepo,
		pageRepo:           pageRepo,
	}
}

// StartSuggestionPageEditStatus は編集開始の結果ステータス
type StartSuggestionPageEditStatus int

const (
	// StartSuggestionPageEditRedirect はページ編集画面にリダイレクト可能な状態
	StartSuggestionPageEditRedirect StartSuggestionPageEditStatus = iota
	// StartSuggestionPageEditConflict は既存の下書きが存在する状態
	StartSuggestionPageEditConflict
)

// ConflictDraftKind はコンフリクト時の下書きの種類
type ConflictDraftKind int

const (
	// ConflictDraftKindNormal は通常の下書き（suggestion_page_idがNULL）
	ConflictDraftKindNormal ConflictDraftKind = iota
	// ConflictDraftKindOtherSuggestion は別の編集提案にリンクされた下書き
	ConflictDraftKindOtherSuggestion
)

// StartSuggestionPageEditInput は編集開始の入力パラメータ
type StartSuggestionPageEditInput struct {
	SpaceIdentifier  model.SpaceIdentifier
	SuggestionNumber model.SuggestionNumber
	SuggestionPageID model.SuggestionPageID
	UserID           model.UserID
	Force            bool
}

// StartSuggestionPageEditOutput は編集開始の出力
type StartSuggestionPageEditOutput struct {
	Status            StartSuggestionPageEditStatus
	PageNumber        model.PageNumber
	ConflictDraftKind ConflictDraftKind
}

// Execute は編集提案ページの編集を開始する
func (uc *StartSuggestionPageEditUsecase) Execute(ctx context.Context, input StartSuggestionPageEditInput) (*StartSuggestionPageEditOutput, error) {
	// 1. データ取得
	data, err := uc.fetchData(ctx, input)
	if err != nil {
		return nil, err
	}

	// 2. 認可チェック
	if err := uc.authorize(ctx, data); err != nil {
		return nil, err
	}

	// 3. ステータスチェック
	if err := uc.checkStatus(ctx, data.suggestion); err != nil {
		return nil, err
	}

	// 4. 編集開始処理
	return uc.startEdit(ctx, data, input.Force)
}

// startSuggestionPageEditData はデータ取得結果をまとめた構造体
type startSuggestionPageEditData struct {
	space          *model.Space
	spaceMember    *model.SpaceMember
	topicMember    *model.TopicMember
	suggestion     *model.Suggestion
	suggestionPage *model.SuggestionPage
	page           *model.Page
}

func (uc *StartSuggestionPageEditUsecase) fetchData(ctx context.Context, input StartSuggestionPageEditInput) (*startSuggestionPageEditData, error) {
	space, err := uc.spaceRepo.FindByIdentifier(ctx, input.SpaceIdentifier)
	if err != nil {
		return nil, fmt.Errorf("スペースの取得に失敗: %w", err)
	}
	if space == nil {
		return nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	spaceMember, err := uc.spaceMemberRepo.FindActiveBySpaceAndUser(ctx, space.ID, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("スペースメンバーの取得に失敗: %w", err)
	}

	suggestion, err := uc.suggestionRepo.FindBySpaceAndNumber(ctx, space.ID, input.SuggestionNumber)
	if err != nil {
		return nil, fmt.Errorf("編集提案の取得に失敗: %w", err)
	}
	if suggestion == nil {
		return nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	var topicMember *model.TopicMember
	if spaceMember != nil && spaceMember.Role != model.SpaceMemberRoleOwner {
		topicMember, err = uc.topicMemberRepo.FindBySpaceMemberAndTopic(ctx, space.ID, spaceMember.ID, suggestion.TopicID)
		if err != nil {
			return nil, fmt.Errorf("トピックメンバーの取得に失敗: %w", err)
		}
	}

	// 編集提案ページが現在の編集提案に属していることを検証
	suggestionPage, err := uc.suggestionPageRepo.FindByID(ctx, input.SuggestionPageID, space.ID)
	if err != nil {
		return nil, fmt.Errorf("編集提案ページの取得に失敗: %w", err)
	}
	if suggestionPage == nil || suggestionPage.SuggestionID != suggestion.ID {
		return nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	// ページを取得（番号の取得とトピックID確認用）
	pages, err := uc.pageRepo.FindByIDs(ctx, []model.PageID{suggestionPage.PageID}, space.ID)
	if err != nil {
		return nil, fmt.Errorf("ページの取得に失敗: %w", err)
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("ページが見つかりません: %s", suggestionPage.PageID)
	}

	return &startSuggestionPageEditData{
		space:          space,
		spaceMember:    spaceMember,
		topicMember:    topicMember,
		suggestion:     suggestion,
		suggestionPage: suggestionPage,
		page:           pages[0],
	}, nil
}

func (uc *StartSuggestionPageEditUsecase) authorize(ctx context.Context, data *startSuggestionPageEditData) error {
	if data.spaceMember == nil {
		return &model.AppError{
			Code:    model.AppErrCodeForbidden,
			UserMsg: i18n.T(ctx, "error_forbidden"),
		}
	}

	topicPolicy := policy.NewTopicPolicy(data.spaceMember, data.topicMember)
	if !topicPolicy.CanEditSuggestionPage(data.suggestion) {
		return &model.AppError{
			Code:    model.AppErrCodeForbidden,
			UserMsg: i18n.T(ctx, "error_forbidden"),
		}
	}

	return nil
}

func (uc *StartSuggestionPageEditUsecase) checkStatus(ctx context.Context, suggestion *model.Suggestion) error {
	if suggestion.Status != model.SuggestionStatusOpen {
		return &model.AppError{
			Code:    model.AppErrCodeConflict,
			UserMsg: i18n.T(ctx, "suggestion_page_edit_error_not_open"),
		}
	}
	return nil
}

func (uc *StartSuggestionPageEditUsecase) startEdit(ctx context.Context, data *startSuggestionPageEditData, force bool) (*StartSuggestionPageEditOutput, error) {
	sp := data.suggestionPage
	page := data.page

	// 既存の下書きを確認
	draft, err := uc.draftPageRepo.FindByPageAndMember(ctx, sp.PageID, data.spaceMember.ID, data.space.ID)
	if err != nil {
		return nil, fmt.Errorf("下書きの取得に失敗: %w", err)
	}

	// 下書きが存在し、同じ編集提案ページにリンク済みの場合はそのままリダイレクト
	if draft != nil && draft.SuggestionPageID != nil && *draft.SuggestionPageID == sp.ID {
		return &StartSuggestionPageEditOutput{
			Status:     StartSuggestionPageEditRedirect,
			PageNumber: page.Number,
		}, nil
	}

	// 下書きが存在し、Force=falseの場合はコンフリクト
	if draft != nil && !force {
		draftKind := ConflictDraftKindNormal
		if draft.SuggestionPageID != nil {
			draftKind = ConflictDraftKindOtherSuggestion
		}
		return &StartSuggestionPageEditOutput{
			Status:            StartSuggestionPageEditConflict,
			PageNumber:        page.Number,
			ConflictDraftKind: draftKind,
		}, nil
	}

	// トランザクションを開始
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	draftPageRepo := uc.draftPageRepo.WithTx(tx)
	now := time.Now()

	if draft != nil {
		// Force=true: 既存の下書きを編集提案の内容で上書き
		_, err = draftPageRepo.Update(ctx, repository.UpdateDraftPageInput{
			ID:                        draft.ID,
			SpaceID:                   data.space.ID,
			TopicID:                   page.TopicID,
			Title:                     sp.Title,
			Body:                      sp.Body,
			BodyHTML:                  sp.BodyHTML,
			LinkedPageIDs:             sp.LinkedPageIDs,
			FeaturedImageAttachmentID: sp.FeaturedImageAttachmentID,
			ModifiedAt:                now,
		})
		if err != nil {
			return nil, fmt.Errorf("下書きの更新に失敗: %w", err)
		}

		// suggestion_page_idを設定
		_, err = draftPageRepo.UpdateSuggestionPageID(ctx, draft.ID, data.space.ID, &sp.ID)
		if err != nil {
			return nil, fmt.Errorf("下書きのsuggestion_page_idの更新に失敗: %w", err)
		}
	} else {
		// 下書きが存在しない場合は新規作成
		_, err = draftPageRepo.Create(ctx, repository.CreateDraftPageInput{
			SpaceID:                   data.space.ID,
			PageID:                    sp.PageID,
			SpaceMemberID:             data.spaceMember.ID,
			TopicID:                   page.TopicID,
			SuggestionPageID:          &sp.ID,
			Title:                     sp.Title,
			Body:                      sp.Body,
			BodyHTML:                  sp.BodyHTML,
			LinkedPageIDs:             sp.LinkedPageIDs,
			FeaturedImageAttachmentID: sp.FeaturedImageAttachmentID,
			ModifiedAt:                now,
		})
		if err != nil {
			return nil, fmt.Errorf("下書きの作成に失敗: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗: %w", err)
	}

	return &StartSuggestionPageEditOutput{
		Status:     StartSuggestionPageEditRedirect,
		PageNumber: page.Number,
	}, nil
}
