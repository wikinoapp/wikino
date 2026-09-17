// Package attachment_og_imageは公開og:image配信用エンドポイントのハンドラーを提供します。
//
// 公開トピックのページから参照されている添付ファイルのみをimgproxy経由で配信します。
// visibility検証はRepositoryのSQLクエリ (`FindPubliclyReferencedBlobByID`) に統合
// されており、Handler / UseCaseは受け取ったblobをそのまま返します。
package attachment_og_image

import (
	"github.com/wikinoapp/wikino/go/internal/image"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handlerは公開og:image配信ハンドラー
//
// ogImageBuilderがnilの場合はimgproxy設定が不完全 (WIKINO_IMGPROXY_URLまたは
// WIKINO_R2_BUCKET_NAME未設定) と判断し、リクエスト時に500を返す。設定済み環境への
// デプロイで初めてフラグONユーザーに500が露出するのを避けるため、main.go起動時に
// WARNログを出して状態を可視化している。
type Handler struct {
	ogImageBuilder         *image.OgImageBuilder
	getAttachmentOgImageUC *usecase.GetAttachmentOgImageUsecase
}

// NewHandlerは新しいHandlerを作成する
func NewHandler(
	ogImageBuilder *image.OgImageBuilder,
	getAttachmentOgImageUC *usecase.GetAttachmentOgImageUsecase,
) *Handler {
	return &Handler{
		ogImageBuilder:         ogImageBuilder,
		getAttachmentOgImageUC: getAttachmentOgImageUC,
	}
}
