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
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// RestorePagesUsecaseは、ゴミ箱で選んだページをまとめてゴミ箱から戻す。
type RestorePagesUsecase struct {
	db              *sql.DB
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	pageRepo        *repository.PageRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
	validator       *validator.BulkRestoredPageCreateValidator
}

// NewRestorePagesUsecaseはRestorePagesUsecaseを生成する。
func NewRestorePagesUsecase(
	db *sql.DB,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	pageRepo *repository.PageRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	validator *validator.BulkRestoredPageCreateValidator,
) *RestorePagesUsecase {
	return &RestorePagesUsecase{
		db:              db,
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		pageRepo:        pageRepo,
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
		validator:       validator,
	}
}

// RestorePagesInputはまとめて復元する入力。PageIDsはフォームで選ばれたページのIDをそのまま渡す。
type RestorePagesInput struct {
	SpaceIdentifier model.SpaceIdentifier
	UserID          model.UserID
	PageIDs         []string
}

// RestorePagesOutputはまとめて復元した結果。
type RestorePagesOutput struct {
	Space *model.Space
}

// Executeは、選ばれたページがすべて復元できるときだけ、1つのトランザクションでゴミ箱から戻す。
//
// どのトピックでもページを復元できない閲覧者にはForbiddenのAppErrorを返す。復元できないページが
// 含まれていればValidationErrorを返し、何も戻さない。検証のあとに別の操作でページが動いて戻せなく
// なったときは、ConflictのAppErrorを返してトランザクションを取り消す。
func (uc *RestorePagesUsecase) Execute(ctx context.Context, input RestorePagesInput) (*RestorePagesOutput, error) {
	space, err := uc.spaceRepo.FindByIdentifier(ctx, input.SpaceIdentifier)
	if err != nil {
		return nil, fmt.Errorf("スペースの取得に失敗: %w", err)
	}
	if space == nil {
		return nil, notFoundError(ctx)
	}

	spaceMember, err := uc.spaceMemberRepo.FindActiveBySpaceAndUser(ctx, space.ID, input.UserID)
	if err != nil {
		return nil, fmt.Errorf("スペースメンバーの取得に失敗: %w", err)
	}
	if spaceMember == nil {
		return nil, forbiddenError(ctx)
	}

	access, err := loadTopicAccess(ctx, uc.topicRepo, uc.topicMemberRepo, space.ID, spaceMember, func(topicMember *model.TopicMember) policy.Authorizer {
		return newAuthorizer(spaceMember, topicMember)
	})
	if err != nil {
		return nil, err
	}
	restorableTopicIDs := access.restorableTopicIDs()
	if len(restorableTopicIDs) == 0 {
		return nil, forbiddenError(ctx)
	}

	trashedPages, err := uc.validator.Validate(ctx, validator.BulkRestoredPageCreateValidatorInput{
		PageIDs:            input.PageIDs,
		SpaceID:            space.ID,
		RestorableTopicIDs: restorableTopicIDs,
	})
	if err != nil {
		return nil, err
	}

	if err := uc.restorePages(ctx, space.ID, trashedPages, restorableTopicIDs, time.Now()); err != nil {
		return nil, err
	}

	return &RestorePagesOutput{Space: space}, nil
}

// restorePagesは、ゴミ箱に入っているページを1つのトランザクションで戻す。先行復元も含めた成功の
// 件数が選んだ件数に満たなければ、トランザクションを取り消して一部だけが戻った状態を残さない。
func (uc *RestorePagesUsecase) restorePages(ctx context.Context, spaceID model.SpaceID, pages []*model.Page, restorableTopicIDs map[model.TopicID]bool, now time.Time) error {
	if len(pages) == 0 {
		return nil
	}

	pageIDs := make([]model.PageID, len(pages))
	for i, pg := range pages {
		pageIDs[i] = pg.ID
	}
	topicIDs := make([]model.TopicID, 0, len(restorableTopicIDs))
	for topicID := range restorableTopicIDs {
		topicIDs = append(topicIDs, topicID)
	}

	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("トランザクションの開始に失敗: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	restored, err := uc.pageRepo.WithTx(tx).RestoreFromTrash(ctx, pageIDs, topicIDs, spaceID, now)
	if err != nil {
		return fmt.Errorf("ページの復元に失敗: %w", err)
	}
	if restored != int64(len(pageIDs)) {
		return &model.AppError{
			Code:     model.AppErrCodeConflict,
			UserMsg:  i18n.T(ctx, "validation_bulk_restored_page_ids_not_restorable"),
			Internal: fmt.Errorf("ページを%d件選んだが、%d件しか戻せなかった", len(pageIDs), restored),
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("トランザクションのコミットに失敗: %w", err)
	}
	return nil
}
