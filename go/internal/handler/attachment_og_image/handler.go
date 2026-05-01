// Package attachment_og_image は公開 og:image 配信用エンドポイントのハンドラーを提供します。
//
// 公開トピックのページから参照されている添付ファイルのみを imgproxy 経由で配信します。
// visibility 検証は Repository の SQL クエリ (`FindPubliclyReferencedBlobByID`) に統合
// されており、Handler / UseCase は受け取った blob をそのまま返します。
package attachment_og_image

import (
	"github.com/wikinoapp/wikino/go/internal/image"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Handler は公開 og:image 配信ハンドラー
//
// ogImageBuilder が nil の場合は imgproxy 設定が不完全 (WIKINO_IMGPROXY_URL または
// WIKINO_R2_BUCKET_NAME 未設定) と判断し、リクエスト時に 500 を返す。設定済み環境への
// デプロイで初めてフラグ ON ユーザーに 500 が露出するのを避けるため、main.go 起動時に
// WARN ログを出して状態を可視化している。
type Handler struct {
	ogImageBuilder         *image.OgImageBuilder
	getAttachmentOgImageUC *usecase.GetAttachmentOgImageUsecase
}

// NewHandler は新しい Handler を作成する
func NewHandler(
	ogImageBuilder *image.OgImageBuilder,
	getAttachmentOgImageUC *usecase.GetAttachmentOgImageUsecase,
) *Handler {
	return &Handler{
		ogImageBuilder:         ogImageBuilder,
		getAttachmentOgImageUC: getAttachmentOgImageUC,
	}
}
