package usecase

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetAttachmentOgImageUsecaseは公開og:image配信エンドポイント用の読み取りUseCase
//
// 「公開トピックのページから参照されている添付ファイル」のみblob情報を返す。
// visibility判定はRepository (`FindPubliclyReferencedBlobByID`) のSQLに統合されており、
// 呼び出し側で検証を忘れる構造的事故が発生しない。非公開・存在しない・不正UUIDの
// いずれも `*model.AppError{Code: AppErrCodeResourceNotFound}` を返し、Handler側では
// 一律で404にレンダリングして「公開でない添付」と「存在しない添付」をレスポンス上は
// 区別しない (添付の存在を秘匿するため)。
type GetAttachmentOgImageUsecase struct {
	attachmentRepo *repository.AttachmentRepository
}

// NewGetAttachmentOgImageUsecaseはGetAttachmentOgImageUsecaseを生成する
func NewGetAttachmentOgImageUsecase(attachmentRepo *repository.AttachmentRepository) *GetAttachmentOgImageUsecase {
	return &GetAttachmentOgImageUsecase{attachmentRepo: attachmentRepo}
}

// GetAttachmentOgImageInputはUseCaseの入力
type GetAttachmentOgImageInput struct {
	AttachmentID model.AttachmentID
}

// GetAttachmentOgImageOutputはUseCaseの出力
//
// Attachment.BlobKey / ContentType / SpaceIDはpopulate済みだが、Filenameは空のまま
// (FindPubliclyReferencedBlobByIDでは取得していない)。og:image配信用途ではFilenameを
// 使わないため問題ない。
type GetAttachmentOgImageOutput struct {
	Attachment *model.Attachment
}

// Executeは公開og:imageとして配信可能な添付ファイルのblob情報を取得する
func (uc *GetAttachmentOgImageUsecase) Execute(ctx context.Context, input GetAttachmentOgImageInput) (*GetAttachmentOgImageOutput, error) {
	attachment, err := uc.attachmentRepo.FindPubliclyReferencedBlobByID(ctx, input.AttachmentID)
	if err != nil {
		return nil, err
	}
	if attachment == nil {
		return nil, &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}
	return &GetAttachmentOgImageOutput{Attachment: attachment}, nil
}
