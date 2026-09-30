package api_reference

import (
	"log/slog"
	"net/http"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates/pages/api_reference"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// ShowはOpenAPI記述を描画するAPIリファレンスを表示する (GET・HEAD /api/reference/v1)。
// 個人のデータを含まないため認証せず、フィーチャーフラグでも隠さない
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitleWithoutSuffix(ctx, "api_reference_title")
	meta.Description = i18n.T(ctx, "api_reference_description")
	meta.OGURL = h.cfg.AppURL() + model.APIReferencePath

	// RedocはフッターにRedoclyのロゴをCDNから読み込む。閲覧者のアクセスを外部に送らないよう、
	// 画像の読み込みを同じオリジンとdata: URIに限る (読み込めないロゴはRedocが表示から外す)。
	// あわせて、使わないプラグインの実行と `<base>` の注入による相対URLの乗っ取りを止める。
	// Redocのインラインスタイルと検索のblob Workerだけ、用途を分けて許可する。
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; worker-src 'self' blob:; object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'")

	data := api_reference.ShowPageData{
		Meta:    meta,
		SpecURL: model.APIOpenAPIDescriptionPath,
	}
	if err := api_reference.Show(data).Render(ctx, w); err != nil {
		slog.ErrorContext(ctx, "APIリファレンスのレンダリングに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
