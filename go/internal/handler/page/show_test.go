package page_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

// TestShow pins the visibility rules of the page detail screen at the HTTP boundary: a trashed page
// is 404 for everyone but a member holding page:trash, who gets it with the trash alert. The
// usecase covers the same rules per branch (get_page_show_test.go); this test fixes the status
// codes and what is actually rendered.
//
// [Ja] TestShow はページ表示画面の可視性ルールを HTTP 境界で固定する。ゴミ箱に入ったページは
// page:trash を持つメンバー以外には 404 で、当該メンバーにはゴミ箱アラート付きで返る。同じ
// ルールは UseCase 側でも分岐ごとに検証しており (get_page_show_test.go)、本テストは
// ステータスコードと実際に描画される内容を固定する。
func TestShow(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	// Member holding page:trash (can open the trash, so a trashed page stays readable).
	//
	// [Ja] page:trash を持つメンバー (ゴミ箱を開けるため、ゴミ箱のページも閲覧できる)。
	trashUserID := testutil.NewUserBuilder(t, tx).
		WithEmail("page-show-trash@example.com").
		WithAtname("pageshowtrash").
		Build()
	// Read-only member (page:read alone must not reveal a trashed page).
	//
	// [Ja] 読み取り専用メンバー (page:read だけではゴミ箱のページは見えてはならない)。
	readerUserID := testutil.NewUserBuilder(t, tx).
		WithEmail("page-show-reader@example.com").
		WithAtname("pageshowreader").
		Build()

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("page-show-space").
		WithName("Page Show Space").
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(trashUserID).
		WithScopes([]model.Scope{model.ScopePageTrash}).
		Build()
	testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(readerUserID).
		WithScopes([]model.Scope{model.ScopePageRead}).
		Build()

	publicTopicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("Public Topic").
		WithVisibility(int32(model.TopicVisibilityPublic)).
		Build()
	privateTopicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(2).
		WithName("Private Topic").
		WithVisibility(int32(model.TopicVisibilityPrivate)).
		Build()

	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(1).
		WithTitle("Public Page Title").
		WithBodyHTML("<p>public page body</p>").
		WithLinkedPageIDs([]model.PageID{}).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(privateTopicID).
		WithNumber(2).
		WithTitle("Private Page Title").
		WithLinkedPageIDs([]model.PageID{}).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(3).
		WithTitle("Trashed Page Title").
		WithBodyHTML("<p>trashed page body</p>").
		WithLinkedPageIDs([]model.PageID{}).
		WithTrashed().
		Build()
	// A page created but never published: no title and no body. It is the case where the generated
	// meta description is empty and the site-wide default has to survive.
	//
	// [Ja] 作成しただけで一度も公開していないページ (タイトルも本文も無い)。生成する meta description
	// が空になり、サイト共通の既定値が残らなければならないケースにあたる。
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(4).
		WithNilTitle().
		WithBodyHTML("").
		WithLinkedPageIDs([]model.PageID{}).
		Build()

	h := setupHandler(t, queries)

	tests := []struct {
		name string
		// spaceIdentifier overrides the requested space identifier. Empty means the stored one.
		//
		// [Ja] spaceIdentifier はリクエストするスペース識別子を上書きする。空なら保存済みの識別子。
		spaceIdentifier string
		pageNumber      string
		userID          *model.UserID
		wantStatus      int
		wantContains    []string
		wantNotContains []string
	}{
		{
			name:       "ゲストは公開トピックのページを閲覧できる",
			pageNumber: "1",
			wantStatus: http.StatusOK,
			wantContains: []string{
				"Public Page Title",
				"<p>public page body</p>",
				"Public Topic",
				"/s/page-show-space/topics/1",
				"aria-current=\"page\"",
				"<title>Public Page Title | Page Show Space</title>",
				`<meta property="og:title" content="Public Page Title | Page Show Space">`,
				`<meta property="og:url" content="https://localhost/s/page-show-space/pages/1">`,
				`<link rel="canonical" href="https://localhost/s/page-show-space/pages/1">`,
				`<meta name="description" content="public page body">`,
				// The page detail is the one long-form content screen, so it overrides the site-wide
				// website type.
				//
				// [Ja] ページ表示画面は唯一の本文ページのため、サイト共通の website 型を上書きする。
				`<meta property="og:type" content="article">`,
				"<script type=\"application/ld+json\">",
				"\"@type\":\"BreadcrumbList\"",
				"\"position\":1,\"name\":\"Page Show Space\",\"item\":\"https://localhost/s/page-show-space\"",
				"\"position\":2,\"name\":\"Public Topic\",\"item\":\"https://localhost/s/page-show-space/topics/1\"",
				"\"position\":3,\"name\":\"Public Page Title\"",
			},
			wantNotContains: []string{
				"このページはゴミ箱に入れられています。",
				"\"name\":\"Public Page Title\",\"item\":",
				`<meta property="og:url" content="">`,
				`<link rel="canonical" href="">`,
				`href="/home"`,
				`https://localhost/home`,
			},
		},
		{
			// spaces.identifier is citext, so this request reaches the same page. The canonical URL
			// must still point at the stored identifier so that the casings do not split the page
			// into several self-declared canonical addresses.
			//
			// [Ja] spaces.identifier は citext のため、このリクエストも同じページに到達する。正規 URL は
			// 保存済みの識別子を指し続け、大文字小文字の違いでページが複数の自己参照 canonical に
			// 分かれないようにする。
			name:            "識別子の大文字小文字が異なっても canonical は保存済みの識別子を指す",
			spaceIdentifier: "PAGE-SHOW-SPACE",
			pageNumber:      "1",
			wantStatus:      http.StatusOK,
			wantContains: []string{
				`<meta property="og:url" content="https://localhost/s/page-show-space/pages/1">`,
				`<link rel="canonical" href="https://localhost/s/page-show-space/pages/1">`,
			},
			wantNotContains: []string{"PAGE-SHOW-SPACE"},
		},
		{
			// The page number in the canonical URL comes from the stored page, not from the URL
			// parameter, so that /pages/001 does not become an address of its own.
			//
			// [Ja] 正規 URL のページ番号は URL パラメータではなく保存済みのページから取る。
			// /pages/001 が独自のアドレスにならないようにするためである。
			name:       "ゼロ埋めのページ番号でも canonical は正規化された番号を指す",
			pageNumber: "001",
			wantStatus: http.StatusOK,
			wantContains: []string{
				`<meta property="og:url" content="https://localhost/s/page-show-space/pages/1">`,
				`<link rel="canonical" href="https://localhost/s/page-show-space/pages/1">`,
			},
			wantNotContains: []string{"/pages/001"},
		},
		{
			name:            "ゲストは非公開トピックのページを閲覧できない",
			pageNumber:      "2",
			wantStatus:      http.StatusNotFound,
			wantNotContains: []string{"Private Page Title"},
		},
		{
			name:            "ゲストはゴミ箱のページを閲覧できない",
			pageNumber:      "3",
			wantStatus:      http.StatusNotFound,
			wantNotContains: []string{"Trashed Page Title", "<p>trashed page body</p>"},
		},
		{
			name:            "page:trash を持たないメンバーはゴミ箱のページを閲覧できない",
			pageNumber:      "3",
			userID:          &readerUserID,
			wantStatus:      http.StatusNotFound,
			wantNotContains: []string{"Trashed Page Title", "<p>trashed page body</p>"},
		},
		{
			name:       "page:trash を持つメンバーはゴミ箱のページをアラート付きで閲覧できる",
			pageNumber: "3",
			userID:     &trashUserID,
			wantStatus: http.StatusOK,
			wantContains: []string{
				"Trashed Page Title",
				"<p>trashed page body</p>",
				"このページはゴミ箱に入れられています。",
				"ゴミ箱を見る",
				"/s/page-show-space/trash",
				`href="/home"`,
				"\"position\":1,\"name\":\"ホーム\",\"item\":\"https://localhost/home\"",
			},
		},
		{
			// This page has no body, so the generated description is empty and the site-wide default
			// must survive. Pinning it here keeps the handler from declaring an empty description.
			//
			// [Ja] このページは本文が無いため生成される説明文は空になり、サイト共通の既定値が
			// 残らなければならない。ここで固定することで、Handler が空の description を出す形に
			// 変わったときに検出できる。
			name:       "タイトル未設定のページは無題と表示され既定の説明文を保つ",
			pageNumber: "4",
			wantStatus: http.StatusOK,
			wantContains: []string{
				"無題",
				"<title>無題 | Page Show Space</title>",
				`<meta name="description" content="Wikinoはオンラインで情報を共有・整理できるWikiアプリケーションです。">`,
			},
		},
		{
			name:       "存在しないページ番号は 404",
			pageNumber: "999",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "数値でないページ番号は 404",
			pageNumber: "abc",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spaceIdentifier := tt.spaceIdentifier
			if spaceIdentifier == "" {
				spaceIdentifier = "page-show-space"
			}

			req := newRequestWithChiParams(t, http.MethodGet, "/s/"+spaceIdentifier+"/pages/"+tt.pageNumber, map[string]string{
				"space_identifier": spaceIdentifier,
				"page_number":      tt.pageNumber,
			})

			ctx := i18n.SetLocale(req.Context(), i18n.LangJa)
			if tt.userID != nil {
				ctx = middleware.SetUserToContext(ctx, &model.User{ID: *tt.userID})
			}
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			h.Show(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status code = %d, want %d", rr.Code, tt.wantStatus)
			}

			body := rr.Body.String()
			for _, want := range tt.wantContains {
				if !strings.Contains(body, want) {
					t.Errorf("response does not contain %q", want)
				}
			}
			for _, notWant := range tt.wantNotContains {
				if strings.Contains(body, notWant) {
					t.Errorf("response unexpectedly contains %q", notWant)
				}
			}
		})
	}
}
