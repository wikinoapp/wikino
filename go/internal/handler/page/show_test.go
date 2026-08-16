package page_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// showCSRFToken is the token the CSRF middleware would have put in the context. Every case renders
// with it available, so that a case asserting the token is absent proves the handler left it out
// rather than that the context happened not to carry one.
//
// [Ja] showCSRFToken は CSRF ミドルウェアがコンテキストに載せるトークンにあたる。全ケースで利用
// できる状態にして描画するため、トークンが出ないことを検証するケースは「コンテキストにたまたま
// 無かった」ではなく「Handler が載せなかった」ことを表す。
const showCSRFToken = "page-show-csrf-token"

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
	// Member who may edit pages (the header's edit button is theirs alone).
	//
	// [Ja] ページを編集できるメンバー (ヘッダーの編集ボタンはこのメンバーにだけ出る)。
	editorUserID := testutil.NewUserBuilder(t, tx).
		WithEmail("page-show-editor@example.com").
		WithAtname("pageshoweditor").
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
	editorSpaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(editorUserID).
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

	// The page listed in the link list of the public page, and the page whose link puts it in the
	// backlink list. Both live in the public topic so that a guest sees the two listings.
	//
	// [Ja] 公開ページのリンク一覧に並ぶページと、そのリンクによってバックリンク一覧に並ぶページ。
	// ゲストにも 2 つの一覧が見えるよう、どちらも公開トピックに置く。
	linkedPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(5).
		WithTitle("Linked Page Title").
		WithLinkedPageIDs([]model.PageID{}).
		Build()
	publicPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(1).
		WithTitle("Public Page Title").
		WithBodyHTML("<p>public page body</p>").
		WithLinkedPageIDs([]model.PageID{linkedPageID}).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(6).
		WithTitle("Backlink Page Title").
		WithLinkedPageIDs([]model.PageID{publicPageID}).
		Build()
	for i := range int(viewmodel.PageBacklinkLimit) {
		testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(publicTopicID).
			WithNumber(model.PageNumber(100 + i)).
			WithTitle(fmt.Sprintf("Paginated Backlink Page %02d", i)).
			WithModifiedAt(time.Date(2020, time.January, 1+i, 0, 0, 0, 0, time.UTC)).
			WithLinkedPageIDs([]model.PageID{publicPageID}).
			Build()
	}
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

	// The two cover images behind the og:image tag: a still image the tag points at, and a GIF the
	// tag has to leave alone.
	//
	// [Ja] og:image タグの元になる 2 つのアイキャッチ画像。タグが指す静止画と、タグが対象外とする GIF。
	coverAttachmentID := testutil.NewAttachmentBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSpaceMemberID(editorSpaceMemberID).
		WithFilename("cover.png").
		Build()
	gifAttachmentID := testutil.NewAttachmentBuilder(t, tx).
		WithSpaceID(spaceID).
		WithSpaceMemberID(editorSpaceMemberID).
		WithFilename("animation.gif").
		WithContentType("image/gif").
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(7).
		WithTitle("Cover Image Page Title").
		WithFeaturedImageAttachmentID(coverAttachmentID).
		WithLinkedPageIDs([]model.PageID{}).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(8).
		WithTitle("GIF Cover Image Page Title").
		WithFeaturedImageAttachmentID(gifAttachmentID).
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

		// rawQuery is the related-page pagination fallback appended to the request.
		//
		// [Ja] rawQuery はリクエストに付ける関連ページのページネーションフォールバック。
		rawQuery string

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
				// The two listings under the body, rendered for a guest as well.
				//
				// [Ja] 本文の下の 2 つの一覧。ゲストにも描画される。
				"リンク",
				"Linked Page Title",
				"バックリンク",
				"Backlink Page Title",
				// The header pins itself to the top for every viewer, so the sentinel, header, and
				// spacer web/sticky-header.ts keys off are rendered for a guest as well.
				//
				// [Ja] ヘッダーはどの閲覧者に対しても上端に固定されるため、web/sticky-header.ts が
				// 手がかりにする sentinel、header、spacer はゲストにも描画される。
				"data-sticky-header-sentinel",
				"data-sticky-header",
				"data-sticky-header-spacer",
			},
			wantNotContains: []string{
				"このページはゴミ箱に入れられています。",
				// A guest may not edit, so neither the header's edit button nor the per-card edit
				// link is offered.
				//
				// [Ja] ゲストは編集できないため、ヘッダーの編集ボタンも各カードの編集リンクも出さない。
				"/s/page-show-space/pages/1/edit",
				"/s/page-show-space/pages/5/edit",
				// Neither may a guest act on the page, so the action dropdown and the CSRF token
				// its form would need stay out of the public HTML entirely.
				//
				// [Ja] ゲストはページを操作することもできないため、操作ドロップダウンも、その
				// フォームが要する CSRF トークンも公開 HTML には載せない。
				"page-actions-dropdown",
				"/s/page-show-space/pages/1/trash",
				"/s/page-show-space/pages/1/move",
				showCSRFToken,
				// The header needs the compact bar's minimum height only when it carries actions;
				// the spacer independently retains any height lost from a wrapped title.
				//
				// [Ja] ヘッダーがコンパクトなバーの最小高を必要とするのは操作領域を持つときだけ。
				// 折り返されたタイトルから失われる高さは spacer が独立して保持する。
				"min-h-[var(--app-sticky-header-height)]",
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
			// The related-list pages leave the page's own content untouched, so every combination
			// of them declares the one address that carries that content instead of an address of
			// its own.
			//
			// [Ja] 関連一覧のページはページ自身の内容を変えないため、どの組み合わせも独自のアドレスでは
			// なく、その内容を持つ 1 つのアドレスを宣言する。
			name:       "関連一覧の2ページ目は canonical と Open Graph URL をクエリ無しの URL に集約する",
			pageNumber: "1",
			rawQuery:   "backlinks_page=2",
			wantStatus: http.StatusOK,
			wantContains: []string{
				`<meta property="og:url" content="https://localhost/s/page-show-space/pages/1">`,
				`<link rel="canonical" href="https://localhost/s/page-show-space/pages/1">`,
			},
			wantNotContains: []string{
				`<meta property="og:url" content="https://localhost/s/page-show-space/pages/1?`,
				`<link rel="canonical" href="https://localhost/s/page-show-space/pages/1?`,
			},
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
			// The link preview points at the persistent og:image URL of the cover image, which stays
			// valid past the cache lifetime of this HTML (the endpoint re-checks visibility per
			// request).
			//
			// [Ja] リンクプレビューはアイキャッチ画像の永続 og:image URL を指す。この URL はこの HTML の
			// キャッシュ寿命を超えても有効なままである (エンドポイントがリクエストごとに公開判定を
			// やり直すため)。
			name:       "アイキャッチ画像を持つページは og:image に添付ファイルの永続 URL を出す",
			pageNumber: "7",
			wantStatus: http.StatusOK,
			wantContains: []string{
				fmt.Sprintf(`<meta property="og:image" content="https://localhost/attachments/%s/og_image">`, coverAttachmentID),
				fmt.Sprintf(`<meta name="twitter:image" content="https://localhost/attachments/%s/og_image">`, coverAttachmentID),
			},
			wantNotContains: []string{"/static/images/og-image.png"},
		},
		{
			// The og:image endpoint serves a still jpg, so an animated cover would be advertised as a
			// preview that has lost what the image was. Rails leaves it out as well.
			//
			// [Ja] og:image エンドポイントは静止画の jpg を配信するため、アニメーション画像を指すと画像の
			// 持ち味を失ったプレビューを宣伝することになる。Rails 版も同じく対象外にしている。
			name:       "GIF のアイキャッチ画像は既定の OGP 画像にフォールバックする",
			pageNumber: "8",
			wantStatus: http.StatusOK,
			wantContains: []string{
				`<meta property="og:image" content="https://localhost/static/images/og-image.png">`,
			},
			wantNotContains: []string{fmt.Sprintf("/attachments/%s/og_image", gifAttachmentID)},
		},
		{
			// A page without a cover image keeps the site-wide default OGP image.
			//
			// [Ja] アイキャッチ画像を持たないページはサイト共通の既定 OGP 画像を保つ。
			name:       "アイキャッチ画像を持たないページは既定の OGP 画像を出す",
			pageNumber: "1",
			wantStatus: http.StatusOK,
			wantContains: []string{
				`<meta property="og:image" content="https://localhost/static/images/og-image.png">`,
			},
			wantNotContains: []string{"/og_image"},
		},
		{
			name:       "ページを編集できるメンバーには編集ボタンとカードの編集リンクが出る",
			pageNumber: "1",
			userID:     &editorUserID,
			wantStatus: http.StatusOK,
			wantContains: []string{
				"/s/page-show-space/pages/1/edit",
				"/s/page-show-space/pages/5/edit",
			},
		},
		{
			// A space:admin member holds both scopes, so the dropdown carries both items and the
			// trash form gets the token it posts with.
			//
			// [Ja] space:admin のメンバーは両方のスコープを持つため、ドロップダウンには 2 つの項目が
			// 載り、ゴミ箱フォームには POST に使うトークンが載る。
			name:       "移動とゴミ箱の権限を持つメンバーには操作ドロップダウンの両方の項目が出る",
			pageNumber: "1",
			userID:     &editorUserID,
			wantStatus: http.StatusOK,
			wantContains: []string{
				`id="page-actions-dropdown"`,
				"/s/page-show-space/pages/1/move",
				"移動する",
				"/s/page-show-space/pages/1/trash",
				"ゴミ箱に入れる",
				"ページをゴミ箱に入れますか？",
				fmt.Sprintf(`name="csrf_token" value="%s"`, showCSRFToken),
				// A header carrying actions reserves the compact bar's minimum height. The adjacent
				// spacer handles any additional height lost from an expanded title.
				//
				// [Ja] 操作領域を持つヘッダーはコンパクトなバーの最小高を確保する。展開時タイトルから
				// 失われる追加の高さは隣接する spacer が処理する。
				"min-h-[var(--app-sticky-header-height)]",
			},
		},
		{
			// The two items ride on different scopes, so page:trash alone opens the trash item
			// without opening the move item or the edit button.
			//
			// [Ja] 2 つの項目は別々のスコープに乗るため、page:trash だけのメンバーには移動項目も
			// 編集ボタンも出ないままゴミ箱項目だけが開く。
			name:       "page:trash だけを持つメンバーにはゴミ箱項目だけが出る",
			pageNumber: "1",
			userID:     &trashUserID,
			wantStatus: http.StatusOK,
			wantContains: []string{
				`id="page-actions-dropdown"`,
				"/s/page-show-space/pages/1/trash",
				fmt.Sprintf(`name="csrf_token" value="%s"`, showCSRFToken),
			},
			wantNotContains: []string{
				"/s/page-show-space/pages/1/move",
				"/s/page-show-space/pages/1/edit",
			},
		},
		{
			// Neither item is theirs, so the dropdown is dropped along with its trigger button
			// instead of being rendered empty.
			//
			// [Ja] どちらの項目も出ないため、空のドロップダウンを描画するのではなくトリガー
			// ボタンごと落とす。
			name:       "page:read だけを持つメンバーには操作ドロップダウンが出ない",
			pageNumber: "1",
			userID:     &readerUserID,
			wantStatus: http.StatusOK,
			wantNotContains: []string{
				"page-actions-dropdown",
				showCSRFToken,
			},
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
			wantNotContains: []string{
				// The page is already in the trash, so re-posting the action would only push back
				// its purge. With no other item left for this member, the dropdown goes too.
				//
				// [Ja] ページはすでにゴミ箱にあり、再度 POST しても完全削除が先送りされるだけで
				// ある。このメンバーには他の項目も残らないため、ドロップダウンごと消える。
				"page-actions-dropdown",
				"/s/page-show-space/pages/3/trash",
			},
		},
		{
			// The move item does not follow the trash state, so a member who may edit keeps it on a
			// trashed page while the trash item is gone.
			//
			// [Ja] 移動項目はゴミ箱状態に従わないため、編集できるメンバーにはゴミ箱のページでも
			// 残る。消えるのはゴミ箱項目だけである。
			name:       "ゴミ箱のページでも編集できるメンバーには移動項目が残る",
			pageNumber: "3",
			userID:     &editorUserID,
			wantStatus: http.StatusOK,
			wantContains: []string{
				`id="page-actions-dropdown"`,
				"/s/page-show-space/pages/3/move",
			},
			wantNotContains: []string{
				"/s/page-show-space/pages/3/trash",
				"ゴミ箱に入れる",
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
		// A stale "load more" URL names a listing slice that is not there. The screen answers it the
		// same way the topic and space screens answer an out-of-range ?page, rather than rendering
		// with the listing silently missing.
		//
		// [Ja] 古い「もっと見る」URL は存在しない一覧の範囲を指す。一覧が黙って消えた画面を描画するの
		// ではなく、トピック詳細・スペース詳細の範囲外 ?page と同じ答え方をする。
		{
			name:       "最終ページより後ろの links_page は 404",
			pageNumber: "1",
			rawQuery:   "links_page=999",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "最終ページより後ろの backlinks_page は 404",
			pageNumber: "1",
			rawQuery:   "backlinks_page=999",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "リンク先ページの最終バックリンクページより後ろの linked_backlinks_page は 404",
			pageNumber: "1",
			rawQuery:   "linked_page_number=5&linked_backlinks_page=2",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "リンク一覧に載っていないカードを指す linked_page_number は 404",
			pageNumber: "1",
			rawQuery:   "linked_page_number=999&linked_backlinks_page=2",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "数値でない linked_page_number は 404",
			pageNumber: "1",
			rawQuery:   "linked_page_number=abc",
			wantStatus: http.StatusNotFound,
		},
		{
			// The first page of every listing is always in range, so the parameters spelled out at
			// their default values render the screen normally.
			//
			// [Ja] 各一覧の 1 ページ目は常に範囲内のため、既定値を明示的に書いたパラメータでも画面は
			// 通常どおり描画される。
			name:         "1 ページ目を指すフォールバックパラメータは通常どおり描画する",
			pageNumber:   "1",
			rawQuery:     "links_page=1&backlinks_page=1",
			wantStatus:   http.StatusOK,
			wantContains: []string{"Linked Page Title", "Backlink Page Title"},
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
			req.URL.RawQuery = tt.rawQuery

			ctx := i18n.SetLocale(req.Context(), i18n.LangJa)
			ctx = middleware.SetCSRFTokenToContext(ctx, showCSRFToken)
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

// TestShow_RelatedPagePagination pins the full-page fallback of the three related-page listings on
// the public screen: the requested slice of each listing is rendered, and the link each "load more"
// offers next carries the state of the listings it does not advance. Advancing the link list is the
// exception, because its nested state belongs to a card the next page replaces.
//
// [Ja] TestShow_RelatedPagePagination は公開画面における 3 種類の関連ページ一覧のフルページ
// フォールバックを固定する。各一覧は要求した範囲を描画し、各「もっと見る」が次に示すリンクは、その
// リンクが進めない一覧の状態を引き継ぐ。リンク一覧を進めるときだけは例外で、ネスト状態は次のページで
// 入れ替わるカードに従属するためである。
func TestShow_RelatedPagePagination(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("show-related-space").
		WithName("Show Related Space").
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("Public Topic").
		WithVisibility(int32(model.TopicVisibilityPublic)).
		Build()

	baseTime := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	linkedCount := 2*int(viewmodel.LinkLimit) + 1
	linkedPageIDs := make([]model.PageID, 0, linkedCount)
	for i := range linkedCount {
		linkedPageIDs = append(linkedPageIDs, testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(model.PageNumber(100+i)).
			WithTitle(fmt.Sprintf("Shown Linked Page %02d", i)).
			WithModifiedAt(baseTime.Add(time.Duration(i)*time.Hour)).
			WithLinkedPageIDs([]model.PageID{}).
			Build())
	}

	pageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Shown Page").
		WithBodyHTML("<p>shown page body</p>").
		WithLinkedPageIDs(linkedPageIDs).
		Build()

	// The card whose nested backlink list the fallback advances sits on the link list's second
	// page, which is the only page it may be advanced from.
	//
	// [Ja] フォールバックがネストしたバックリンク一覧を進めるカードはリンク一覧の 2 ページ目にある。
	// そこが、そのカードを進められる唯一のページである。
	selectedLinkedPageID := linkedPageIDs[int(viewmodel.LinkLimit)]
	selectedLinkedPageNumber := model.PageNumber(100 + int(viewmodel.LinkLimit))
	for i := range 2*int(viewmodel.BacklinkLimit) + 1 {
		testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(model.PageNumber(200 + i)).
			WithTitle(fmt.Sprintf("Shown Nested Backlink %02d", i)).
			WithModifiedAt(baseTime.Add(time.Duration(100+i) * time.Hour)).
			WithLinkedPageIDs([]model.PageID{selectedLinkedPageID}).
			Build()
	}
	for i := range 2*int(viewmodel.PageBacklinkLimit) + 1 {
		testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(model.PageNumber(300 + i)).
			WithTitle(fmt.Sprintf("Shown Page Backlink %02d", i)).
			WithModifiedAt(baseTime.Add(time.Duration(200+i) * time.Hour)).
			WithLinkedPageIDs([]model.PageID{pageID}).
			Build()
	}

	h := setupHandler(t, queries)

	rawQuery := fmt.Sprintf(
		"links_page=2&linked_page_number=%d&linked_backlinks_page=2&backlinks_page=2",
		selectedLinkedPageNumber,
	)
	req := newRequestWithChiParams(t, http.MethodGet, "/s/show-related-space/pages/1", map[string]string{
		"space_identifier": "show-related-space",
		"page_number":      "1",
	})
	req.URL.RawQuery = rawQuery
	req = req.WithContext(i18n.SetLocale(req.Context(), i18n.LangJa))

	rr := httptest.NewRecorder()
	h.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	wantContains := []string{
		// The second slice of each listing, which only the requested state can produce.
		//
		// [Ja] 各一覧の 2 番目の範囲。要求した状態でしか描画されない。
		fmt.Sprintf("Shown Linked Page %02d", viewmodel.LinkLimit),
		fmt.Sprintf("Shown Nested Backlink %02d", viewmodel.BacklinkLimit),
		fmt.Sprintf("Shown Page Backlink %02d", viewmodel.PageBacklinkLimit),
		// The full-page fallback drops the nested state because it replaces the parent slice, while
		// the htmx fragment below preserves it because the old cards remain in the DOM.
		//
		// [Ja] フルページフォールバックは親の範囲を入れ替えるためネスト状態を落とす。一方、下の
		// htmx フラグメントは古いカードが DOM に残るためネスト状態を維持する。
		`href="/s/show-related-space/pages/1?backlinks_page=2&amp;links_page=3#page-link-list-content"`,
		fmt.Sprintf(`href="/s/show-related-space/pages/1?backlinks_page=2&amp;linked_backlinks_page=3&amp;linked_page_number=%[1]d&amp;links_page=2#page-link-list-item-%[1]d"`, selectedLinkedPageNumber),
		fmt.Sprintf(`href="/s/show-related-space/pages/1?backlinks_page=3&amp;linked_backlinks_page=2&amp;linked_page_number=%d&amp;links_page=2#page-backlink-list-content"`, selectedLinkedPageNumber),
		// The fragment URLs stay on the page detail screen's saved-link source.
		//
		// [Ja] フラグメント URL はページ表示画面の保存済みリンクを使う側に留まる。
		fmt.Sprintf(`hx-get="/s/show-related-space/pages/1/link_list?backlinks_page=2&amp;context=show&amp;linked_backlinks_page=2&amp;linked_page_number=%d&amp;linked_page_parent_page=2&amp;page=3"`, selectedLinkedPageNumber),
	}
	for _, want := range wantContains {
		if !strings.Contains(body, want) {
			t.Errorf("response does not contain %q", want)
		}
	}

	// The first slice of each listing has been replaced, not appended to.
	//
	// [Ja] 各一覧の最初の範囲は、追記ではなく差し替えられている。
	for _, notWant := range []string{"Shown Linked Page 00", "Shown Nested Backlink 00", "Shown Page Backlink 00"} {
		if strings.Contains(body, notWant) {
			t.Errorf("response unexpectedly contains %q", notWant)
		}
	}
}
