package space_feed_test

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/handler/space_feed"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

const testDomain = "wikino.example.com"

// fixtureはテストデータと、そのデータを使うルーター
type fixture struct {
	router          http.Handler
	path            string
	publicPageID    model.PageID
	memberUserID    model.UserID
	spaceIdentifier string
}

// newFixtureはserve.goと同じパターンでルートを登録したルーターを返す。
// スペース識別子は一意制約があり、並行するテストのトランザクションで重ならないよう
// テストごとに変える。
func newFixture(t *testing.T, spaceIdentifier string) fixture {
	t.Helper()

	_, tx := testutil.SetupTx(t)
	q := testutil.QueriesWithTx(tx)
	cfg := &config.Config{Domain: testDomain}
	uc := usecase.NewGetSpaceFeedUsecase(
		cfg,
		repository.NewSpaceRepository(q),
		repository.NewPageRepository(q),
		repository.NewTopicRepository(q),
		repository.NewAttachmentRepository(q),
	)
	h := space_feed.NewHandler(cfg, uc)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier(spaceIdentifier).
		WithName("フィードのスペース").
		Build()
	memberUserID := testutil.NewUserBuilder(t, tx).
		WithEmail(spaceIdentifier + "@example.com").
		WithAtname(strings.ReplaceAll(spaceIdentifier, "-", "_")).
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(memberUserID).
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

	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	publicPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(1).
		WithTitle("公開ページ").
		WithBody("公開ページの本文 <b>太字</b>").
		WithPublishedAt(baseTime).
		WithModifiedAt(baseTime.Add(time.Hour)).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(privateTopicID).
		WithNumber(2).
		WithTitle("非公開ページ").
		WithBody("非公開ページの本文").
		WithPublishedAt(baseTime.Add(2 * time.Hour)).
		Build()

	r := chi.NewRouter()
	r.Use(middleware.NewCSRF(&config.Config{}).Middleware)
	r.Use(session.NewFlashManager("", false, false).Middleware)
	r.Get("/s/{space_identifier}/atom", h.Show)
	r.Head("/s/{space_identifier}/atom", h.Show)

	return fixture{
		router:          r,
		path:            "/s/" + spaceIdentifier + "/atom",
		publicPageID:    publicPageID,
		memberUserID:    memberUserID,
		spaceIdentifier: spaceIdentifier,
	}
}

func serve(router http.Handler, req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

// parsedFeedは、応答のXMLをAtomとして読むための構造体。
// 要素の数を数えるため、1つだけ出るべき要素もスライスで受ける
type parsedFeed struct {
	XMLName xml.Name `xml:"http://www.w3.org/2005/Atom feed"`
	ID      string   `xml:"id"`
	Updated []string `xml:"updated"`
	Authors []struct {
		Name string `xml:"name"`
	} `xml:"author"`
	Entries []struct {
		ID        string   `xml:"id"`
		Title     string   `xml:"title"`
		Published []string `xml:"published"`
		Updated   []string `xml:"updated"`
		Content   struct {
			Type string `xml:"type,attr"`
			Body string `xml:",chardata"`
		} `xml:"content"`
	} `xml:"entry"`
}

func TestShow_OK(t *testing.T) {
	t.Parallel()

	f := newFixture(t, "sfeed-ok")
	rr := serve(f.router, httptest.NewRequest(http.MethodGet, f.path, nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	if got, want := rr.Header().Get("Content-Type"), "application/atom+xml; charset=utf-8"; got != want {
		t.Errorf("Content-Type = %q、期待値 = %q", got, want)
	}
	if got := rr.Header().Get("Cache-Control"); got != "no-cache" {
		t.Errorf("Cache-Control = %q、期待値 = %q", got, "no-cache")
	}
	if got := rr.Header().Get("ETag"); !strings.HasPrefix(got, `"`) || !strings.HasSuffix(got, `"`) || len(got) < 3 {
		t.Errorf("ETag = %q、引用符で囲んだ強いETagを期待", got)
	}
	if got, want := rr.Header().Get("Content-Length"), strconv.Itoa(rr.Body.Len()); got != want {
		t.Errorf("Content-Length = %q、期待値 = %q", got, want)
	}
	if got := rr.Header().Get("Set-Cookie"); got != "" {
		t.Errorf("Set-Cookie = %q、期待値 = 空", got)
	}
	if !strings.HasPrefix(rr.Body.String(), xml.Header) {
		t.Errorf("本文がXML宣言で始まっていない: %s", rr.Body.String())
	}

	var feed parsedFeed
	if err := xml.Unmarshal(rr.Body.Bytes(), &feed); err != nil {
		t.Fatalf("XMLとしての読み込みのエラー = %v", err)
	}
	// フィードのidはRails版と同じ形にする (購読し直さずに済むよう)
	if want := "tag:" + testDomain + ",2025:/s/" + f.spaceIdentifier + "/atom"; feed.ID != want {
		t.Errorf("フィードのid = %q、期待値 = %q", feed.ID, want)
	}
	if len(feed.Updated) != 1 {
		t.Errorf("フィードのupdatedの数 = %d、期待値 = 1", len(feed.Updated))
	}
	if len(feed.Authors) != 1 || feed.Authors[0].Name != "フィードのスペース" {
		t.Errorf("フィードのauthor = %+v、名前がスペース名の1件を期待", feed.Authors)
	}

	if len(feed.Entries) != 1 {
		t.Fatalf("エントリーの数 = %d、期待値 = 1 (公開トピックのページだけ)", len(feed.Entries))
	}
	entry := feed.Entries[0]
	// エントリーのidはRails版と同じ形にする (既読のエントリーが新着として出直さないよう)
	if want := "tag:Wikino,2025:Page/" + string(f.publicPageID); entry.ID != want {
		t.Errorf("エントリーのid = %q、期待値 = %q", entry.ID, want)
	}
	if entry.Title != "公開ページ" {
		t.Errorf("エントリーのtitle = %q、期待値 = %q", entry.Title, "公開ページ")
	}
	if len(entry.Published) != 1 || entry.Published[0] != "2025-01-01T00:00:00Z" {
		t.Errorf("エントリーのpublished = %v、[2025-01-01T00:00:00Z] を期待", entry.Published)
	}
	if len(entry.Updated) != 1 || entry.Updated[0] != "2025-01-01T01:00:00Z" {
		t.Errorf("エントリーのupdated = %v、[2025-01-01T01:00:00Z] を期待", entry.Updated)
	}
	if entry.Content.Type != "html" {
		t.Errorf("contentのtype = %q、期待値 = %q", entry.Content.Type, "html")
	}
	if !strings.Contains(entry.Content.Body, "<b>太字</b>") {
		t.Errorf("contentの本文 = %q、描画したHTMLを期待", entry.Content.Body)
	}
}

func TestShow_Head(t *testing.T) {
	t.Parallel()

	f := newFixture(t, "sfeed-head")
	get := serve(f.router, httptest.NewRequest(http.MethodGet, f.path, nil))
	rr := serve(f.router, httptest.NewRequest(http.MethodHead, f.path, nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	if got, want := rr.Header().Get("ETag"), get.Header().Get("ETag"); got != want {
		t.Errorf("ETag = %q、GETと同じ %q を期待", got, want)
	}
	if got, want := rr.Header().Get("Content-Length"), strconv.Itoa(get.Body.Len()); got != want {
		t.Errorf("Content-Length = %q、GETの本文の長さ %q を期待", got, want)
	}
	if rr.Body.Len() != 0 {
		t.Errorf("本文の長さ = %d、期待値 = 0", rr.Body.Len())
	}
}

func TestShow_NotModified(t *testing.T) {
	t.Parallel()

	f := newFixture(t, "sfeed-304")
	etag := serve(f.router, httptest.NewRequest(http.MethodGet, f.path, nil)).Header().Get("ETag")

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
			req := httptest.NewRequest(http.MethodGet, f.path, nil)
			req.Header.Set("If-None-Match", tt.ifNoneMatch)
			rr := serve(f.router, req)

			if rr.Code != tt.wantStatus {
				t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, tt.wantStatus)
			}
			if got := rr.Header().Get("ETag"); got != etag {
				t.Errorf("ETag = %q、期待値 = %q", got, etag)
			}
			if got := rr.Header().Get("Cache-Control"); got != "no-cache" {
				t.Errorf("Cache-Control = %q、期待値 = %q", got, "no-cache")
			}
			if tt.wantStatus == http.StatusNotModified && rr.Body.Len() != 0 {
				t.Errorf("本文の長さ = %d、期待値 = 0", rr.Body.Len())
			}
		})
	}
}

func TestShow_NotFound(t *testing.T) {
	t.Parallel()

	f := newFixture(t, "sfeed-404")
	rr := serve(f.router, httptest.NewRequest(http.MethodGet, "/s/sfeed-missing/atom", nil))

	if rr.Code != http.StatusNotFound {
		t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusNotFound)
	}
	if got, want := rr.Header().Get("Cache-Control"), "private, no-store"; got != want {
		t.Errorf("Cache-Control = %q、期待値 = %q", got, want)
	}
}

func TestShow_NoCookie(t *testing.T) {
	t.Parallel()

	f := newFixture(t, "sfeed-cookie")
	req := httptest.NewRequest(http.MethodGet, f.path, nil)
	// 壊れたフラッシュのCookieを送っても、消すためのSet-Cookieを返さない
	req.AddCookie(&http.Cookie{Name: session.FlashCookieName, Value: "invalid"})
	rr := serve(f.router, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("Set-Cookie"); got != "" {
		t.Errorf("Set-Cookie = %q、期待値 = 空", got)
	}
}

func TestShow_SignedInMember(t *testing.T) {
	t.Parallel()

	f := newFixture(t, "sfeed-member")
	anonymous := serve(f.router, httptest.NewRequest(http.MethodGet, f.path, nil))

	req := httptest.NewRequest(http.MethodGet, f.path, nil)
	req = req.WithContext(middleware.SetUserToContext(req.Context(), &model.User{ID: f.memberUserID}))
	rr := serve(f.router, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}
	if strings.Contains(rr.Body.String(), "非公開ページ") {
		t.Errorf("ログインしているメンバーへのフィードに非公開トピックのページが載っている: %s", rr.Body.String())
	}
	if rr.Body.String() != anonymous.Body.String() {
		t.Error("ログインしているメンバーへのフィードが、未ログインのときと異なる")
	}
}
