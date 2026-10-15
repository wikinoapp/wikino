package usecase

import (
	"context"
	"fmt"
	"slices"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/policy"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// apiDraftPageAccessReposは公開APIで下書きを解決するのに必要なリポジトリ群
type apiDraftPageAccessRepos struct {
	pageRepo        *repository.PageRepository
	draftPageRepo   *repository.DraftPageRepository
	topicRepo       *repository.TopicRepository
	topicMemberRepo *repository.TopicMemberRepository
}

// apiDraftPageAccessは公開APIでページ番号から解決した、トークンの持ち主の下書きと判定に使うデータ
type apiDraftPageAccess struct {
	// draftはトークンの持ち主の下書き。Pageに下書きのページを持つ
	draft *model.DraftPage
	// topicは下書きが属するトピック
	topic      *model.Topic
	authorizer policy.Authorizer
}

// findAPIDraftPageは、トークンが属するスペースのページについて、トークンの持ち主の下書きを取得する。
// 他のメンバーの下書きは所有者をクエリで限定して返さない (存在を秘匿するため未存在として扱う)。
// ページや下書きが存在しない場合に加え、ゴミ箱のページの下書き・廃棄済みのトピックの下書き・
// 閲覧者が開けない非公開トピックの下書きも未存在として扱う。
// 下書きを読む・保存する・削除するといった操作の判定は、返したauthorizerで呼び出し元が行う
func findAPIDraftPage(
	ctx context.Context,
	repos apiDraftPageAccessRepos,
	principal *model.APIPrincipal,
	space *model.Space,
	pageNumber int32,
) (*apiDraftPageAccess, error) {
	notFound := &model.AppError{
		Code:    model.AppErrCodeResourceNotFound,
		UserMsg: i18n.T(ctx, "error_not_found_message"),
	}

	spaceMember := principal.SpaceMember
	if spaceMember == nil {
		return nil, notFound
	}

	page, err := repos.pageRepo.FindBySpaceAndNumber(ctx, space.ID, model.PageNumber(pageNumber))
	if err != nil {
		return nil, fmt.Errorf("ページの取得に失敗: %w", err)
	}
	if !isAPIVisiblePage(page) {
		return nil, notFound
	}

	draft, err := repos.draftPageRepo.FindByPageAndMember(ctx, page.ID, spaceMember.ID, space.ID)
	if err != nil {
		return nil, fmt.Errorf("下書きの取得に失敗: %w", err)
	}
	if draft == nil {
		return nil, notFound
	}

	// 一覧と同じく、トピックは下書きの所属トピックで判定する
	topic, err := repos.topicRepo.FindBySpaceAndID(ctx, space.ID, draft.TopicID)
	if err != nil {
		return nil, fmt.Errorf("トピックの取得に失敗: %w", err)
	}
	if topic == nil {
		return nil, notFound
	}

	topicMember, err := repos.topicMemberRepo.FindBySpaceMemberAndTopic(ctx, space.ID, spaceMember.ID, topic.ID)
	if err != nil {
		return nil, fmt.Errorf("トピックメンバーの取得に失敗: %w", err)
	}

	authorizer := newAPIAuthorizer(principal, topicMember)
	if !authorizer.CanShowTopic(topic) {
		return nil, notFound
	}

	draft.Page = page
	return &apiDraftPageAccess{draft: draft, topic: topic, authorizer: authorizer}, nil
}

// apiDraftPageMatchesは下書きの今の版が、`If-Match` で前提にした版と一致するかを返す
func apiDraftPageMatches(draft *model.DraftPage, ifMatchDigests []string, ifMatchAny bool) bool {
	return ifMatchAny || slices.Contains(ifMatchDigests, draft.ContentDigest())
}

// apiDraftPageModifiedErrorは、下書きが `If-Match` の版の後に更新されていたことを表すエラーを返す
func apiDraftPageModifiedError(ctx context.Context) error {
	return &model.AppError{
		Code:    model.AppErrCodePreconditionFailed,
		UserMsg: i18n.T(ctx, "api_error_draft_page_modified"),
	}
}
