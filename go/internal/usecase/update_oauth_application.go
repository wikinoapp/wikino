package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// UpdateOAuthApplicationUsecaseは、スペースのOAuthアプリの名前とリダイレクトURIを更新する。
type UpdateOAuthApplicationUsecase struct {
	spaceRepo            *repository.SpaceRepository
	spaceMemberRepo      *repository.SpaceMemberRepository
	featureFlagRepo      *repository.FeatureFlagRepository
	oauthApplicationRepo *repository.OAuthApplicationRepository
	updateValidator      *validator.OAuthApplicationUpdateValidator
}

// NewUpdateOAuthApplicationUsecaseはUpdateOAuthApplicationUsecaseを生成する
func NewUpdateOAuthApplicationUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	featureFlagRepo *repository.FeatureFlagRepository,
	oauthApplicationRepo *repository.OAuthApplicationRepository,
	updateValidator *validator.OAuthApplicationUpdateValidator,
) *UpdateOAuthApplicationUsecase {
	return &UpdateOAuthApplicationUsecase{
		spaceRepo:            spaceRepo,
		spaceMemberRepo:      spaceMemberRepo,
		featureFlagRepo:      featureFlagRepo,
		oauthApplicationRepo: oauthApplicationRepo,
		updateValidator:      updateValidator,
	}
}

// UpdateOAuthApplicationInputは更新の入力パラメータ。
// RedirectURIsはフォームが送信した文字列で、変換はバリデーターが行う。
type UpdateOAuthApplicationInput struct {
	SpaceIdentifier    model.SpaceIdentifier
	UserID             model.UserID
	OAuthApplicationID model.OAuthApplicationID
	ExpectedVersion    int64
	Name               string
	RedirectURIs       string
}

// UpdateOAuthApplicationOutputは更新したアプリを保持する
type UpdateOAuthApplicationOutput struct {
	Space       *model.Space
	Application *model.OAuthApplication
}

// ExecuteはOAuthアプリの名前とリダイレクトURIを更新する。
//
// 作成したメンバーかどうかは問わない。アプリはスペースが持ち、oauth_application:writeは
// スペースのアプリ全体を扱う権限であるため。クライアントの種別は変えない。種別を変えると
// シークレットの発行や破棄が伴い、連携中のクライアントの認証方法が変わってしまうためである。
// 発行済みの許可とトークンはそのまま使える。
func (uc *UpdateOAuthApplicationUsecase) Execute(ctx context.Context, input UpdateOAuthApplicationInput) (*UpdateOAuthApplicationOutput, error) {
	// 1. データ取得と認可チェック
	access, err := fetchPublicAPISettingsAccess(
		ctx, uc.spaceRepo, uc.spaceMemberRepo, uc.featureFlagRepo, input.SpaceIdentifier, input.UserID,
		policy.Authorizer.CanUpdateOAuthApplication,
	)
	if err != nil {
		return nil, err
	}

	notFound := &model.AppError{Code: model.AppErrCodeResourceNotFound, UserMsg: i18n.T(ctx, "error_not_found_message")}

	app, err := uc.oauthApplicationRepo.FindByIDAndSpaceID(ctx, input.OAuthApplicationID, access.space.ID)
	if err != nil {
		return nil, fmt.Errorf("OAuthアプリの取得に失敗: %w", err)
	}
	if app == nil {
		return nil, notFound
	}

	// 2. バリデーション
	validated, err := uc.updateValidator.Validate(ctx, validator.OAuthApplicationUpdateValidatorInput{
		Name:         input.Name,
		RedirectURIs: input.RedirectURIs,
	})
	if err != nil {
		return nil, err
	}

	// 3. 永続化
	updated, err := uc.oauthApplicationRepo.Update(ctx, repository.UpdateOAuthApplicationInput{
		ID:              app.ID,
		SpaceID:         access.space.ID,
		ExpectedVersion: input.ExpectedVersion,
		Name:            input.Name,
		RedirectURIs:    validated.RedirectURIs,
	})
	if err != nil {
		return nil, fmt.Errorf("OAuthアプリの更新に失敗: %w", err)
	}
	if updated == nil {
		// UPDATEの条件には版も含む。行が無ければ削除と競合を区別する。
		current, err := uc.oauthApplicationRepo.FindByIDAndSpaceID(ctx, app.ID, access.space.ID)
		if err != nil {
			return nil, fmt.Errorf("OAuthアプリの再取得に失敗: %w", err)
		}
		if current == nil {
			return nil, notFound
		}
		return nil, &model.AppError{Code: model.AppErrCodeConflict, UserMsg: i18n.T(ctx, "oauth_application_edit_conflict")}
	}

	return &UpdateOAuthApplicationOutput{
		Space:       access.space,
		Application: updated,
	}, nil
}
