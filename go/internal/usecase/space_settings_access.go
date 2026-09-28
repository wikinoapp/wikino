package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// SpaceSettingsItemsは、スペース設定のトップに出す項目のうち閲覧者が開けるものを表す。
// 設定のトップ自体を開けるか (CanShow) と、スペース画面のメニューに設定へのリンクを出すかは
// この値から決めるため、2つの画面が誰を通すかについて別々の答えを出すことはない。
type SpaceSettingsItems struct {
	// CanUpdateSpaceは既存の項目 (一般・エクスポート・添付ファイル・削除) を出すかを表す (space:write)。
	CanUpdateSpace bool

	// CanShowPersonalAccessTokensは個人アクセストークンの項目を出すかを表す。
	// personal_access_token:readを持ち、公開APIのフィーチャーフラグが有効なときにtrueになる。
	CanShowPersonalAccessTokens bool

	// CanShowOAuthGrantsは連携中のアプリの項目を出すかを表す。
	// oauth_grant:readを持ち、公開APIのフィーチャーフラグが有効なときにtrueになる。
	CanShowOAuthGrants bool

	// CanShowOAuthApplicationsはOAuthアプリの項目を出すかを表す。
	// oauth_application:readを持ち、公開APIのフィーチャーフラグが有効なときにtrueになる。
	CanShowOAuthApplications bool
}

// CanShowは閲覧者がスペース設定のトップを開けるか (出す項目が1つでもあるか) を返す。
func (i SpaceSettingsItems) CanShow() bool {
	return i.CanUpdateSpace || i.CanShowPersonalAccessTokens || i.CanShowOAuthGrants || i.CanShowOAuthApplications
}

// resolveSpaceSettingsItemsは、スペースメンバーが開けるスペース設定の項目を解決する。
// メンバーでない (spaceMemberがnil) 場合は、どの項目も開けない。
//
// フィーチャーフラグは、トークン管理かOAuthアプリのスコープを持つメンバーのときだけ引く。
// フラグが効くのは公開APIの項目だけで、既存の項目はフラグにかかわらず出すためである。
func resolveSpaceSettingsItems(
	ctx context.Context,
	featureFlagRepo *repository.FeatureFlagRepository,
	spaceMember *model.SpaceMember,
) (SpaceSettingsItems, error) {
	if spaceMember == nil {
		return SpaceSettingsItems{}, nil
	}

	authorizer := newAuthorizer(spaceMember, nil)
	items := SpaceSettingsItems{CanUpdateSpace: authorizer.CanUpdateSpace()}

	canShowPAT := authorizer.CanShowPersonalAccessTokens()
	canShowOAuthGrants := authorizer.CanShowOAuthGrants()
	canShowOAuthApplications := authorizer.CanShowOAuthApplications()
	if !canShowPAT && !canShowOAuthGrants && !canShowOAuthApplications {
		return items, nil
	}

	enabled, err := featureFlagRepo.IsEnabled(ctx, spaceMember.UserID, model.FeatureFlagPublicAPI)
	if err != nil {
		return SpaceSettingsItems{}, fmt.Errorf("フィーチャーフラグの判定に失敗: %w", err)
	}
	items.CanShowPersonalAccessTokens = enabled && canShowPAT
	items.CanShowOAuthGrants = enabled && canShowOAuthGrants
	items.CanShowOAuthApplications = enabled && canShowOAuthApplications

	return items, nil
}
