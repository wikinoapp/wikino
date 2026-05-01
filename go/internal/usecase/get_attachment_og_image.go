package usecase

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// GetAttachmentOgImageUsecase は公開 og:image 配信エンドポイント用の読み取り UseCase
//
// 「公開トピックのページから参照されている添付ファイル」のみ blob 情報を返す。
// visibility 判定は Repository (`FindPubliclyReferencedBlobByID`) の SQL に統合されており、
// 呼び出し側で検証を忘れる構造的事故が発生しない。非公開・存在しない・不正 UUID の
// いずれも `*model.AppError{Code: AppErrCodeResourceNotFound}` を返し、Handler 側では
// 一律で 404 にレンダリングして「公開でない添付」と「存在しない添付」をレスポンス上は
// 区別しない (添付の存在を秘匿するため)。
type GetAttachmentOgImageUsecase struct {
	attachmentRepo *repository.AttachmentRepository
}

// NewGetAttachmentOgImageUsecase は GetAttachmentOgImageUsecase を生成する
func NewGetAttachmentOgImageUsecase(attachmentRepo *repository.AttachmentRepository) *GetAttachmentOgImageUsecase {
	return &GetAttachmentOgImageUsecase{attachmentRepo: attachmentRepo}
}

// GetAttachmentOgImageInput は UseCase の入力
type GetAttachmentOgImageInput struct {
	AttachmentID model.AttachmentID
}

// GetAttachmentOgImageOutput は UseCase の出力
//
// Attachment.BlobKey / ContentType / SpaceID は populate 済みだが、Filename は空のまま
// (FindPubliclyReferencedBlobByID では取得していない)。og:image 配信用途では Filename を
// 使わないため問題ない。
type GetAttachmentOgImageOutput struct {
	Attachment *model.Attachment
}

// Execute は公開 og:image として配信可能な添付ファイルの blob 情報を取得する
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
