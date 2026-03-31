package suggestion

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// Create は編集提案を作成します (POST /s/{space_identifier}/topics/{topic_number}/suggestions)
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 認証済みユーザーを取得
	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.Redirect(w, r, "/sign_in", http.StatusFound)
		return
	}

	// URLパラメータを取得
	spaceIdentifier := model.SpaceIdentifier(chi.URLParam(r, "space_identifier"))
	topicNumberStr := chi.URLParam(r, "topic_number")

	topicNumber, err := strconv.ParseInt(topicNumberStr, 10, 32)
	if err != nil {
		handler.NotFound(w, r)
		return
	}

	// フォームデータをパース
	if err := r.ParseForm(); err != nil {
		slog.ErrorContext(ctx, "フォームのパースに失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	title := r.FormValue("title")
	body := r.FormValue("body")
	draftPageIDStrs := r.Form["draft_page_ids"]

	// 下書きページIDをドメイン型に変換
	draftPageIDs := make([]model.DraftPageID, len(draftPageIDStrs))
	for i, id := range draftPageIDStrs {
		draftPageIDs[i] = model.DraftPageID(id)
	}

	// UseCase を実行
	createOutput, err := h.createSuggestionUsecase.Execute(ctx, usecase.CreateSuggestionInput{
		SpaceIdentifier: spaceIdentifier,
		TopicNumber:     int32(topicNumber),
		UserID:          user.ID,
		Title:           title,
		Body:            body,
		DraftPageIDs:    draftPageIDs,
	})
	if err != nil {
		h.handleCreateError(w, r, err, user, spaceIdentifier, int32(topicNumber), title, body, draftPageIDStrs)
		return
	}

	// フラッシュメッセージを設定してリダイレクト
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "suggestion_create_success"))
	suggestionPath := fmt.Sprintf("/s/%s/topics/%d/suggestions/%d",
		string(spaceIdentifier),
		topicNumber,
		createOutput.Suggestion.Number,
	)
	http.Redirect(w, r, suggestionPath, http.StatusSeeOther)
}

func (h *Handler) handleCreateError(w http.ResponseWriter, r *http.Request, err error, user *model.User, spaceIdentifier model.SpaceIdentifier, topicNumber int32, title, body string, draftPageIDStrs []string) {
	ctx := r.Context()

	if ve := model.AsValidationError(err); ve != nil {
		// バリデーションエラー → フォーム再描画
		output, getErr := h.getSuggestionNewUsecase.Execute(ctx, usecase.GetSuggestionNewInput{
			SpaceIdentifier: spaceIdentifier,
			TopicNumber:     topicNumber,
			UserID:          user.ID,
		})
		if getErr != nil || output == nil {
			slog.ErrorContext(ctx, "フォーム再表示用データの取得に失敗", "error", getErr)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusUnprocessableEntity)
		h.renderNewForm(w, r, user, spaceIdentifier, output, ve, title, body, draftPageIDStrs)
		return
	}

	if ae := model.AsAppError(err); ae != nil {
		switch ae.Code {
		case model.AppErrCodeResourceNotFound:
			handler.NotFound(w, r)
		case model.AppErrCodeForbidden:
			handler.NotFound(w, r)
		default:
			slog.ErrorContext(ctx, ae.LogString())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	slog.ErrorContext(ctx, "編集提案の作成に失敗", "error", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}
