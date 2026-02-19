package page_location

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/middleware"
)

// pageLocationItem はページロケーション検索結果の1件を表す
type pageLocationItem struct {
	Key string `json:"key"`
}

// pageLocationsResponse はページロケーション検索のJSONレスポンス
type pageLocationsResponse struct {
	PageLocations []pageLocationItem `json:"page_locations"`
}

// Index はページロケーションを検索します (GET /go/s/{space_identifier}/page_locations)
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 認証済みユーザーを取得
	user := middleware.UserFromContext(ctx)
	if user == nil {
		http.NotFound(w, r)
		return
	}

	// URLパラメータを取得
	spaceIdentifier := chi.URLParam(r, "space_identifier")

	// スペースを取得
	space, err := h.spaceRepo.FindByIdentifier(ctx, spaceIdentifier)
	if err != nil {
		slog.ErrorContext(ctx, "スペースの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if space == nil {
		http.NotFound(w, r)
		return
	}

	// アクティブなスペースメンバーか確認
	spaceMember, err := h.spaceMemberRepo.FindActiveBySpaceAndUser(ctx, space.ID, user.ID)
	if err != nil {
		slog.ErrorContext(ctx, "スペースメンバーの取得に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if spaceMember == nil {
		http.NotFound(w, r)
		return
	}

	// 検索キーワードを取得
	q := r.URL.Query().Get("q")

	// ページロケーションを検索
	locations, err := h.pageRepo.SearchPageLocations(ctx, space.ID, q)
	if err != nil {
		slog.ErrorContext(ctx, "ページロケーションの検索に失敗", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// レスポンスを構築
	items := make([]pageLocationItem, len(locations))
	for i, loc := range locations {
		items[i] = pageLocationItem{
			Key: fmt.Sprintf("%s/%s", loc.TopicName, loc.PageTitle),
		}
	}

	resp := pageLocationsResponse{
		PageLocations: items,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.ErrorContext(ctx, "JSONエンコードに失敗", "error", err)
	}
}
