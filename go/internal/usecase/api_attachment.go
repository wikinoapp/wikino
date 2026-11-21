package usecase

import (
	"context"
	"fmt"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// findViewableAPIAttachmentは、公開APIの呼び出し主体principalが見られる、スペースidentifierの添付ファイルを返す。
// APIトークンのattachment:readを必須とする。そのうえで、スペースで閲覧権限を持つか、
// 有効な参照が1件以上あり、その参照元がすべて公開トピックのページであれば閲覧を許可する。
// 添付ファイルが存在しない場合に加え、見る権限が無い場合も存在を秘匿するため未存在 (*model.AppError) として扱う
func findViewableAPIAttachment(
	ctx context.Context,
	attachmentRepo *repository.AttachmentRepository,
	principal *model.APIPrincipal,
	identifier model.SpaceIdentifier,
	attachmentID model.AttachmentID,
) (*model.Attachment, error) {
	space, err := resolveAPISpace(ctx, principal, identifier)
	if err != nil {
		return nil, err
	}

	notFound := &model.AppError{
		Code:    model.AppErrCodeResourceNotFound,
		UserMsg: i18n.T(ctx, "error_not_found_message"),
	}

	// 公開添付ファイルを許可するPolicyとは別に、APIの操作に必須のトークンスコープを確認する。
	// HTTP以外の入口から呼ばれても、トークンに許されていない操作を実行させないためである
	if !principal.HasScope(model.ScopeAttachmentRead) {
		return nil, notFound
	}

	attachment, err := attachmentRepo.FindBlobByIDAndSpace(ctx, attachmentID, space.ID)
	if err != nil {
		return nil, fmt.Errorf("添付ファイルの取得に失敗: %w", err)
	}
	if attachment == nil {
		return nil, notFound
	}

	public, err := attachmentRepo.FindPubliclyReferencedBlobByIDAndSpace(ctx, attachment.ID, space.ID)
	if err != nil {
		return nil, fmt.Errorf("添付ファイルの参照元の公開範囲の取得に失敗: %w", err)
	}
	allReferencingPagesPublic := public != nil

	// attachment:readはスペースにしか付けられず、判定でトピックを見ないため、トピックメンバーは要らない
	if !newAPIAuthorizer(principal, nil).CanShowAttachment(allReferencingPagesPublic) {
		return nil, notFound
	}
	return attachment, nil
}
