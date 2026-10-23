package usecase

import (
	"context"
	"time"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// GetOAuthDeviceConsentUsecaseは、利用者が入力したユーザーコードを照合し、デバイス認可の同意画面に
// 示す内容を取得する。
type GetOAuthDeviceConsentUsecase struct {
	featureFlagRepo       *repository.FeatureFlagRepository
	spaceMemberRepo       *repository.SpaceMemberRepository
	topicMemberRepo       *repository.TopicMemberRepository
	verificationValidator *validator.OAuthDeviceVerificationValidator
}

// NewGetOAuthDeviceConsentUsecaseはGetOAuthDeviceConsentUsecaseを生成する
func NewGetOAuthDeviceConsentUsecase(
	featureFlagRepo *repository.FeatureFlagRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	verificationValidator *validator.OAuthDeviceVerificationValidator,
) *GetOAuthDeviceConsentUsecase {
	return &GetOAuthDeviceConsentUsecase{
		featureFlagRepo:       featureFlagRepo,
		spaceMemberRepo:       spaceMemberRepo,
		topicMemberRepo:       topicMemberRepo,
		verificationValidator: verificationValidator,
	}
}

// GetOAuthDeviceConsentInputは同意画面の入力パラメータ。UserCodeは利用者が入力したユーザーコード
type GetOAuthDeviceConsentInput struct {
	UserID   model.UserID
	UserCode string
}

// GetOAuthDeviceConsentOutputはデバイス認可の同意画面に示す内容
type GetOAuthDeviceConsentOutput struct {
	Application *model.OAuthApplication
	Space       *model.Space
	Scopes      []model.Scope

	// UserCodeは表示の形にそろえたユーザーコードで、同意の送信へ引き継ぐ
	UserCode string

	// RequestedAtは要求を開始した日時。利用者が自分で開始したログインかを確かめられるよう示す
	RequestedAt time.Time

	// IneffectiveScopesは、Scopesのうちログイン中のユーザーが今の権限では持たないもの
	IneffectiveScopes []model.Scope

	// DeniedReasonは、ログイン中のユーザーが連携先のスペースでアプリを許可できない理由。
	// 空でなければ、同意の代わりに理由を示す
	DeniedReason OAuthAuthorizationDeniedReason
}

// Executeはユーザーコードを照合し、同意画面に示す内容を返す。コードを受け付けなければ
// *model.ValidationErrorを、フィーチャーフラグが無効なら*model.AppErrorを返す
func (uc *GetOAuthDeviceConsentUsecase) Execute(ctx context.Context, input GetOAuthDeviceConsentInput) (*GetOAuthDeviceConsentOutput, error) {
	// 1. データ取得と検証
	if err := checkPublicAPIEnabled(ctx, uc.featureFlagRepo, input.UserID); err != nil {
		return nil, err
	}

	v, err := uc.verificationValidator.Validate(ctx, validator.OAuthDeviceVerificationValidatorInput{UserCode: input.UserCode})
	if err != nil {
		return nil, err
	}

	// 2. 認可チェック (許可できない場合も、理由を示して拒否できるよう結果として返す)
	decision, err := decideOAuthAuthorization(ctx, uc.spaceMemberRepo, v.Space, input.UserID)
	if err != nil {
		return nil, err
	}

	output := &GetOAuthDeviceConsentOutput{
		Application:  v.Application,
		Space:        v.Space,
		Scopes:       v.DeviceAuthorization.Scopes,
		UserCode:     v.UserCode,
		RequestedAt:  v.DeviceAuthorization.CreatedAt,
		DeniedReason: decision.denied,
	}
	if decision.denied != "" {
		return output, nil
	}

	output.IneffectiveScopes, err = ineffectiveAPITokenScopes(ctx, uc.topicMemberRepo, decision.spaceMember, output.Scopes)
	if err != nil {
		return nil, err
	}
	return output, nil
}
