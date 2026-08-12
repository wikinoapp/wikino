package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetPageShowUsecase aggregates the data shown on the page detail screen
// (GET /s/:space_identifier/pages/:page_number).
//
// It is a separate UseCase from GetPageDetailUsecase, which backs the editor and bails out unless
// the user is a space member. This screen is reachable by guests, so visibility is decided from the
// topic's visibility and the page's trash state instead of from membership alone.
//
// [Ja] GetPageShowUsecase はページ表示画面 (GET /s/:space_identifier/pages/:page_number) に
// 表示するデータを集約する読み取り UseCase。
//
// 編集画面向けの GetPageDetailUsecase はスペースメンバーでなければ何も返さないため流用せず、
// 別 UseCase として実装する。本画面はゲストも到達するので、可視性はメンバーかどうかではなく
// トピックの公開設定とページのゴミ箱状態から判定する。
type GetPageShowUsecase struct {
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	pageRepo        *repository.PageRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
	attachmentRepo  *repository.AttachmentRepository
}

// NewGetPageShowUsecase creates a GetPageShowUsecase.
//
// [Ja] NewGetPageShowUsecase は GetPageShowUsecase を生成する。
func NewGetPageShowUsecase(
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	pageRepo *repository.PageRepository,
	topicRepo *repository.TopicRepository,
	topicMemberRepo *repository.TopicMemberRepository,
	attachmentRepo *repository.AttachmentRepository,
) *GetPageShowUsecase {
	return &GetPageShowUsecase{
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		pageRepo:        pageRepo,
		topicRepo:       topicRepo,
		topicMemberRepo: topicMemberRepo,
		attachmentRepo:  attachmentRepo,
	}
}

// GetPageShowInput holds the input parameters for fetching the page detail.
// UserID is nil when the user is not signed in (a public topic's page is viewable without signing in).
//
// [Ja] GetPageShowInput はページ表示画面のデータ取得の入力パラメータ。
// UserID は未ログイン時に nil になる (公開トピックのページは未ログインでも閲覧できる)。
type GetPageShowInput struct {
	SpaceIdentifier model.SpaceIdentifier
	PageNumber      int32
	UserID          *model.UserID
}

// GetPageShowOutput is the output of fetching the page detail.
//
// [Ja] GetPageShowOutput はページ表示画面のデータ取得の出力。
type GetPageShowOutput struct {
	Space       *model.Space
	SpaceMember *model.SpaceMember
	Page        *model.Page
	Topic       *model.Topic

	// IsTrashed reports whether the page is in the trash. It is only ever true for a viewer allowed
	// to see it (a member holding page:trash); everyone else gets a not-found error instead. The
	// template can therefore render the trash alert whenever this is true.
	//
	// [Ja] IsTrashed はページがゴミ箱に入っているかを表す。true になるのは閲覧を許可された閲覧者
	// (page:trash を持つメンバー) の場合だけで、それ以外には not found エラーを返す。したがって
	// テンプレートは true のときにゴミ箱アラートを出せばよい。
	IsTrashed bool

	CanUpdatePage bool

	// FeaturedImageAttachment is the page's cover image attachment, used to build the og:image meta
	// tag. It is nil when the page has no cover image, or when the repository cannot resolve the
	// attachment. Only ID / SpaceID / Filename are populated (see
	// AttachmentRepository.FindByIDAndSpace).
	//
	// [Ja] FeaturedImageAttachment は og:image メタタグの組み立てに使うページのアイキャッチ画像の
	// 添付ファイル。アイキャッチ画像を持たない場合や、Repository が添付ファイルを解決できない場合は
	// nil。populate されるのは ID / SpaceID / Filename のみ
	// (AttachmentRepository.FindByIDAndSpace を参照)。
	FeaturedImageAttachment *model.Attachment
}

// Execute fetches the data shown on the page detail screen. It returns a *model.AppError with
// AppErrCodeResourceNotFound whenever the page must not be shown to the current viewer, so that the
// handler cannot tell "hidden" and "missing" apart in the response.
//
// [Ja] Execute はページ表示画面に表示するデータを取得する。現在の閲覧者に見せてはいけない場合は
// AppErrCodeResourceNotFound の *model.AppError を返し、レスポンス上で「隠している」と
// 「存在しない」を区別しないようにする。
func (uc *GetPageShowUsecase) Execute(ctx context.Context, input GetPageShowInput) (*GetPageShowOutput, error) {
	data, err := fetchPageAccessDataAllowingGuest(ctx, uc.pageAccessRepos(), input.SpaceIdentifier, input.PageNumber, input.UserID)
	if err != nil {
		return nil, err
	}

	authorizer := newAuthorizer(data.spaceMember, data.topicMember)

	if !authorizer.CanShowTopic(data.topic) {
		return nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	// A trashed page stays readable only for members who can open the trash, so that they can decide
	// whether to restore it. Guests and members without page:trash get a 404 instead of the body.
	//
	// [Ja] ゴミ箱に入ったページは、復元の判断ができるようゴミ箱を開ける権限を持つメンバーにだけ
	// 見せる。ゲストと page:trash を持たないメンバーには本文ではなく 404 を返す。
	isTrashed := data.page.TrashedAt != nil
	if isTrashed && !authorizer.CanShowTrash() {
		return nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	featuredImageAttachment, err := uc.findFeaturedImageAttachment(ctx, data.page, data.space.ID)
	if err != nil {
		return nil, err
	}

	return &GetPageShowOutput{
		Space:                   data.space,
		SpaceMember:             data.spaceMember,
		Page:                    data.page,
		Topic:                   data.topic,
		IsTrashed:               isTrashed,
		CanUpdatePage:           authorizer.CanUpdatePage(),
		FeaturedImageAttachment: featuredImageAttachment,
	}, nil
}

func (uc *GetPageShowUsecase) pageAccessRepos() pageAccessRepos {
	return pageAccessRepos{
		spaceRepo:       uc.spaceRepo,
		spaceMemberRepo: uc.spaceMemberRepo,
		pageRepo:        uc.pageRepo,
		topicRepo:       uc.topicRepo,
		topicMemberRepo: uc.topicMemberRepo,
	}
}

// findFeaturedImageAttachment resolves the page's cover image attachment for the og:image meta tag.
// A page without a cover image, or one whose attachment the repository cannot resolve, yields nil
// rather than an error: the meta tag falls back to the default OGP image and the page itself still
// renders.
//
// [Ja] findFeaturedImageAttachment は og:image メタタグ用にページのアイキャッチ画像の添付ファイルを
// 解決する。アイキャッチ画像を持たないページや、Repository が添付ファイルを解決できないページでは
// エラーにせず nil を返す。メタタグは既定の OGP 画像にフォールバックし、ページ自体は描画できる。
func (uc *GetPageShowUsecase) findFeaturedImageAttachment(ctx context.Context, pg *model.Page, spaceID model.SpaceID) (*model.Attachment, error) {
	if pg.FeaturedImageAttachmentID == nil {
		return nil, nil
	}

	attachment, err := uc.attachmentRepo.FindByIDAndSpace(ctx, *pg.FeaturedImageAttachmentID, spaceID)
	if err != nil {
		return nil, fmt.Errorf("アイキャッチ画像の添付ファイルの取得に失敗: %w", err)
	}

	return attachment, nil
}
