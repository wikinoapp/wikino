package page

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/templates"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// New creates a page and sends the visitor to its edit screen
// (GET /s/{space_identifier}/topics/{topic_number}/pages/new). The optional title and body query
// parameters are stored as a draft, so an external entry point such as a bookmarklet opens the
// editor with those values already filled in.
//
// [Ja] New はページを作成し、その編集画面へ遷移させます
// (GET /s/{space_identifier}/topics/{topic_number}/pages/new)。任意の title / body クエリ
// パラメータは下書きとして保存するため、ブックマークレットのような外部の入口からも、値が入った
// 状態でエディタが開きます。
func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	spaceIdentifier := model.SpaceIdentifier(chi.URLParam(r, "space_identifier"))
	topicNumberStr := chi.URLParam(r, "topic_number")

	topicNumber, err := strconv.ParseInt(topicNumberStr, 10, 32)
	if err != nil {
		handler.NotFound(w, r)
		return
	}

	// The title and body parameter names end up embedded in bookmarklets distributed outside the
	// app, which makes them a public contract: keep them as they are. Their values are passed on
	// untouched, because trimming or normalizing them here would silently change what the visitor
	// finds in the editor; an over-long title is reported by the existing validation when the page
	// is published.
	//
	// [Ja] title / body というパラメータ名はアプリの外に配布されるブックマークレットに埋め込まれる
	// ため公開の契約であり、変えない。値は手を加えずに渡す。ここで切り詰めや正規化を行うと、
	// 閲覧者がエディタで目にする内容を暗黙に変えてしまうためで、長すぎるタイトルは公開時の既存の
	// バリデーションが指摘する。
	queryParams := r.URL.Query()

	output, err := h.createPageUC.Execute(ctx, usecase.CreatePageInput{
		SpaceIdentifier: spaceIdentifier,
		TopicNumber:     int32(topicNumber),
		UserID:          user.ID,
		Title:           queryParams.Get("title"),
		Body:            queryParams.Get("body"),
	})
	if err != nil {
		if ae := model.AsAppError(err); ae != nil {
			switch ae.Code {
			case model.AppErrCodeResourceNotFound, model.AppErrCodeForbidden:
				handler.NotFound(w, r)
			default:
				slog.ErrorContext(ctx, ae.LogString())
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}
		slog.ErrorContext(ctx, "ページの作成に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Each visit creates a different page, so the transfer is a temporary one (302), as in the
	// Rails version.
	//
	// [Ja] 訪問のたびに別のページが作られるため、転送は一時的なもの (302) とする。Rails 版と同じ。
	editPath := templates.PageEditPath(viewmodel.NewSpaceIdentifier(spaceIdentifier), int32(output.Page.Number))
	http.Redirect(w, r, string(editPath), http.StatusFound)
}
