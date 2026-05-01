package attachment_og_image

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Show は GET /attachments/:id/og_image - 公開トピックの og:image 用リサイズ画像へリダイレクトする
//
// 処理の流れ:
//  1. URL から attachment ID を取得
//  2. UseCase で「公開トピックのページから参照されている添付」の blob 情報を取得
//     (visibility 検証は Repository の SQL に統合済み)
//  3. OgImageBuilder で imgproxy 署名付き URL を生成
//  4. 302 redirect で imgproxy へ
//  5. Cache-Control: public, max-age=60, s-maxage=300 を付与
//
// 404 を返すケース (添付の存在を秘匿するため、UseCase レベルでは Forbidden と NotFound を区別しない):
//   - 不正な UUID
//   - 存在しない attachment_id
//   - 公開トピック以外のページから参照されている添付
//   - どのページからも参照されていない添付
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// imgproxy 設定の整合性は DB アクセス前に確認する
	if h.ogImageBuilder == nil {
		slog.ErrorContext(ctx, "OgImageBuilder が初期化されていません (WIKINO_IMGPROXY_URL / WIKINO_R2_BUCKET_NAME を確認してください)")
		writeServerError(w)
		return
	}

	attachmentID := model.AttachmentID(chi.URLParam(r, "attachment_id"))

	output, err := h.getAttachmentOgImageUC.Execute(ctx, usecase.GetAttachmentOgImageInput{
		AttachmentID: attachmentID,
	})
	if err != nil {
		var ae *model.AppError
		if errors.As(err, &ae) {
			if ae.Code == model.AppErrCodeResourceNotFound {
				writeNotFound(w, r)
				return
			}
			slog.ErrorContext(ctx, ae.LogString())
			writeServerError(w)
			return
		}
		slog.ErrorContext(ctx, "公開 og:image 用 blob 情報取得に失敗", "error", err, "attachment_id", string(attachmentID))
		writeServerError(w)
		return
	}

	imgproxyURL, err := h.ogImageBuilder.BuildOgImageURL(output.Attachment.BlobKey, time.Now())
	if err != nil {
		slog.ErrorContext(ctx, "imgproxy URL の生成に失敗", "error", err, "attachment_id", string(attachmentID))
		writeServerError(w)
		return
	}

	// CDN とブラウザにキャッシュを許可するが、visibility 変更時の leak window を
	// 短く保つため max-age / s-maxage を抑え目に設定する。
	w.Header().Set("Cache-Control", "public, max-age=60, s-maxage=300")
	http.Redirect(w, r, imgproxyURL, http.StatusFound)
}

// writeNotFound は 404 レスポンスを書き込む際に Cache-Control: private, no-store を付与する。
//
// 公開トピック → 非公開トピックへの移動などで一時的に 404 になった添付ファイルが、
// 再度公開トピックへ戻った際に CDN キャッシュ済みの 404 を返し続ける「逆方向 leak」を
// 防ぐため、404 はキャッシュ禁止にする。`Cache-Control` は `WriteHeader` 前にセットする
// 必要があるため、`handler.NotFound` を呼ぶ前に Header をセットしている。
func writeNotFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	handler.NotFound(w, r)
}

// writeServerError は 500 レスポンスを書き込む際に Cache-Control: private, no-store を付与する。
//
// 設定不備や一時的な障害でエラーになっているレスポンスを CDN にキャッシュさせないため、
// `http.Error` の前に Header をセットする (`http.Error` は内部で `WriteHeader` を呼ぶ)。
func writeServerError(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "private, no-store")
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}
