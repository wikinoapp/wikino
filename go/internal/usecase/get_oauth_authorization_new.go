package usecase

import (
	"context"
	"fmt"
	"slices"

	"github.com/wikinoapp/wikino/go/internal/cimd"
	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetOAuthAuthorizationNewUsecaseは、OAuthの認可要求を検証し、同意画面に示す内容を取得する。
type GetOAuthAuthorizationNewUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	topicMemberRepo *repository.TopicMemberRepository
	requests        *oauthAuthorizationRequestResolver
}

// NewGetOAuthAuthorizationNewUsecaseはGetOAuthAuthorizationNewUsecaseを生成する
func NewGetOAuthAuthorizationNewUsecase(
	cfg *config.Config,
	featureFlagRepo *repository.FeatureFlagRepository,
	oauthApplicationRepo *repository.OAuthApplicationRepository,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	cimdFetcher *cimd.Fetcher,
) *GetOAuthAuthorizationNewUsecase {
	return &GetOAuthAuthorizationNewUsecase{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		topicMemberRepo: topicMemberRepo,
		requests: &oauthAuthorizationRequestResolver{
			appURL:               cfg.AppURL(),
			featureFlagRepo:      featureFlagRepo,
			oauthApplicationRepo: oauthApplicationRepo,
			spaceRepo:            spaceRepo,
			cimdFetcher:          cimdFetcher,
		},
	}
}

// GetOAuthAuthorizationNewInputは同意画面の入力パラメータ
type GetOAuthAuthorizationNewInput struct {
	UserID model.UserID
	Params OAuthAuthorizationParams
}

// GetOAuthAuthorizationNewOutputは同意画面に示す内容
type GetOAuthAuthorizationNewOutput struct {
	Application *model.OAuthApplication
	Scopes      []model.Scope
	RedirectURI string

	// Spaceは連携するスペース。SpaceOptionsを返すとき (利用者がまだスペースを選んでいないとき) はnil
	Space *model.Space

	// SpaceSelectedは、Spaceを利用者が選んだか (選べるスペースが1つで選ぶ段を省いた場合も含む)。
	// 真なら、同意の送信に `space_identifier` としてSpaceを引き継ぐ
	SpaceSelected bool

	// SpaceOptionsは、利用者がまだ連携するスペースを選んでいないときに、同意の前に示すスペースの選択肢。
	// 空でない (または選べるスペースが無い) ときは、同意画面の代わりにスペースを選ぶ段を示す
	SpaceOptions []OAuthAuthorizationSpaceOption

	// IneffectiveScopesは、Scopesのうちログイン中のユーザーが今の権限では持たないもの。
	// 付与してもトークンでは効かないため、同意画面で注記を添える
	IneffectiveScopes []model.Scope

	// DeniedReasonは、ログイン中のユーザーが連携先のスペースでアプリを許可できない理由。
	// 空でなければ、同意画面の代わりに理由を示す
	DeniedReason OAuthAuthorizationDeniedReason
}

// NeedsSpaceSelectionは、同意の前に連携するスペースを利用者に選ばせるかを返す
func (o *GetOAuthAuthorizationNewOutput) NeedsSpaceSelection() bool {
	return o.Space == nil
}

// OAuthAuthorizationSpaceOptionは、連携するスペースを選ぶ段に示すスペース。DeniedReasonが空でなければ、
// その理由を添えて選べないように示す
type OAuthAuthorizationSpaceOption struct {
	Space        *model.Space
	DeniedReason OAuthAuthorizationDeniedReason
}

// Executeは認可要求を検証し、同意画面に示す内容を返す。要求が不正なら
// *model.OAuthAuthorizationErrorを、要求の宛先のフィーチャーフラグ (CIMDのクライアントではMCPサーバーの
// フラグも) が無効なら*model.AppErrorを返す。
//
// MCPサーバーを宛先にした、スペースに属さないクライアントの要求で、利用者がまだスペースを選んでいなければ、
// ユーザーが有効なメンバーであるスペースを選択肢として返す。選べるスペースが1つなら、選ぶ段を省いて
// そのスペースの同意画面の内容を返す
func (uc *GetOAuthAuthorizationNewUsecase) Execute(ctx context.Context, input GetOAuthAuthorizationNewInput) (*GetOAuthAuthorizationNewOutput, error) {
	// 1. データ取得と検証
	req, err := uc.requests.resolve(ctx, input.UserID, input.Params)
	if err != nil {
		return nil, err
	}

	output := &GetOAuthAuthorizationNewOutput{
		Application:   req.application,
		Space:         req.space,
		SpaceSelected: req.spaceSelected,
		Scopes:        req.scopes,
		RedirectURI:   req.redirectURI,
	}
	if req.space == nil {
		options, err := uc.spaceOptions(ctx, input.UserID)
		if err != nil {
			return nil, err
		}
		selectable := slices.DeleteFunc(slices.Clone(options), func(o OAuthAuthorizationSpaceOption) bool { return o.DeniedReason != "" })
		if len(selectable) != 1 {
			output.SpaceOptions = options
			return output, nil
		}
		output.Space = selectable[0].Space
		output.SpaceSelected = true
	}

	// 2. 認可チェック (許可できない場合も、理由を示してクライアントへ戻れるよう結果として返す)
	decision, err := decideOAuthAuthorization(ctx, uc.spaceMemberRepo, output.Space, input.UserID)
	if err != nil {
		return nil, err
	}
	output.DeniedReason = decision.denied
	if decision.denied != "" {
		return output, nil
	}

	output.IneffectiveScopes, err = ineffectiveAPITokenScopes(ctx, uc.topicMemberRepo, decision.spaceMember, req.scopes)
	if err != nil {
		return nil, err
	}
	return output, nil
}

// spaceOptionsは、ユーザーuserIDが有効なメンバーであるスペースを、連携を許可できるか (`oauth_grant:write`
// を持つか) とともに返す。並び順はユーザーがスペースに参加した日の降順
func (uc *GetOAuthAuthorizationNewUsecase) spaceOptions(ctx context.Context, userID model.UserID) ([]OAuthAuthorizationSpaceOption, error) {
	spaces, err := uc.spaceRepo.ListActiveByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("スペースの一覧の取得に失敗: %w", err)
	}
	spaceIDs := make([]model.SpaceID, len(spaces))
	for i, space := range spaces {
		spaceIDs[i] = space.ID
	}
	members, err := uc.spaceMemberRepo.ListActiveByUserAndSpaceIDs(ctx, userID, spaceIDs)
	if err != nil {
		return nil, fmt.Errorf("スペースメンバーの取得に失敗: %w", err)
	}
	memberBySpace := make(map[model.SpaceID]*model.SpaceMember, len(members))
	for _, member := range members {
		memberBySpace[member.SpaceID] = member
	}

	options := make([]OAuthAuthorizationSpaceOption, 0, len(spaces))
	for _, space := range spaces {
		member, ok := memberBySpace[space.ID]
		if !ok {
			continue
		}
		option := OAuthAuthorizationSpaceOption{Space: space}
		if !newAuthorizer(member, nil).CanCreateOAuthGrant() {
			option.DeniedReason = OAuthAuthorizationDeniedNoPermission
		}
		options = append(options, option)
	}
	return options, nil
}
