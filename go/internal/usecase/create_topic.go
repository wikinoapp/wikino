package usecase

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// CreateTopicUsecase creates a topic and joins its creator to it.
//
// [Ja] CreateTopicUsecase はトピックを作成し、作成者をそのトピックに参加させる。
type CreateTopicUsecase struct {
	db              *sql.DB
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
	createValidator *validator.TopicCreateValidator
}

// NewCreateTopicUsecase は CreateTopicUsecase を生成する
func NewCreateTopicUsecase(
	db *sql.DB,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	createValidator *validator.TopicCreateValidator,
) *CreateTopicUsecase {
	return &CreateTopicUsecase{
		db:              db,
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
		createValidator: createValidator,
	}
}

// CreateTopicInput はトピック作成の入力パラメータ。
// Visibility はフォームが送信した文字列で、変換はバリデーターが行う。
type CreateTopicInput struct {
	SpaceIdentifier model.SpaceIdentifier
	UserID          model.UserID
	Name            string
	Description     string
	Visibility      string
}

// CreateTopicOutput は作成されたトピックとそれが属するスペースを保持する
type CreateTopicOutput struct {
	Space *model.Space
	Topic *model.Topic
}

// Execute はトピックを作成する
func (uc *CreateTopicUsecase) Execute(ctx context.Context, input CreateTopicInput) (*CreateTopicOutput, error) {
	// 1. データ取得と認可チェック
	space, spaceMember, err := fetchTopicCreateAccess(ctx, uc.spaceRepo, uc.spaceMemberRepo, input.SpaceIdentifier, input.UserID)
	if err != nil {
		return nil, err
	}

	// 2. バリデーション
	visibility, err := uc.createValidator.Validate(ctx, validator.TopicCreateValidatorInput{
		Name:        input.Name,
		Description: input.Description,
		Visibility:  input.Visibility,
		SpaceID:     space.ID,
	})
	if err != nil {
		return nil, err
	}

	// 3. 永続化 (トランザクション)
	return uc.createTopic(ctx, space, spaceMember, input, visibility)
}

// createTopic creates the topic and joins its creator to it in one transaction, so that a topic
// nobody has joined is never left behind: its creator is the only member it starts with, and the
// screens that list the topics a member takes part in would not show it.
//
// [Ja] createTopic はトピックの作成と作成者の参加を 1 つのトランザクションで行う。誰も参加して
// いないトピックが残らないようにするためである。作成直後のメンバーは作成者だけであり、メンバーが
// 参加しているトピックを並べる画面にはそのトピックが出てこなくなる。
func (uc *CreateTopicUsecase) createTopic(
	ctx context.Context,
	space *model.Space,
	spaceMember *model.SpaceMember,
	input CreateTopicInput,
	visibility model.TopicVisibility,
) (*CreateTopicOutput, error) {
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// The space is locked before the next number is read. topics(space_id, number) is unique, so
	// two topics created at the same time would otherwise take the number the other one is about
	// to insert, and the later insert would fail.
	//
	// [Ja] 次の番号を読む前にスペースをロックする。topics(space_id, number) は一意であり、ロック
	// しなければ同時に作られた 2 つのトピックが互いに相手の入れようとしている番号を取り、後から
	// INSERT した側が失敗する。
	locked, err := uc.spaceRepo.WithTx(tx).LockByID(ctx, space.ID)
	if err != nil {
		return nil, fmt.Errorf("スペースのロックに失敗: %w", err)
	}
	if !locked {
		return nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	topicRepo := uc.topicRepo.WithTx(tx)
	topicMemberRepo := uc.topicMemberRepo.WithTx(tx)

	number, err := topicRepo.NextTopicNumber(ctx, space.ID)
	if err != nil {
		return nil, fmt.Errorf("次のトピック番号の取得に失敗: %w", err)
	}

	topic, err := topicRepo.Create(ctx, repository.CreateTopicInput{
		SpaceID:     space.ID,
		Number:      number,
		Name:        input.Name,
		Description: input.Description,
		Visibility:  visibility,
	})
	if err != nil {
		return nil, fmt.Errorf("トピックの作成に失敗: %w", err)
	}

	if _, err := topicMemberRepo.Create(ctx, repository.CreateTopicMemberInput{
		SpaceID:       space.ID,
		TopicID:       topic.ID,
		SpaceMemberID: spaceMember.ID,
	}); err != nil {
		return nil, fmt.Errorf("トピックメンバーの作成に失敗: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗: %w", err)
	}

	return &CreateTopicOutput{Space: space, Topic: topic}, nil
}
