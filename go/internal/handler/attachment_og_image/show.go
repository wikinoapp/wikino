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

// ShowはGET /attachments/:id/og_image - 公開トピックのog:image用リサイズ画像へリダイレクトする
//
// 処理の流れ:
//  1. URLからattachment IDを取得
//  2. UseCaseで「公開トピックのページから参照されている添付」のblob情報を取得
//     (visibility検証はRepositoryのSQLに統合済み)
//  3. OgImageBuilderでimgproxy署名付きURLを生成
//  4. 302 redirectでimgproxyへ
//  5. Cache-Control: public, max-age=60, s-maxage=300を付与
//
// 404を返すケース (添付の存在を秘匿するため、UseCaseレベルではForbiddenとNotFoundを区別しない):
//   - 不正なUUID
//   - 存在しないattachment_id
//   - 公開トピック以外のページから参照されている添付
//   - どのページからも参照されていない添付
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// imgproxy設定の整合性はDBアクセス前に確認する
	if h.ogImageBuilder == nil {
		slog.ErrorContext(ctx, "OgImageBuilderが初期化されていません (WIKINO_IMGPROXY_URL / WIKINO_R2_BUCKET_NAMEを確認してください)")
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
		slog.ErrorContext(ctx, "公開og:image用blob情報取得に失敗", "error", err, "attachment_id", string(attachmentID))
		writeServerError(w)
		return
	}

	imgproxyURL, err := h.ogImageBuilder.BuildOgImageURL(output.Attachment.BlobKey, time.Now())
	if err != nil {
		slog.ErrorContext(ctx, "imgproxy URLの生成に失敗", "error", err, "attachment_id", string(attachmentID))
		writeServerError(w)
		return
	}

	// CDNとブラウザにキャッシュを許可するが、visibility変更時のleak windowを
	// 短く保つためmax-age / s-maxageを抑え目に設定する。
	w.Header().Set("Cache-Control", "public, max-age=60, s-maxage=300")
	http.Redirect(w, r, imgproxyURL, http.StatusFound)
}

// writeNotFoundは404レスポンスを書き込む際にCache-Control: private, no-storeを付与する。
//
// 公開トピック → 非公開トピックへの移動などで一時的に404になった添付ファイルが、
// 再度公開トピックへ戻った際にCDNキャッシュ済みの404を返し続ける「逆方向leak」を
// 防ぐため、404はキャッシュ禁止にする。`Cache-Control` は `WriteHeader` 前にセットする
// 必要があるため、`handler.NotFound` を呼ぶ前にHeaderをセットしている。
func writeNotFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	handler.NotFound(w, r)
}

// writeServerErrorは500レスポンスを書き込む際にCache-Control: private, no-storeを付与する。
//
// 設定不備や一時的な障害でエラーになっているレスポンスをCDNにキャッシュさせないため、
// `http.Error` の前にHeaderをセットする (`http.Error` は内部で `WriteHeader` を呼ぶ)。
func writeServerError(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "private, no-store")
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}
