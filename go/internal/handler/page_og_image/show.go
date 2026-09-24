package page_og_image

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/ogcard"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// cacheControlは200・304・302に付けるCache-Control。
// `/attachments/:id/og_image` と同じ値にし、公開状態の変化が反映されるまでの時間を揃える。
// URLにバージョンを含むため、同じURLの内容が変わるのは公開状態が変わったときだけである。
const cacheControl = "public, max-age=60, s-maxage=300"

// ShowはGETまたはHEAD /s/{space_identifier}/pages/{page_number}/og_image/{version}.png -
// ページのog:image用カード画像を返す
//
// 処理の流れ:
//  1. UseCaseで、ゲストに見せてよいページ (公開トピック・ゴミ箱に入っていない・タイトルあり) を取得する
//  2. 取得した内容からバージョンを計算し、リクエストが正規URL (保存済みのスペース識別子・
//     現在のバージョン・クエリなし) でなければ正規URLへ302で転送する
//  3. If-None-MatchがバージョンのETagと一致すれば304を返す
//  4. カードを描画してPNGを返す
//
// 正規URLでないリクエストをその場で描画しないのは、任意のバージョンでCDNのキャッシュを
// 迂回して描画を繰り返させられないようにするため。302は描画を伴わず、転送先の正規URLは
// CDNにキャッシュされる。
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	spaceIdentifier := model.SpaceIdentifier(chi.URLParam(r, "space_identifier"))

	pageNumber, err := strconv.ParseInt(chi.URLParam(r, "page_number"), 10, 32)
	if err != nil {
		writeNotFound(w, r)
		return
	}

	output, err := h.getPageOgImageUC.Execute(ctx, usecase.GetPageOgImageInput{
		SpaceIdentifier: spaceIdentifier,
		PageNumber:      int32(pageNumber),
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
		slog.ErrorContext(ctx, "カード画像に描く内容の取得に失敗", "error", err, "space_identifier", string(spaceIdentifier), "page_number", pageNumber)
		writeServerError(w)
		return
	}

	card := ogcard.NewPageCard(output.Space, output.Topic, output.Page)
	version := card.Version()

	canonicalPath := string(templates.PageOGImagePath(
		viewmodel.SpaceIdentifier(output.Space.Identifier),
		viewmodel.PageNumber(output.Page.Number),
		version,
	))
	if r.URL.Path != canonicalPath || r.URL.RawQuery != "" || r.URL.ForceQuery {
		w.Header().Set("Cache-Control", cacheControl)
		http.Redirect(w, r, canonicalPath, http.StatusFound)
		return
	}

	// バージョンはカードの描画結果を決める値 (内容とデザインのリビジョン) のハッシュのため、
	// そのまま強いETagに使える
	etag := `"` + version + `"`
	if etagMatches(r.Header.Get("If-None-Match"), etag) {
		w.Header().Set("Cache-Control", cacheControl)
		w.Header().Set("ETag", etag)
		w.WriteHeader(http.StatusNotModified)
		return
	}

	png, err := h.renderer.Render(card)
	if err != nil {
		slog.ErrorContext(ctx, "カード画像の描画に失敗", "error", err, "space_identifier", string(output.Space.Identifier), "page_number", output.Page.Number)
		writeServerError(w)
		return
	}

	w.Header().Set("Cache-Control", cacheControl)
	w.Header().Set("ETag", etag)
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Length", strconv.Itoa(len(png)))
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead {
		return
	}
	if _, err := w.Write(png); err != nil {
		slog.WarnContext(ctx, "カード画像の書き込みに失敗", "error", err)
	}
}

// etagMatchesはIf-None-Matchの値がetagと一致するかを返す
//
// If-None-Matchの比較は弱い比較 (RFC 9110 13.1.2) のため、`W/` の有無は区別しない。
// 複数のETagの列挙と、任意の表現に一致する `*` にも対応する。
func etagMatches(ifNoneMatch, etag string) bool {
	if ifNoneMatch == "" {
		return false
	}
	for candidate := range strings.SplitSeq(ifNoneMatch, ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "*" || strings.TrimPrefix(candidate, "W/") == etag {
			return true
		}
	}
	return false
}

// writeNotFoundは404レスポンスを書き込む際にCache-Control: private, no-storeを付与する。
//
// 非公開トピックへ移動したページなどで一時的に404になったカード画像が、再び公開された後も
// CDNにキャッシュされた404を返し続けないよう、404はキャッシュ禁止にする。
func writeNotFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	handler.NotFound(w, r)
}

// writeServerErrorは500レスポンスを書き込む際にCache-Control: private, no-storeを付与する。
//
// 一時的な障害でエラーになっているレスポンスをCDNにキャッシュさせないため。
func writeServerError(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "private, no-store")
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}
