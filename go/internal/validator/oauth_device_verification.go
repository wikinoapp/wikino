package validator

import (
	"context"
	"fmt"
	"time"

	"github.com/wikinoapp/wikino/go/internal/auth"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// OAuthDeviceUserCodeFieldは、利用者が入力したユーザーコードを送るフォームの項目名。
// ユーザーコードを受け付けないエラーはこの項目に付ける
const OAuthDeviceUserCodeField = "user_code"

// OAuthDeviceVerificationValidatorは、ユーザーコードを照合し、許可待ちの有効な要求を取得する。
type OAuthDeviceVerificationValidator struct {
	oauthDeviceAuthorizationRepo *repository.OAuthDeviceAuthorizationRepository
	oauthApplicationRepo         *repository.OAuthApplicationRepository
	spaceRepo                    *repository.SpaceRepository
}

// NewOAuthDeviceVerificationValidatorはOAuthDeviceVerificationValidatorを生成する。
func NewOAuthDeviceVerificationValidator(
	deviceRepo *repository.OAuthDeviceAuthorizationRepository,
	applicationRepo *repository.OAuthApplicationRepository,
	spaceRepo *repository.SpaceRepository,
) *OAuthDeviceVerificationValidator {
	return &OAuthDeviceVerificationValidator{oauthDeviceAuthorizationRepo: deviceRepo, oauthApplicationRepo: applicationRepo, spaceRepo: spaceRepo}
}

// OAuthDeviceVerificationValidatorInputはユーザーコードの照合の入力パラメータ。
type OAuthDeviceVerificationValidatorInput struct {
	UserCode string
}

// OAuthDeviceVerificationValidateOutputは検証済みの要求と、要求したアプリ・連携先のスペース。
type OAuthDeviceVerificationValidateOutput struct {
	DeviceAuthorization *model.OAuthDeviceAuthorization
	Application         *model.OAuthApplication
	Space               *model.Space
	// UserCodeは表示の形 (`BCDF-GHJK`) にそろえたユーザーコード。
	UserCode string
}

// Validateは、利用者が入力したユーザーコードから、許可か拒否を待っている
// デバイス認可の要求を引く。形式の誤り・該当なし・期限切れ・許可や拒否の済んだ要求・使えなくなった
// アプリやスペースは区別せず、同じ*model.ValidationErrorを返す。どれに当たったかを答えると、
// 総当たりで有効なコードを探す手掛かりになるため。
//
// ユーザーコードで要求を引くクエリは `space_id` の規約の例外で、以降の取得はこの要求のスペースで絞る
func (v *OAuthDeviceVerificationValidator) Validate(ctx context.Context, input OAuthDeviceVerificationValidatorInput) (*OAuthDeviceVerificationValidateOutput, error) {
	userCode, ok := auth.NormalizeOAuthUserCode(input.UserCode)
	if !ok {
		return nil, NewInvalidOAuthDeviceUserCodeError(ctx)
	}

	d, err := v.oauthDeviceAuthorizationRepo.FindByUserCodeDigest(ctx, auth.DigestOpaqueToken(userCode))
	if err != nil {
		return nil, fmt.Errorf("デバイス認可の要求の取得に失敗: %w", err)
	}
	if d == nil || d.Status != model.OAuthDeviceAuthorizationStatusPending || d.IsExpired(time.Now()) {
		return nil, NewInvalidOAuthDeviceUserCodeError(ctx)
	}

	apps, err := v.oauthApplicationRepo.ListAvailableInSpaceByIDs(ctx, []model.OAuthApplicationID{d.OAuthApplicationID}, d.SpaceID)
	if err != nil {
		return nil, fmt.Errorf("OAuthアプリの取得に失敗: %w", err)
	}
	if len(apps) == 0 {
		return nil, NewInvalidOAuthDeviceUserCodeError(ctx)
	}

	space, err := v.spaceRepo.FindByID(ctx, d.SpaceID)
	if err != nil {
		return nil, fmt.Errorf("スペースの取得に失敗: %w", err)
	}
	if space == nil {
		return nil, NewInvalidOAuthDeviceUserCodeError(ctx)
	}

	return &OAuthDeviceVerificationValidateOutput{
		DeviceAuthorization: d,
		Application:         apps[0],
		Space:               space,
		UserCode:            auth.FormatOAuthUserCode(userCode),
	}, nil
}

// NewInvalidOAuthDeviceUserCodeErrorは、入力されたユーザーコードを受け付けないことを表すエラーを返す。
// UseCaseがロック後の再検証で要求を受け付けなくなったときも、照合と区別しないよう同じエラーを使う
func NewInvalidOAuthDeviceUserCodeError(ctx context.Context) error {
	ve := model.NewValidationError()
	ve.AddField(OAuthDeviceUserCodeField, i18n.T(ctx, "validation_oauth_device_user_code_invalid"))
	return ve
}
