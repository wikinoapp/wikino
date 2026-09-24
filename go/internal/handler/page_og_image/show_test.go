package page_og_image_test

import (
	"bytes"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/page_og_image"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/ogcard"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

const (
	wantPublicCacheControl  = "public, max-age=60, s-maxage=300"
	wantPrivateCacheControl = "private, no-store"
)

// fixtureはテストデータを作成したトランザクションと、そのデータを使うルーター
type fixture struct {
	router http.Handler
	// canonicalPathは公開ページ (番号1) のカード画像の正規URL
	canonicalPath string
	// pathは指定したページ番号とバージョンのカード画像のパス
	path func(pageNumber string, version string) string
}

// newFixtureはserve.goと同じパターンでルートを登録したルーターを返す。
// パスの末尾の `{version}.png` からバージョンを取り出すところまで含めて確かめるため、
// ハンドラーを直接呼ばずにchiのルーターを通す。
// スペース識別子は一意制約があり、並行するテストのトランザクションで重ならないよう
// テストごとに変える。
func newFixture(t *testing.T, spaceIdentifier string) fixture {
	t.Helper()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	uc := usecase.NewGetPageOgImageUsecase(
		repository.NewSpaceRepository(q),
		repository.NewPageRepository(q),
		repository.NewTopicRepository(q),
	)
	renderer, err := ogcard.NewRenderer()
	if err != nil {
		t.Fatalf("ogcard.NewRenderer()のエラー = %v", err)
	}
	h := page_og_image.NewHandler(renderer, uc)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier(spaceIdentifier).
		WithName("OGスペース").
		Build()
	publicTopicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("公開トピック").
		WithVisibility(int32(model.TopicVisibilityPublic)).
		Build()
	privateTopicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(2).
		WithName("非公開トピック").
		WithVisibility(int32(model.TopicVisibilityPrivate)).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(1).
		WithTitle("公開ページ").
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(privateTopicID).
		WithNumber(2).
		WithTitle("非公開ページ").
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(3).
		WithTitle("ゴミ箱のページ").
		WithTrashed().
		Build()

	r := chi.NewRouter()
	r.Use(middleware.NewCSRF(&config.Config{}).Middleware)
	r.Use(session.NewFlashManager("", false, false).Middleware)
	r.Get("/s/{space_identifier}/pages/{page_number}/og_image/{version}.png", h.Show)
	r.Head("/s/{space_identifier}/pages/{page_number}/og_image/{version}.png", h.Show)

	path := func(pageNumber string, version string) string {
		return "/s/" + spaceIdentifier + "/pages/" + pageNumber + "/og_image/" + version + ".png"
	}
	return fixture{
		router:        r,
		canonicalPath: path("1", publicPageVersion),
		path:          path,
	}
}

// publicPageVersionは公開ページ (番号1) のカードのバージョン
var publicPageVersion = ogcard.Card{SpaceName: "OGスペース", TopicName: "公開トピック", PageTitle: "公開ページ"}.Version()

func serve(router http.Handler, method, target string, header http.Header) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, nil)
	for k, vs := range header {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func TestShow_OK(t *testing.T) {
	t.Parallel()

	f := newFixture(t, "pogi-ok")
	rr := serve(f.router, http.MethodGet, f.canonicalPath, nil)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("Content-Type"); got != "image/png" {
		t.Errorf("Content-Type = %q、期待値 = %q", got, "image/png")
	}
	if got := rr.Header().Get("Cache-Control"); got != wantPublicCacheControl {
		t.Errorf("Cache-Control = %q、期待値 = %q", got, wantPublicCacheControl)
	}
	if got, want := rr.Header().Get("ETag"), `"`+publicPageVersion+`"`; got != want {
		t.Errorf("ETag = %q、期待値 = %q", got, want)
	}
	if got, want := rr.Header().Get("Content-Length"), strconv.Itoa(rr.Body.Len()); got != want {
		t.Errorf("Content-Length = %q、期待値 = %q", got, want)
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(rr.Body.Bytes()))
	if err != nil {
		t.Fatalf("PNGとしてのデコードのエラー = %v", err)
	}
	if cfg.Width != ogcard.Width || cfg.Height != ogcard.Height {
		t.Errorf("画像の寸法 = %d×%d、期待値 = %d×%d", cfg.Width, cfg.Height, ogcard.Width, ogcard.Height)
	}
}

func TestShow_Head(t *testing.T) {
	t.Parallel()

	f := newFixture(t, "pogi-head")
	rr := serve(f.router, http.MethodHead, f.canonicalPath, nil)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("Content-Type"); got != "image/png" {
		t.Errorf("Content-Type = %q、期待値 = %q", got, "image/png")
	}
	if got := rr.Header().Get("ETag"); got == "" {
		t.Error("ETagが空")
	}
	if got := rr.Header().Get("Content-Length"); got == "" || got == "0" {
		t.Errorf("Content-Length = %q、GETと同じ本文の長さを期待", got)
	}
	if rr.Body.Len() != 0 {
		t.Errorf("本文の長さ = %d、期待値 = 0", rr.Body.Len())
	}
}

func TestShow_NotModified(t *testing.T) {
	t.Parallel()

	f := newFixture(t, "pogi-304")
	etag := `"` + publicPageVersion + `"`

	tests := []struct {
		name        string
		ifNoneMatch string
		wantStatus  int
	}{
		{name: "ETagが一致すれば304を返す", ifNoneMatch: etag, wantStatus: http.StatusNotModified},
		{name: "弱いETagでも一致すれば304を返す", ifNoneMatch: "W/" + etag, wantStatus: http.StatusNotModified},
		{name: "列挙の中に一致するETagがあれば304を返す", ifNoneMatch: `"other", ` + etag, wantStatus: http.StatusNotModified},
		{name: "*なら304を返す", ifNoneMatch: "*", wantStatus: http.StatusNotModified},
		{name: "ETagが一致しなければ200を返す", ifNoneMatch: `"other"`, wantStatus: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := serve(f.router, http.MethodGet, f.canonicalPath, http.Header{"If-None-Match": {tt.ifNoneMatch}})

			if rr.Code != tt.wantStatus {
				t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, tt.wantStatus)
			}
			if got := rr.Header().Get("Cache-Control"); got != wantPublicCacheControl {
				t.Errorf("Cache-Control = %q、期待値 = %q", got, wantPublicCacheControl)
			}
			if got := rr.Header().Get("ETag"); got != etag {
				t.Errorf("ETag = %q、期待値 = %q", got, etag)
			}
			if tt.wantStatus == http.StatusNotModified && rr.Body.Len() != 0 {
				t.Errorf("本文の長さ = %d、期待値 = 0", rr.Body.Len())
			}
		})
	}
}

func TestShow_RedirectToCanonical(t *testing.T) {
	t.Parallel()

	f := newFixture(t, "pogi-302")

	tests := []struct {
		name   string
		target string
	}{
		{name: "バージョンが違う", target: f.path("1", "0000000000000000")},
		{name: "クエリが付いている", target: f.canonicalPath + "?v=1"},
		{name: "空のクエリが付いている", target: f.canonicalPath + "?"},
		{name: "スペース識別子の大文字小文字が違う", target: "/s/POGI-302/pages/1/og_image/" + publicPageVersion + ".png"},
		{name: "ページ番号に先頭の0が付いている", target: f.path("01", publicPageVersion)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := serve(f.router, http.MethodGet, tt.target, nil)

			if rr.Code != http.StatusFound {
				t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusFound)
			}
			if got := rr.Header().Get("Location"); got != f.canonicalPath {
				t.Errorf("Location = %q、期待値 = %q", got, f.canonicalPath)
			}
			if got := rr.Header().Get("Cache-Control"); got != wantPublicCacheControl {
				t.Errorf("Cache-Control = %q、期待値 = %q", got, wantPublicCacheControl)
			}
			if got := rr.Header().Get("Content-Type"); got == "image/png" {
				t.Error("転送のレスポンスで画像を描画している")
			}
		})
	}
}

func TestShow_NotFound(t *testing.T) {
	t.Parallel()

	f := newFixture(t, "pogi-404")

	tests := []struct {
		name   string
		target string
	}{
		{name: "非公開トピックのページ", target: f.path("2", publicPageVersion)},
		{name: "ゴミ箱に入ったページ", target: f.path("3", publicPageVersion)},
		{name: "存在しないページ", target: f.path("999", publicPageVersion)},
		{name: "存在しないスペース", target: "/s/pogi-missing/pages/1/og_image/" + publicPageVersion + ".png"},
		{name: "数値でないページ番号", target: f.path("abc", publicPageVersion)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := serve(f.router, http.MethodGet, tt.target, nil)

			if rr.Code != http.StatusNotFound {
				t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusNotFound)
			}
			if got := rr.Header().Get("Cache-Control"); got != wantPrivateCacheControl {
				t.Errorf("Cache-Control = %q、期待値 = %q", got, wantPrivateCacheControl)
			}
			if got := rr.Header().Get("Location"); got != "" {
				t.Errorf("Location = %q、期待値 = 空", got)
			}
		})
	}
}

func TestShow_FlashCookieDoesNotSetCookie(t *testing.T) {
	t.Parallel()

	f := newFixture(t, "pogi-flash")
	tests := []struct {
		name        string
		method      string
		target      string
		ifNoneMatch string
		wantStatus  int
	}{
		{name: "200", method: http.MethodGet, target: f.canonicalPath, wantStatus: http.StatusOK},
		{name: "HEAD", method: http.MethodHead, target: f.canonicalPath, wantStatus: http.StatusOK},
		{name: "302", method: http.MethodGet, target: f.path("1", "old"), wantStatus: http.StatusFound},
		{name: "304", method: http.MethodGet, target: f.canonicalPath, ifNoneMatch: `"` + publicPageVersion + `"`, wantStatus: http.StatusNotModified},
		{name: "404", method: http.MethodGet, target: f.path("999", publicPageVersion), wantStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.target, nil)
			req.AddCookie(&http.Cookie{Name: session.FlashCookieName, Value: "invalid"})
			if tt.ifNoneMatch != "" {
				req.Header.Set("If-None-Match", tt.ifNoneMatch)
			}
			rr := httptest.NewRecorder()
			f.router.ServeHTTP(rr, req)
			if rr.Code != tt.wantStatus {
				t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, tt.wantStatus)
			}
			if got := rr.Header().Get("Set-Cookie"); got != "" {
				t.Errorf("Set-Cookie = %q、期待値 = 空", got)
			}
		})
	}
}
