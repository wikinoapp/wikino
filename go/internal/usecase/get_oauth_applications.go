package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetOAuthApplicationsUsecaseは、OAuthアプリの一覧の画面が表示するものを集める。
type GetOAuthApplicationsUsecase struct {
	spaceRepo            *repository.SpaceRepository
	spaceMemberRepo      *repository.SpaceMemberRepository
	userRepo             *repository.UserRepository
	featureFlagRepo      *repository.FeatureFlagRepository
	oauthApplicationRepo *repository.OAuthApplicationRepository
}

// NewGetOAuthApplicationsUsecaseはGetOAuthApplicationsUsecaseを生成する
func NewGetOAuthApplicationsUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	userRepo *repository.UserRepository,
	featureFlagRepo *repository.FeatureFlagRepository,
	oauthApplicationRepo *repository.OAuthApplicationRepository,
) *GetOAuthApplicationsUsecase {
	return &GetOAuthApplicationsUsecase{
		spaceRepo:            spaceRepo,
		spaceMemberRepo:      spaceMemberRepo,
		userRepo:             userRepo,
		featureFlagRepo:      featureFlagRepo,
		oauthApplicationRepo: oauthApplicationRepo,
	}
}

// GetOAuthApplicationsInputは一覧の画面を描画するための入力パラメータ
type GetOAuthApplicationsInput struct {
	SpaceIdentifier model.SpaceIdentifier
	UserID          model.UserID
}

// GetOAuthApplicationsOutputは一覧の画面に表示するデータを保持する
type GetOAuthApplicationsOutput struct {
	Space *model.Space

	// Applicationsはスペースの削除されていないアプリで、新しい順に並ぶ
	Applications []*model.OAuthApplication

	// Creatorsはアプリを作成したメンバーのユーザー。作成したメンバーがいなくなったアプリの
	// 作成者は含まない
	Creators map[model.SpaceMemberID]*model.User

	// CanCreateは閲覧者が新しいアプリを登録できるか (oauth_application:write) を表す
	CanCreate bool
}

// Executeはスペースと、そのスペースのOAuthアプリの一覧を取得する。
//
// 個人アクセストークンと違い、閲覧者が作成したものに限らずスペースのアプリをすべて出す。
// アプリはスペースが持ち、oauth_application:*はスペースのアプリ全体を扱う権限であるため。
func (uc *GetOAuthApplicationsUsecase) Execute(ctx context.Context, input GetOAuthApplicationsInput) (*GetOAuthApplicationsOutput, error) {
	access, err := fetchPublicAPISettingsAccess(
		ctx, uc.spaceRepo, uc.spaceMemberRepo, uc.featureFlagRepo, input.SpaceIdentifier, input.UserID,
		policy.Authorizer.CanShowOAuthApplications,
	)
	if err != nil {
		return nil, err
	}

	apps, err := uc.oauthApplicationRepo.ListBySpace(ctx, access.space.ID)
	if err != nil {
		return nil, fmt.Errorf("OAuthアプリの一覧の取得に失敗: %w", err)
	}

	creators, err := buildOAuthApplicationCreators(ctx, uc.spaceMemberRepo, uc.userRepo, apps, access.space.ID)
	if err != nil {
		return nil, err
	}

	return &GetOAuthApplicationsOutput{
		Space:        access.space,
		Applications: apps,
		Creators:     creators,
		CanCreate:    access.authorizer.CanCreateOAuthApplication(),
	}, nil
}

// buildOAuthApplicationCreatorsは、アプリを作成したメンバーのユーザーを引く。作成したメンバーが
// いなくなったアプリ (CreatedSpaceMemberIDがnil) は引かない。
func buildOAuthApplicationCreators(
	ctx context.Context,
	spaceMemberRepo *repository.SpaceMemberRepository,
	userRepo *repository.UserRepository,
	apps []*model.OAuthApplication,
	spaceID model.SpaceID,
) (map[model.SpaceMemberID]*model.User, error) {
	memberIDs := make([]model.SpaceMemberID, 0, len(apps))
	for _, app := range apps {
		if app.CreatedSpaceMemberID != nil {
			memberIDs = append(memberIDs, *app.CreatedSpaceMemberID)
		}
	}

	creators, err := buildUserMapBySpaceMemberIDs(ctx, spaceMemberRepo, userRepo, memberIDs, spaceID)
	if err != nil {
		return nil, fmt.Errorf("OAuthアプリを作成したメンバーの取得に失敗: %w", err)
	}
	return creators, nil
}
