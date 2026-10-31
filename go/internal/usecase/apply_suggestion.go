package usecase

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// ApplySuggestionUsecaseは編集提案反映ユースケース
type ApplySuggestionUsecase struct {
	db                    *sql.DB
	spaceRepo             *repository.SpaceRepository
	spaceMemberRepo       *repository.SpaceMemberRepository
	topicMemberRepo       *repository.TopicMemberRepository
	suggestionRepo        *repository.SuggestionRepository
	suggestionPageRepo    *repository.SuggestionPageRepository
	pageRepo              *repository.PageRepository
	pageRevisionRepo      *repository.PageRevisionRepository
	pageEditorRepo        *repository.PageEditorRepository
	attachmentRepo        *repository.AttachmentRepository
	pageAttachmentRefRepo *repository.PageAttachmentReferenceRepository
	draftPageRepo         *repository.DraftPageRepository
	applyValidator        *validator.SuggestionApplyValidator
}

// NewApplySuggestionUsecaseはApplySuggestionUsecaseを生成する
func NewApplySuggestionUsecase(
	db *sql.DB,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	suggestionRepo *repository.SuggestionRepository,
	suggestionPageRepo *repository.SuggestionPageRepository,
	pageRepo *repository.PageRepository,
	pageRevisionRepo *repository.PageRevisionRepository,
	pageEditorRepo *repository.PageEditorRepository,
	attachmentRepo *repository.AttachmentRepository,
	pageAttachmentRefRepo *repository.PageAttachmentReferenceRepository,
	draftPageRepo *repository.DraftPageRepository,
	applyValidator *validator.SuggestionApplyValidator,
) *ApplySuggestionUsecase {
	return &ApplySuggestionUsecase{
		db:                    db,
		spaceRepo:             spaceRepo,
		spaceMemberRepo:       spaceMemberRepo,
		topicMemberRepo:       topicMemberRepo,
		suggestionRepo:        suggestionRepo,
		suggestionPageRepo:    suggestionPageRepo,
		pageRepo:              pageRepo,
		pageRevisionRepo:      pageRevisionRepo,
		pageEditorRepo:        pageEditorRepo,
		attachmentRepo:        attachmentRepo,
		pageAttachmentRefRepo: pageAttachmentRefRepo,
		draftPageRepo:         draftPageRepo,
		applyValidator:        applyValidator,
	}
}

// ApplySuggestionInputは編集提案反映の入力パラメータ
type ApplySuggestionInput struct {
	SpaceIdentifier  model.SpaceIdentifier
	SuggestionNumber model.SuggestionNumber
	UserID           model.UserID
}

// ApplySuggestionOutputは編集提案反映の出力パラメータ
type ApplySuggestionOutput struct {
	Suggestion *model.Suggestion
}

// Executeは編集提案をトピックに反映する
func (uc *ApplySuggestionUsecase) Execute(ctx context.Context, input ApplySuggestionInput) (*ApplySuggestionOutput, error) {
	return retryOnDeadlock(ctx, func() (*ApplySuggestionOutput, error) {
		return uc.executeOnce(ctx, input)
	})
}

// executeOnceは取得・認可・検証から永続化までを1回実行する
func (uc *ApplySuggestionUsecase) executeOnce(ctx context.Context, input ApplySuggestionInput) (*ApplySuggestionOutput, error) {
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
	if output := uc.checkIdempotency(data.suggestion); output != nil {
		return output, nil
	}
	if err := uc.checkStatus(ctx, data.suggestion); err != nil {
		return nil, err
	}

	// 4. バリデーション (トランザクション外)
	validateOutput, err := uc.applyValidator.Validate(ctx, validator.SuggestionApplyValidatorInput{
		SpaceID:         data.space.ID,
		SpaceIdentifier: data.space.Identifier,
		Entries:         buildApplyValidatorEntries(data.suggestionPages, data.pages),
	})
	if err != nil {
		return nil, err
	}

	// 5. 永続化 (トランザクション)
	return uc.applySuggestion(ctx, data, validateOutput.PageReplacements)
}

// buildApplyValidatorEntriesはSuggestionPageとPageのペアからValidator入力を構築する
func buildApplyValidatorEntries(suggestionPages []*model.SuggestionPage, pages []*model.Page) []validator.SuggestionApplyValidatorEntry {
	topicIDByPageID := make(map[model.PageID]model.TopicID, len(pages))
	for _, p := range pages {
		topicIDByPageID[p.ID] = p.TopicID
	}

	entries := make([]validator.SuggestionApplyValidatorEntry, 0, len(suggestionPages))
	for _, sp := range suggestionPages {
		entries = append(entries, validator.SuggestionApplyValidatorEntry{
			PageID:  sp.PageID,
			TopicID: topicIDByPageID[sp.PageID],
			Title:   sp.Title,
		})
	}
	return entries
}

// applySuggestionDataはデータ取得結果をまとめた構造体
type applySuggestionData struct {
	space           *model.Space
	spaceMember     *model.SpaceMember
	topicMember     *model.TopicMember
	suggestion      *model.Suggestion
	suggestionPages []*model.SuggestionPage
	pages           []*model.Page
}

func (uc *ApplySuggestionUsecase) fetchData(ctx context.Context, input ApplySuggestionInput) (*applySuggestionData, error) {
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
	if spaceMember != nil {
		topicMember, err = uc.topicMemberRepo.FindBySpaceMemberAndTopic(ctx, space.ID, spaceMember.ID, suggestion.TopicID)
		if err != nil {
			return nil, fmt.Errorf("トピックメンバーの取得に失敗: %w", err)
		}
	}

	suggestionPages, err := uc.suggestionPageRepo.ListBySuggestionID(ctx, suggestion.ID, space.ID)
	if err != nil {
		return nil, fmt.Errorf("編集提案ページの取得に失敗: %w", err)
	}

	pageIDs := make([]model.PageID, len(suggestionPages))
	for i, sp := range suggestionPages {
		pageIDs[i] = sp.PageID
	}

	pages, err := uc.pageRepo.FindByIDs(ctx, pageIDs, space.ID)
	if err != nil {
		return nil, fmt.Errorf("ページの取得に失敗: %w", err)
	}

	return &applySuggestionData{
		space:           space,
		spaceMember:     spaceMember,
		topicMember:     topicMember,
		suggestion:      suggestion,
		suggestionPages: suggestionPages,
		pages:           pages,
	}, nil
}

func (uc *ApplySuggestionUsecase) authorize(ctx context.Context, data *applySuggestionData) error {
	if data.spaceMember == nil {
		return &model.AppError{
			Code:    model.AppErrCodeForbidden,
			UserMsg: i18n.T(ctx, "error_forbidden"),
		}
	}

	authorizer := newAuthorizer(data.spaceMember, data.topicMember)
	if !authorizer.CanApplySuggestion() {
		return &model.AppError{
			Code:    model.AppErrCodeForbidden,
			UserMsg: i18n.T(ctx, "error_forbidden"),
		}
	}

	return nil
}

// checkIdempotencyは既に反映済みの場合に成功出力を返す (べき等性)
func (uc *ApplySuggestionUsecase) checkIdempotency(suggestion *model.Suggestion) *ApplySuggestionOutput {
	if suggestion.Status == model.SuggestionStatusApplied {
		return &ApplySuggestionOutput{Suggestion: suggestion}
	}
	return nil
}

func (uc *ApplySuggestionUsecase) checkStatus(ctx context.Context, suggestion *model.Suggestion) error {
	if suggestion.Status != model.SuggestionStatusOpen {
		return &model.AppError{
			Code:    model.AppErrCodeConflict,
			UserMsg: i18n.T(ctx, "suggestion_apply_error"),
		}
	}
	return nil
}

func (uc *ApplySuggestionUsecase) applySuggestion(ctx context.Context, data *applySuggestionData, pageReplacements []validator.SuggestionApplyPageReplacement) (*ApplySuggestionOutput, error) {
	now := time.Now()
	spaceID := data.space.ID

	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗しました: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	suggestionRepo := uc.suggestionRepo.WithTx(tx)
	pageRepo := uc.pageRepo.WithTx(tx)
	pageRevisionRepo := uc.pageRevisionRepo.WithTx(tx)
	pageEditorRepo := uc.pageEditorRepo.WithTx(tx)
	topicMemberRepo := uc.topicMemberRepo.WithTx(tx)
	attachmentRepo := uc.attachmentRepo.WithTx(tx)
	pageAttachmentRefRepo := uc.pageAttachmentRefRepo.WithTx(tx)

	// 並行する公開がページと編集提案のリンクを付け替えていれば、そのコミットを待つ。
	// 取得済みのリンクの記録を反映すると、先行する付け替えを古いIDで上書きしてしまう
	pageIDs := make([]model.PageID, len(data.pages))
	for i, page := range data.pages {
		pageIDs[i] = page.ID
	}
	if err := pageRepo.LockByIDs(ctx, pageIDs, spaceID); err != nil {
		return nil, fmt.Errorf("反映先のページのロックに失敗しました: %w", err)
	}
	currentPages, err := uc.suggestionPageRepo.WithTx(tx).ListBySuggestionID(ctx, data.suggestion.ID, spaceID)
	if err != nil {
		return nil, fmt.Errorf("編集提案のリンクの再取得に失敗しました: %w", err)
	}
	currentByID := make(map[model.SuggestionPageID]*model.SuggestionPage, len(currentPages))
	for _, sp := range currentPages {
		currentByID[sp.ID] = sp
	}
	for _, sp := range data.suggestionPages {
		current := currentByID[sp.ID]
		// 本文やタイトルも変わっていれば、検証していない内容を反映しない。
		// リンクの付け替えはUpdatedAtを変えないため、その場合だけ記録を引き継げる
		if current == nil || !current.UpdatedAt.Equal(sp.UpdatedAt) {
			return nil, &model.AppError{
				Code:    model.AppErrCodeConflict,
				UserMsg: i18n.T(ctx, "error_suggestion_conflict_during_apply"),
			}
		}
		sp.LinkedPageIDs = current.LinkedPageIDs
	}

	// 競合する未公開ページを論理削除
	// トランザクション開始後にページの状態が変わっている可能性があるため、
	// 論理削除直前に再取得し「未公開かつ本文が空」条件を再確認する (TOCTOU対策)。
	// 論理削除したページへのリンクは、ページの書き込みの後に置き換えたページへ付け替える
	var discardedReplacements []validator.SuggestionApplyPageReplacement
	if len(pageReplacements) > 0 {
		conflictingPageIDs := make([]model.PageID, len(pageReplacements))
		for i, r := range pageReplacements {
			conflictingPageIDs[i] = r.ConflictingPageID
		}
		conflictPages, err := pageRepo.FindByIDs(ctx, conflictingPageIDs, spaceID)
		if err != nil {
			return nil, fmt.Errorf("競合ページの再取得に失敗しました: %w", err)
		}

		conflictPageMap := make(map[model.PageID]*model.Page, len(conflictPages))
		for _, cp := range conflictPages {
			conflictPageMap[cp.ID] = cp
		}

		for _, replacement := range pageReplacements {
			conflictID := replacement.ConflictingPageID
			conflictPage, ok := conflictPageMap[conflictID]
			if !ok {
				continue
			}
			if conflictPage.PublishedAt != nil || conflictPage.Body != "" {
				return nil, &model.AppError{
					Code:    model.AppErrCodeConflict,
					UserMsg: i18n.T(ctx, "error_suggestion_conflict_during_apply"),
				}
			}
			if err := pageRepo.DiscardByID(ctx, conflictID, spaceID, now); err != nil {
				return nil, fmt.Errorf("競合する未公開ページの論理削除に失敗しました: %w", err)
			}
			discardedReplacements = append(discardedReplacements, replacement)
		}
	}

	// ページをマップに変換
	pageMap := make(map[model.PageID]*model.Page, len(data.pages))
	for _, p := range data.pages {
		pageMap[p.ID] = p
	}

	// 各SuggestionPageの内容をPageに反映
	for _, sp := range data.suggestionPages {
		page := pageMap[sp.PageID]
		if page == nil {
			return nil, fmt.Errorf("ページが見つかりません: %s", sp.PageID)
		}

		// Pageを更新
		_, err = pageRepo.Update(ctx, repository.UpdatePageInput{
			ID:                        sp.PageID,
			SpaceID:                   spaceID,
			TopicID:                   page.TopicID,
			Title:                     sp.Title,
			Body:                      sp.Body,
			LinkedPageIDs:             sp.LinkedPageIDs,
			FeaturedImageAttachmentID: sp.FeaturedImageAttachmentID,
			ModifiedAt:                now,
			PublishedAt:               &now,
		})
		if err != nil {
			return nil, fmt.Errorf("ページの更新に失敗しました: %w", err)
		}

		// 添付ファイル参照の同期
		if err := syncAttachmentReferences(ctx, sp.Body, sp.PageID, spaceID, attachmentRepo, pageAttachmentRefRepo); err != nil {
			return nil, fmt.Errorf("添付ファイル参照の同期に失敗しました: %w", err)
		}

		// PageRevisionを作成 (スナップショット)
		var title string
		if sp.Title != nil {
			title = *sp.Title
		}
		_, err = pageRevisionRepo.Create(ctx, repository.CreatePageRevisionInput{
			SpaceID:       spaceID,
			SpaceMemberID: data.spaceMember.ID,
			PageID:        sp.PageID,
			Title:         title,
			Body:          sp.Body,
		})
		if err != nil {
			return nil, fmt.Errorf("ページリビジョンの作成に失敗しました: %w", err)
		}

		// PageEditorを追加・更新
		pageEditor, err := pageEditorRepo.FindOrCreate(ctx, repository.FindOrCreateInput{
			SpaceID:            spaceID,
			PageID:             sp.PageID,
			SpaceMemberID:      data.spaceMember.ID,
			LastPageModifiedAt: now,
		})
		if err != nil {
			return nil, fmt.Errorf("ページ編集者の追加に失敗しました: %w", err)
		}

		_, err = pageEditorRepo.UpdateLastPageModifiedAt(ctx, repository.UpdateLastPageModifiedAtInput{
			ID:                 pageEditor.ID,
			SpaceID:            spaceID,
			LastPageModifiedAt: now,
		})
		if err != nil {
			return nil, fmt.Errorf("ページ編集者の更新に失敗しました: %w", err)
		}

		// TopicMemberのlast_page_modified_atを更新
		err = topicMemberRepo.UpdateLastPageModifiedAt(ctx, spaceID, page.TopicID, data.spaceMember.ID, now)
		if err != nil {
			return nil, fmt.Errorf("トピックメンバーの更新に失敗しました: %w", err)
		}
	}

	draftPageRepo := uc.draftPageRepo.WithTx(tx)
	suggestionPageRepo := uc.suggestionPageRepo.WithTx(tx)
	for _, replacement := range discardedReplacements {
		if err := reassignLinkedPageIDs(ctx, spaceID, replacement.ConflictingPageID, replacement.PageID, pageRepo, draftPageRepo, suggestionPageRepo); err != nil {
			return nil, err
		}
	}

	// 編集提案のステータスを「反映済み」に変更
	updatedSuggestion, err := suggestionRepo.UpdateStatus(ctx, repository.UpdateStatusInput{
		ID:        data.suggestion.ID,
		SpaceID:   spaceID,
		Status:    model.SuggestionStatusApplied,
		AppliedAt: &now,
	})
	if err != nil {
		return nil, fmt.Errorf("編集提案のステータス更新に失敗しました: %w", err)
	}

	// 下書きのsuggestion_page_idをクリアして再利用可能にする
	if err := draftPageRepo.ClearSuggestionPageIDsBySuggestionID(ctx, data.suggestion.ID, spaceID); err != nil {
		return nil, fmt.Errorf("下書きのsuggestion_page_idクリアに失敗しました: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗しました: %w", err)
	}

	return &ApplySuggestionOutput{
		Suggestion: updatedSuggestion,
	}, nil
}
