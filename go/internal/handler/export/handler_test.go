package export_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/handler"
	"github.com/wikinoapp/wikino/go/internal/handler/export_download"
	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/storage"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// exportRouter mirrors the export namespace as cmd/wikino/serve.go registers it, so that the way
// the routes answer a method can be checked. The requests are served through a router rather than
// by calling the handlers directly, because what is under test is the resolution of a method to a
// route, which only the router does.
//
// [Ja] exportRouter は cmd/wikino/serve.go が登録するとおりにエクスポートの名前空間を組み立てる。
// ルートがメソッドにどう答えるかを確かめるためである。ハンドラーを直接呼ばずルーター越しに
// リクエストを流すのは、メソッドからルートへの解決というルーターだけが行うことが対象だからである。
func exportRouter(t *testing.T, queries *query.Queries, userID model.UserID) *chi.Mux {
	t.Helper()

	exportHandler := setupHandler(t, queries)
	downloadHandler := export_download.NewHandler(usecase.NewGetExportDownloadUsecase(
		repository.NewSpaceRepository(queries),
		repository.NewSpaceMemberRepository(queries),
		repository.NewExportRepository(queries),
		storage.NewFakeObjectStorage(),
	))

	r := chi.NewRouter()
	r.NotFound(handler.NotFound)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := i18n.SetLocale(req.Context(), i18n.LangJa)
			ctx = middleware.SetUserToContext(ctx, &model.User{ID: userID, Atname: "export-user"})
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
	r.Route("/s/{space_identifier}/settings/exports", func(r chi.Router) {
		r.MethodNotAllowed(handler.NotFound)

		r.Post("/", exportHandler.Create)
		r.Get("/new", exportHandler.New)
		r.Head("/new", exportHandler.New)

		exportPath := "/{export_id:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}}"
		r.Get(exportPath, exportHandler.Show)
		r.Head(exportPath, exportHandler.Show)
		r.Get(exportPath+"/download", downloadHandler.Show)
		r.Head(exportPath+"/download", downloadHandler.Show)
	})

	return r
}

// TestExportRoutes_HeadAnswersLikeGet pins HEAD to the same answer GET gives. The reverse proxy
// hands the export namespace to Go for every method, and the Rails router these screens came from
// reads a HEAD that matches nothing as a GET, so a screen that answers 200 to GET has to answer
// 200 to HEAD as well.
//
// [Ja] TestExportRoutes_HeadAnswersLikeGet は、HEAD が GET と同じ答えを返すことを固定する。
// リバースプロキシはエクスポートの名前空間を全メソッドで Go へ渡し、これらの画面の移行元である
// Rails のルーターは一致しない HEAD を GET として読むため、GET に 200 を返す画面は HEAD にも
// 200 を返す必要がある。
func TestExportRoutes_HeadAnswersLikeGet(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID, spaceID, spaceMemberID := exportSpace(t, tx, "exp-route-head", nil)
	exportID := testutil.NewExportBuilder(t, tx).
		WithSpaceID(spaceID).
		WithQueuedByID(spaceMemberID).
		WithStatus(model.ExportStatusSucceeded).
		WithStatusChangedAt(time.Now()).
		WithObjectKey("exports/exp-route-head/archive.zip").
		Build()

	server := httptest.NewServer(exportRouter(t, queries, userID))
	t.Cleanup(server.Close)

	// The download answers with a redirect to the object storage, which the client must not follow.
	//
	// [Ja] ダウンロードはオブジェクトストレージへのリダイレクトで答えるため、クライアントに
	// たどらせない。
	client := server.Client()
	client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }

	base := "/s/exp-route-head/settings/exports"
	tests := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{name: "開始画面", path: base + "/new", wantStatus: http.StatusOK},
		{name: "状態表示画面", path: base + "/" + exportID.String(), wantStatus: http.StatusOK},
		{name: "ダウンロード", path: base + "/" + exportID.String() + "/download", wantStatus: http.StatusSeeOther},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequestWithContext(t.Context(), http.MethodHead, server.URL+tt.path, nil)
			if err != nil {
				t.Fatalf("request creation failed: %v", err)
			}

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("response body read failed: %v", err)
			}
			if err := resp.Body.Close(); err != nil {
				t.Fatalf("response body close failed: %v", err)
			}

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status code = %d, want %d", resp.StatusCode, tt.wantStatus)
			}
			if len(body) != 0 {
				t.Errorf("response body = %q, want empty", body)
			}
		})
	}
}

// TestExportRoutes_RejectsUnsupportedRequestsWithNotFound covers what the namespace answers to the
// requests it does not serve. The reverse proxy sends every method under the namespace here so that
// a start cannot reach the Rails writer that has no guard against a second export, which leaves the
// router to reject the rest the way the Rails version did: with a 404 page, not chi's bodiless 405.
//
// [Ja] TestExportRoutes_RejectsUnsupportedRequestsWithNotFound は、名前空間が受け持たない
// リクエストに何を返すかを対象とする。リバースプロキシは、2 つ目のエクスポートを防ぐ仕組みを
// 持たない Rails の書き込み処理へ開始が届かないよう、配下の全メソッドをここへ送る。残りを
// Rails 版と同じように拒否するのはルーターの役目で、chi の本文なし 405 ではなく 404 ページを返す。
func TestExportRoutes_RejectsUnsupportedRequestsWithNotFound(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID, _, _ := exportSpace(t, tx, "exp-route-reject", nil)

	server := httptest.NewServer(exportRouter(t, queries, userID))
	t.Cleanup(server.Close)

	base := "/s/exp-route-reject/settings/exports"
	tests := []struct {
		name   string
		method string
		path   string
	}{
		{name: "一覧のGET", method: http.MethodGet, path: base},
		{name: "一覧のDELETE", method: http.MethodDelete, path: base},
		{name: "開始画面へのPOST", method: http.MethodPost, path: base + "/new"},
		{name: "format拡張子付き", method: http.MethodGet, path: base + ".json"},
		{name: "不正なエクスポートID", method: http.MethodGet, path: base + "/not-a-uuid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequestWithContext(t.Context(), tt.method, server.URL+tt.path, nil)
			if err != nil {
				t.Fatalf("request creation failed: %v", err)
			}

			resp, err := server.Client().Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("response body read failed: %v", err)
			}
			if err := resp.Body.Close(); err != nil {
				t.Fatalf("response body close failed: %v", err)
			}

			if resp.StatusCode != http.StatusNotFound {
				t.Errorf("status code = %d, want %d", resp.StatusCode, http.StatusNotFound)
			}
			if len(body) == 0 {
				t.Error("404ページの本文が返っていない")
			}
		})
	}
}

// TestExportRoutes_StartResolvesOnTheBarePath keeps the start reachable at the path the form posts
// to. The namespace is a sub-router, so the start is registered on "/" inside it, and only the
// router can tell whether that still answers the path without a trailing slash.
//
// [Ja] TestExportRoutes_StartResolvesOnTheBarePath は、フォームの送信先のパスで開始に届くことを
// 守る。名前空間はサブルーターなので開始はその中の "/" に登録されており、末尾スラッシュ無しの
// パスに今も答えるかを知れるのはルーターだけである。
func TestExportRoutes_StartResolvesOnTheBarePath(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID, _, _ := exportSpace(t, tx, "exp-route-start", nil)
	router := exportRouter(t, queries, userID)

	for _, path := range []string{
		"/s/exp-route-start/settings/exports",
		"/s/exp-route-start/settings/exports/",
	} {
		if !router.Match(chi.NewRouteContext(), http.MethodPost, path) {
			t.Errorf("POST %s がエクスポート開始のルートに解決されていない", path)
		}
	}
}
