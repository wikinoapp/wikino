package page_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/wikinoapp/wikino/go/internal/i18n"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// showCSRFTokenはCSRFミドルウェアがコンテキストに載せるトークンにあたる。全ケースで利用
// できる状態にして描画するため、トークンが出ないことを検証するケースは「コンテキストにたまたま
// 無かった」ではなく「Handlerが載せなかった」ことを表す。
const showCSRFToken = "page-show-csrf-token"

// TestShowはページ表示画面の可視性ルールをHTTP境界で固定する。ゴミ箱に入ったページは
// page:trashを持つメンバー以外には404で、当該メンバーにはゴミ箱アラート付きで返る。同じ
// ルールはUseCase側でも分岐ごとに検証しており (get_page_show_test.go)、本テストは
// ステータスコードと実際に描画される内容を固定する。
func TestShow(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	// page:trashを持つメンバー (ゴミ箱を開けるため、ゴミ箱のページも閲覧できる)。
	trashUserID := testutil.NewUserBuilder(t, tx).
		WithEmail("page-show-trash@example.com").
		WithAtname("pageshowtrash").
		Build()
	// 読み取り専用メンバー (page:readだけではゴミ箱のページは見えてはならない)。
	readerUserID := testutil.NewUserBuilder(t, tx).
		WithEmail("page-show-reader@example.com").
		WithAtname("pageshowreader").
		Build()
	// ページを編集できるメンバー (ヘッダーの編集ボタンはこのメンバーにだけ出る)。
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

	// 公開ページのリンク一覧に並ぶページと、そのリンクによってバックリンク一覧に並ぶページ。
	// ゲストにも2つの一覧が見えるよう、どちらも公開トピックに置く。
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
	// 作成しただけで一度も公開していないページ (タイトルも本文も無い)。生成するmeta description
	// が空になり、サイト共通の既定値が残らなければならないケースにあたる。
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(4).
		WithNilTitle().
		WithBodyHTML("").
		WithLinkedPageIDs([]model.PageID{}).
		Build()

	// og:imageタグの元になる2つのアイキャッチ画像。タグが指す静止画と、タグが対象外とするGIF。
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
		// spaceIdentifierはリクエストするスペース識別子を上書きする。空なら保存済みの識別子。
		spaceIdentifier string
		pageNumber      string

		// rawQueryはリクエストに付ける関連ページのページネーションフォールバック。
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
				// ページ表示画面は唯一の本文ページのため、サイト共通のwebsite型を上書きする。
				`<meta property="og:type" content="article">`,
				"<script type=\"application/ld+json\">",
				"\"@type\":\"BreadcrumbList\"",
				"\"position\":1,\"name\":\"Page Show Space\",\"item\":\"https://localhost/s/page-show-space\"",
				"\"position\":2,\"name\":\"Public Topic\",\"item\":\"https://localhost/s/page-show-space/topics/1\"",
				"\"position\":3,\"name\":\"Public Page Title\"",
				// 本文の下の2つの一覧。ゲストにも描画される。
				"リンク",
				"Linked Page Title",
				"バックリンク",
				"Backlink Page Title",
				// ヘッダーはどの閲覧者に対しても上端に固定されるため、web/sticky-header.tsが
				// 手がかりにするsentinel、header、spacerはゲストにも描画される。
				"data-sticky-header-sentinel",
				"data-sticky-header",
				"data-sticky-header-spacer",
				// 本文コンテナは、web/markdown-table.tsがテーブルを包むスクロール領域に付ける
				// 名前を持つ。横に長いテーブルのあるページへ来たゲストにも、キーボードで到達できる
				// 領域が用意される。
				`data-markdown-table-label="スクロールできる表"`,
			},
			wantNotContains: []string{
				"このページはゴミ箱に入れられています。",
				// ゲストは編集できないため、ヘッダーの編集ボタンも各カードの編集リンクも出さない。
				"/s/page-show-space/pages/1/edit",
				"/s/page-show-space/pages/5/edit",
				// ゲストはページを操作することもできないため、操作ドロップダウンも、その
				// フォームが要するCSRFトークンも公開HTMLには載せない。
				"page-actions-dropdown",
				"/s/page-show-space/pages/1/trash",
				"/s/page-show-space/pages/1/move",
				showCSRFToken,
				// ヘッダーがコンパクトなバーの最小高を必要とするのは操作領域を持つときだけ。
				// 折り返されたタイトルから失われる高さはspacerが独立して保持する。
				"min-h-[var(--app-sticky-header-height)]",
				"\"name\":\"Public Page Title\",\"item\":",
				`<meta property="og:url" content="">`,
				`<link rel="canonical" href="">`,
				`href="/home"`,
				`https://localhost/home`,
			},
		},
		{
			// spaces.identifierはcitextのため、このリクエストも同じページに到達する。正規URLは
			// 保存済みの識別子を指し続け、大文字小文字の違いでページが複数の自己参照canonicalに
			// 分かれないようにする。
			name:            "識別子の大文字小文字が異なってもcanonicalは保存済みの識別子を指す",
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
			// 関連一覧のページはページ自身の内容を変えないため、どの組み合わせも独自のアドレスでは
			// なく、その内容を持つ1つのアドレスを宣言する。
			name:       "関連一覧の2ページ目はcanonicalとOpen Graph URLをクエリ無しのURLに集約する",
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
			// 正規URLのページ番号はURLパラメータではなく保存済みのページから取る。
			// /pages/001が独自のアドレスにならないようにするためである。
			name:       "ゼロ埋めのページ番号でもcanonicalは正規化された番号を指す",
			pageNumber: "001",
			wantStatus: http.StatusOK,
			wantContains: []string{
				`<meta property="og:url" content="https://localhost/s/page-show-space/pages/1">`,
				`<link rel="canonical" href="https://localhost/s/page-show-space/pages/1">`,
			},
			wantNotContains: []string{"/pages/001"},
		},
		{
			// リンクプレビューはアイキャッチ画像の永続og:image URLを指す。このURLはこのHTMLの
			// キャッシュ寿命を超えても有効なままである (エンドポイントがリクエストごとに公開判定を
			// やり直すため)。
			name:       "アイキャッチ画像を持つページはog:imageに添付ファイルの永続URLを出す",
			pageNumber: "7",
			wantStatus: http.StatusOK,
			wantContains: []string{
				fmt.Sprintf(`<meta property="og:image" content="https://localhost/attachments/%s/og_image">`, coverAttachmentID),
				fmt.Sprintf(`<meta name="twitter:image" content="https://localhost/attachments/%s/og_image">`, coverAttachmentID),
			},
			wantNotContains: []string{"/static/images/og-image.png"},
		},
		{
			// og:imageエンドポイントは静止画のjpgを配信するため、アニメーション画像を指すと画像の
			// 持ち味を失ったプレビューを宣伝することになる。Rails版も同じく対象外にしている。
			name:       "GIFのアイキャッチ画像は既定のOGP画像にフォールバックする",
			pageNumber: "8",
			wantStatus: http.StatusOK,
			wantContains: []string{
				`<meta property="og:image" content="https://localhost/static/images/og-image.png">`,
			},
			wantNotContains: []string{fmt.Sprintf("/attachments/%s/og_image", gifAttachmentID)},
		},
		{
			// アイキャッチ画像を持たないページはサイト共通の既定OGP画像を保つ。
			name:       "アイキャッチ画像を持たないページは既定のOGP画像を出す",
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
			// space:adminのメンバーは両方のスコープを持つため、ドロップダウンには2つの項目が
			// 載り、ゴミ箱フォームにはPOSTに使うトークンが載る。
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
				// 操作領域を持つヘッダーはコンパクトなバーの最小高を確保する。展開時タイトルから
				// 失われる追加の高さは隣接するspacerが処理する。
				"min-h-[var(--app-sticky-header-height)]",
			},
		},
		{
			// 操作領域は、タイトルが何行に折り返しても同じ位置に出るよう見出し行の上端に置き、
			// ヘッダーが固定されてタイトルが操作領域より低くなったときだけ中央に戻す。
			name:       "見出し行は操作領域を上端に置き、固定時だけ中央に戻す",
			pageNumber: "1",
			userID:     &editorUserID,
			wantStatus: http.StatusOK,
			wantContains: []string{
				`class="flex flex-wrap md:flex-nowrap items-start group-data-stuck:items-center justify-between gap-2 group-data-stuck:py-2"`,
			},
			wantNotContains: []string{
				`class="flex flex-wrap md:flex-nowrap items-center justify-between gap-2 group-data-stuck:py-2"`,
			},
		},
		{
			// 2つの項目は別々のスコープに乗るため、page:trashだけのメンバーには移動項目も
			// 編集ボタンも出ないままゴミ箱項目だけが開く。
			name:       "page:trashだけを持つメンバーにはゴミ箱項目だけが出る",
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
			// どちらの項目も出ないため、空のドロップダウンを描画するのではなくトリガー
			// ボタンごと落とす。
			name:       "page:readだけを持つメンバーには操作ドロップダウンが出ない",
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
			name:            "page:trashを持たないメンバーはゴミ箱のページを閲覧できない",
			pageNumber:      "3",
			userID:          &readerUserID,
			wantStatus:      http.StatusNotFound,
			wantNotContains: []string{"Trashed Page Title", "<p>trashed page body</p>"},
		},
		{
			name:       "page:trashを持つメンバーはゴミ箱のページをアラート付きで閲覧できる",
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
				// ページはすでにゴミ箱にあり、再度POSTしても完全削除が先送りされるだけで
				// ある。このメンバーには他の項目も残らないため、ドロップダウンごと消える。
				"page-actions-dropdown",
				"/s/page-show-space/pages/3/trash",
			},
		},
		{
			// 移動項目はゴミ箱状態に従わないため、編集できるメンバーにはゴミ箱のページでも
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
			// このページは本文が無いため生成される説明文は空になり、サイト共通の既定値が
			// 残らなければならない。ここで固定することで、Handlerが空のdescriptionを出す形に
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
			name:       "存在しないページ番号は404",
			pageNumber: "999",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "数値でないページ番号は404",
			pageNumber: "abc",
			wantStatus: http.StatusNotFound,
		},
		// 古い「もっと見る」URLは存在しない一覧の範囲を指す。一覧が黙って消えた画面を描画するの
		// ではなく、トピック詳細・スペース詳細の範囲外 ?pageと同じ答え方をする。
		{
			name:       "最終ページより後ろのlinks_pageは404",
			pageNumber: "1",
			rawQuery:   "links_page=999",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "最終ページより後ろのbacklinks_pageは404",
			pageNumber: "1",
			rawQuery:   "backlinks_page=999",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "リンク先ページの最終バックリンクページより後ろのlinked_backlinks_pageは404",
			pageNumber: "1",
			rawQuery:   "linked_page_number=5&linked_backlinks_page=2",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "リンク一覧に載っていないカードを指すlinked_page_numberは404",
			pageNumber: "1",
			rawQuery:   "linked_page_number=999&linked_backlinks_page=2",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "数値でないlinked_page_numberは404",
			pageNumber: "1",
			rawQuery:   "linked_page_number=abc",
			wantStatus: http.StatusNotFound,
		},
		{
			// 各一覧の1ページ目は常に範囲内のため、既定値を明示的に書いたパラメータでも画面は
			// 通常どおり描画される。
			name:         "1ページ目を指すフォールバックパラメータは通常どおり描画する",
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
				t.Errorf("ステータスコード = %d、期待値 = %d", rr.Code, tt.wantStatus)
			}

			body := rr.Body.String()
			for _, want := range tt.wantContains {
				if !strings.Contains(body, want) {
					t.Errorf("レスポンスに%qが含まれていない", want)
				}
			}
			for _, notWant := range tt.wantNotContains {
				if strings.Contains(body, notWant) {
					t.Errorf("レスポンスに想定外の%qが含まれている", notWant)
				}
			}
		})
	}
}

// TestShow_HeadはHEADがGETと同じ可視性判定を通り、HTTPサーバーが描画した文書の本文を
// 返さないことを固定する。
func TestShow_Head(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("page-show-head-space").
		Build()
	publicTopicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("Public HEAD Topic").
		WithVisibility(int32(model.TopicVisibilityPublic)).
		Build()
	privateTopicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(2).
		WithName("Private HEAD Topic").
		WithVisibility(int32(model.TopicVisibilityPrivate)).
		Build()

	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(1).
		WithTitle("Public HEAD Page").
		WithLinkedPageIDs([]model.PageID{}).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(privateTopicID).
		WithNumber(2).
		WithTitle("Private HEAD Page").
		WithLinkedPageIDs([]model.PageID{}).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(publicTopicID).
		WithNumber(3).
		WithTitle("Trashed HEAD Page").
		WithLinkedPageIDs([]model.PageID{}).
		WithTrashed().
		Build()

	h := setupHandler(t, queries)
	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := i18n.SetLocale(r.Context(), i18n.LangJa)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	router.Head("/s/{space_identifier}/pages/{page_number}", h.Show)

	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	tests := []struct {
		name       string
		pageNumber string
		wantStatus int
	}{
		{
			name:       "公開ページは200",
			pageNumber: "1",
			wantStatus: http.StatusOK,
		},
		{
			name:       "存在しないページは404",
			pageNumber: "999",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "ゲストが閲覧できない非公開ページは404",
			pageNumber: "2",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "ゴミ箱権限のないゲストにゴミ箱のページは404",
			pageNumber: "3",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := "/s/page-show-head-space/pages/" + tt.pageNumber
			req, err := http.NewRequestWithContext(t.Context(), http.MethodHead, server.URL+path, nil)
			if err != nil {
				t.Fatalf("リクエストの作成に失敗: %v", err)
			}

			resp, err := server.Client().Do(req)
			if err != nil {
				t.Fatalf("リクエストに失敗: %v", err)
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("レスポンス本文の読み込みに失敗: %v", err)
			}
			if err := resp.Body.Close(); err != nil {
				t.Fatalf("レスポンス本文のクローズに失敗: %v", err)
			}

			if resp.StatusCode != tt.wantStatus {
				t.Errorf("ステータスコード = %d、期待値 = %d", resp.StatusCode, tt.wantStatus)
			}
			if len(body) != 0 {
				t.Errorf("レスポンス本文 = %q、期待値 = 空", body)
			}
		})
	}
}

// TestShow_RelatedPagePaginationは公開画面における3種類の関連ページ一覧のフルページ
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
	linkedCount := int(viewmodel.LinkLimit + viewmodel.RelatedPageFollowingLimit + 1)
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

	// フォールバックがネストしたバックリンク一覧を進めるカードはリンク一覧の2ページ目にある。
	// そこが、そのカードを進められる唯一のページである。
	selectedLinkedPageID := linkedPageIDs[int(viewmodel.LinkLimit)]
	selectedLinkedPageNumber := model.PageNumber(100 + int(viewmodel.LinkLimit))
	for i := range int(viewmodel.BacklinkLimit + viewmodel.RelatedPageFollowingLimit + 1) {
		testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(model.PageNumber(200 + i)).
			WithTitle(fmt.Sprintf("Shown Nested Backlink %02d", i)).
			WithModifiedAt(baseTime.Add(time.Duration(100+i) * time.Hour)).
			WithLinkedPageIDs([]model.PageID{selectedLinkedPageID}).
			Build()
	}
	for i := range int(viewmodel.PageBacklinkLimit + viewmodel.RelatedPageFollowingLimit + 1) {
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
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	wantContains := []string{
		// 各一覧の2番目の範囲。要求した状態でしか描画されない。
		fmt.Sprintf("Shown Linked Page %02d", viewmodel.LinkLimit),
		fmt.Sprintf("Shown Nested Backlink %02d", viewmodel.BacklinkLimit),
		fmt.Sprintf("Shown Page Backlink %02d", viewmodel.PageBacklinkLimit),
		// フルページフォールバックは親の範囲を入れ替えるためネスト状態を落とす。一方、下の
		// htmxフラグメントは古いカードがDOMに残るためネスト状態を維持する。
		`href="/s/show-related-space/pages/1?backlinks_page=2&amp;links_page=3#page-link-list-content"`,
		fmt.Sprintf(`href="/s/show-related-space/pages/1?backlinks_page=2&amp;linked_backlinks_page=3&amp;linked_page_number=%[1]d&amp;links_page=2#page-link-list-item-%[1]d"`, selectedLinkedPageNumber),
		fmt.Sprintf(`href="/s/show-related-space/pages/1?backlinks_page=3&amp;linked_backlinks_page=2&amp;linked_page_number=%d&amp;links_page=2#page-backlink-list-content"`, selectedLinkedPageNumber),
		// フラグメントURLはページ表示画面の保存済みリンクを使う側に留まる。
		fmt.Sprintf(`hx-get="/s/show-related-space/pages/1/link_list?backlinks_page=2&amp;context=show&amp;linked_backlinks_page=2&amp;linked_page_number=%d&amp;linked_page_parent_page=2&amp;page=3"`, selectedLinkedPageNumber),
	}
	for _, want := range wantContains {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに%qが含まれていない", want)
		}
	}

	// 各一覧の最初の範囲は、追記ではなく差し替えられている。
	for _, notWant := range []string{"Shown Linked Page 00", "Shown Nested Backlink 00", "Shown Page Backlink 00"} {
		if strings.Contains(body, notWant) {
			t.Errorf("レスポンスに想定外の%qが含まれている", notWant)
		}
	}

	// 上のフォールバックURLが指すアンカーはいずれも描画されている。JavaScriptが使えない
	// 閲覧者も、画面の先頭ではなく自分が進めていた一覧に着地する。3つのアンカーは3セクションに
	// 散っており、ネスト一覧のものは関連リンクのグループへ移っている。
	for _, anchor := range []string{
		"page-link-list-content",
		fmt.Sprintf("page-link-list-item-%d", selectedLinkedPageNumber),
		"page-backlink-list-content",
	} {
		if !strings.Contains(body, fmt.Sprintf(`id="%s"`, anchor)) {
			t.Errorf("ページ全体へのフォールバックのアンカー%qが描画されていない", anchor)
		}
	}
}

// TestShow_RelatedLinkSectionsは本文下の一覧の3セクション構成を固定する。リンク先ページ、
// リンク先ページごとに束ねたそのバックリンク、そしてこのページ自身のバックリンクである。どこからも
// リンクされていないリンク先ページがグループを生まないことも併せて固定する。
func TestShow_RelatedLinkSections(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("show-sections-space").
		WithName("Show Sections Space").
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("Public Topic").
		WithVisibility(int32(model.TopicVisibilityPublic)).
		Build()

	// リンク先ページのうち片方にはバックリンクがあり、もう片方には無い。セクションにはちょうど
	// 1つのグループが残る。
	linkedPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Linked Page").
		WithLinkedPageIDs([]model.PageID{}).
		Build()
	lonelyLinkedPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(3).
		WithTitle("Lonely Linked Page").
		WithLinkedPageIDs([]model.PageID{}).
		Build()

	pageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Shown Page").
		WithBodyHTML("<p>shown page body</p>").
		WithLinkedPageIDs([]model.PageID{linkedPageID, lonelyLinkedPageID}).
		Build()

	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(4).
		WithTitle("Related Link Page").
		WithLinkedPageIDs([]model.PageID{linkedPageID}).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(5).
		WithTitle("Backlink Page").
		WithLinkedPageIDs([]model.PageID{pageID}).
		Build()

	h := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/s/show-sections-space/pages/1", map[string]string{
		"space_identifier": "show-sections-space",
		"page_number":      "1",
	})
	req = req.WithContext(i18n.SetLocale(req.Context(), i18n.LangJa))

	rr := httptest.NewRecorder()
	h.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	for _, want := range []string{
		"関連リンク",
		"このページが直接リンクしているページ",
		"リンク先のページからさらに辿れるページ",
		"このページにリンクしているページ",
		`id="page-link-list-item-2"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("レスポンスに%qが含まれていない", want)
		}
	}

	// どこからもリンクされていないリンク先ページにはグループが付かない。バックリンクの
	// セクションは、これまでどおりこのページ自身のバックリンクだけを表す。
	if strings.Contains(body, `id="page-link-list-item-3"`) {
		t.Error("バックリンクの無いリンク先ページに関連リンクのグループが付いている")
	}

	// 3セクションはリンク、関連リンク、バックリンクの順に並ぶ。リンク先ページのバックリンクは
	// リンクセクションのカードの隣ではなく、関連リンクのセクションに置かれる。
	linksIndex := strings.Index(body, `id="page-link-list-content"`)
	relatedIndex := strings.Index(body, `id="page-related-link-list"`)
	backlinksIndex := strings.Index(body, `id="page-backlink-list-content"`)
	if linksIndex == -1 || relatedIndex == -1 || backlinksIndex == -1 {
		t.Fatalf(
			"セクションの位置 = links:%d related-links:%d backlinks:%d、3つのセクションすべてが描画されていない",
			linksIndex,
			relatedIndex,
			backlinksIndex,
		)
	}
	if linksIndex >= relatedIndex || relatedIndex >= backlinksIndex {
		t.Errorf(
			"セクションの位置 = links:%d related-links:%d backlinks:%d、期待値 = links < related-links < backlinks",
			linksIndex,
			relatedIndex,
			backlinksIndex,
		)
	}
	if got := strings.Index(body, "Related Link Page"); got < relatedIndex {
		t.Errorf("リンク先ページのバックリンクが位置%dに描画され、位置%dの関連リンクセクションより前にある", got, relatedIndex)
	}

	// 3つのセクションはいずれも見出しの脇に説明を持つ。名前が互いに近い3つを、そのセクションが
	// 何を並べているかで読み分けられるようにするためである。
	for _, heading := range []string{"リンク", "関連リンク", "バックリンク"} {
		marker := fmt.Sprintf(">%s</h2>", heading)
		headingIndex := strings.Index(body, marker)
		if headingIndex == -1 {
			t.Errorf("見出し%qが描画されていない", heading)
			continue
		}
		if !strings.HasPrefix(body[headingIndex+len(marker):], `<p class="text-sm text-muted-foreground">`) {
			t.Errorf("見出し%qの後ろに説明文が無い", heading)
		}
	}
}

// TestShow_EmptyRelatedLinkSectionは、空の関連リンクセクションがOOBの追記先を文書内に残し
// つつ、セクションのラッパーで見出しと説明を隠すことを固定する。
func TestShow_EmptyRelatedLinkSection(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("show-empty-related-space").
		WithName("Show Empty Related Space").
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("Public Topic").
		WithVisibility(int32(model.TopicVisibilityPublic)).
		Build()

	linkedPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Linked Page Without Other Backlinks").
		WithLinkedPageIDs([]model.PageID{}).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Shown Page With Empty Related Links").
		WithLinkedPageIDs([]model.PageID{linkedPageID}).
		Build()

	h := setupHandler(t, queries)

	req := newRequestWithChiParams(t, http.MethodGet, "/s/show-empty-related-space/pages/1", map[string]string{
		"space_identifier": "show-empty-related-space",
		"page_number":      "1",
	})
	req = req.WithContext(i18n.SetLocale(req.Context(), i18n.LangJa))

	rr := httptest.NewRecorder()
	h.Show(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("ステータスコード = %d、期待値 = %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, `id="page-related-link-list"`) {
		t.Error("空の関連リンクのコンテナがOOBの対象として残っていない")
	}
	if !strings.Contains(body, `not-has-[#page-related-link-list>*]:hidden`) {
		t.Error("コンテナが空なのに関連リンクセクションが隠れていない")
	}
	if strings.Contains(body, `id="page-link-list-item-`) {
		t.Error("空の関連リンクセクションにグループが含まれている")
	}
}
