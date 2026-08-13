package draft_page_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	xhtml "golang.org/x/net/html"

	"github.com/wikinoapp/wikino/go/internal/config"
	pagehandler "github.com/wikinoapp/wikino/go/internal/handler/page"
	pagebacklinklisthandler "github.com/wikinoapp/wikino/go/internal/handler/page_backlink_list"
	pagebacklinkshandler "github.com/wikinoapp/wikino/go/internal/handler/page_backlinks"
	pagelinklisthandler "github.com/wikinoapp/wikino/go/internal/handler/page_link_list"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/session"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/viewmodel"
)

// TestShow_ReconcilesEveryRelatedPageStateAfterDraftShrink pins both draft-refresh modes. A stale
// requested page is clamped to the last surviving page, and a nested state whose parent card left
// the draft is cleared. The one-page editor must refetch after clamping because its first slice was
// empty, while the cumulative editor already fetched the surviving prefix.
//
// [Ja] TestShow_ReconcilesEveryRelatedPageStateAfterDraftShrink は、下書き再取得の両モードを固定する。
// 古い要求ページは残存する最終ページへ戻し、親カードが下書きから消えたネスト状態は解除する。
// 1 ページ単位編集画面は最初のスライスが空なので補正後に再取得し、累積編集画面は取得済みの残存範囲を使う。
func TestShow_ReconcilesEveryRelatedPageStateAfterDraftShrink(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("related-shrink@example.com").
		WithAtname("relatedshrink").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("related-shrink").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID).
		Build()

	survivingPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(2).
		WithTitle("Surviving Draft Link").
		Build()
	removedPageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(3).
		WithTitle("Removed Draft Link").
		Build()
	sourcePageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Shrink Source").
		WithLinkedPageIDs([]model.PageID{removedPageID}).
		Build()
	testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(sourcePageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		WithTitle("Shrink Source Draft").
		WithLinkedPageIDs([]model.PageID{survivingPageID}).
		Build()
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(4).
		WithTitle("Only Page Backlink").
		WithLinkedPageIDs([]model.PageID{sourcePageID}).
		Build()

	for i := range int(viewmodel.BacklinkLimit) + 1 {
		testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(model.PageNumber(100 + i)).
			WithTitle(fmt.Sprintf("Surviving Nested Backlink %02d", i)).
			WithLinkedPageIDs([]model.PageID{survivingPageID}).
			Build()
	}

	handler := setupHandler(t, queries)
	tests := []struct {
		name                   string
		context                viewmodel.PageLinkContext
		selectedPageNumber     int32
		wantSelectedPageValue  string
		wantNestedPageValue    string
		wantNestedPageTwoItems bool
	}{
		{
			name:               "cumulative editor clears a removed parent",
			context:            viewmodel.PageLinkContextEdit,
			selectedPageNumber: 3,
		},
		{
			name:               "one-page editor clears a removed parent",
			context:            viewmodel.PageLinkContextEditPaginated,
			selectedPageNumber: 3,
		},
		{
			name:                   "cumulative editor keeps a parent moved to the clamped page",
			context:                viewmodel.PageLinkContextEdit,
			selectedPageNumber:     2,
			wantSelectedPageValue:  "2",
			wantNestedPageValue:    "2",
			wantNestedPageTwoItems: true,
		},
		{
			name:                   "one-page editor keeps a parent moved to the clamped page",
			context:                viewmodel.PageLinkContextEditPaginated,
			selectedPageNumber:     2,
			wantSelectedPageValue:  "2",
			wantNestedPageValue:    "2",
			wantNestedPageTwoItems: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rawQuery := fmt.Sprintf(
				"context=%s&links_page=2&linked_page_number=%d&linked_backlinks_page=2&backlinks_page=2",
				tt.context,
				tt.selectedPageNumber,
			)
			req := newShowRequest(t, "/s/related-shrink/pages/1/draft_page?"+rawQuery, map[string]string{
				"space_identifier": "related-shrink",
				"page_number":      "1",
			})
			req.URL.RawQuery = rawQuery
			req = req.WithContext(middleware.SetUserToContext(req.Context(), &model.User{ID: userID}))

			rr := httptest.NewRecorder()
			handler.Show(rr, req)
			if rr.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
			}

			body := rr.Body.String()
			if !strings.Contains(body, "Surviving Draft Link") {
				t.Error("response lost the surviving draft link after clamping")
			}
			if strings.Contains(body, "Removed Draft Link") {
				t.Error("response followed the saved link set instead of the shrunken draft")
			}
			for _, stateID := range []string{"page-link-list-state", "page-backlink-list-state"} {
				if got := inputValueByID(t, body, stateID); got != "" {
					t.Errorf("%s value = %q, want first-page empty value", stateID, got)
				}
			}
			if got := inputValueByName(t, body, viewmodel.LinkedPageNumberQueryParam); got != tt.wantSelectedPageValue {
				t.Errorf("selected nested page value = %q, want %q", got, tt.wantSelectedPageValue)
			}
			if got := inputValueByName(t, body, viewmodel.LinkedBacklinkPageQueryParam); got != tt.wantNestedPageValue {
				t.Errorf("nested backlink page value = %q, want %q", got, tt.wantNestedPageValue)
			}
			if tt.wantNestedPageTwoItems && !strings.Contains(body, "Surviving Nested Backlink 00") {
				t.Error("response did not refetch the surviving card's second nested page")
			}
		})
	}
}

// TestRelatedPageHandlers_ResponseStateChain drives the real page-link-list and page-backlink
// handlers, applies each OOB state value to the next request, and finally refreshes the draft. It
// fixes the HTTP-boundary contract that independent state elements compose into both loaded ranges.
//
// [Ja] TestRelatedPageHandlers_ResponseStateChain は実際のリンク一覧・ページ自身のバックリンク Handler を
// 呼び、各 OOB 状態値を次の要求へ反映して、最後に下書きを再取得する。独立した状態要素を合成すると
// 両一覧の読み込み済み範囲が保たれる HTTP 境界の契約を固定する。
func TestRelatedPageHandlers_ResponseStateChain(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("related-flow@example.com").
		WithAtname("relatedflow").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("related-flow").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID).
		Build()

	baseTime := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	linkedCount := int(viewmodel.LinkLimit) + 1
	linkedPageIDs := make([]model.PageID, 0, linkedCount)
	for i := range linkedCount {
		linkedPageIDs = append(linkedPageIDs, testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(model.PageNumber(100+i)).
			WithTitle(fmt.Sprintf("Flow Link %02d", i)).
			WithModifiedAt(baseTime.Add(time.Duration(i)*time.Hour)).
			Build())
	}
	sourcePageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Flow Source").
		WithLinkedPageIDs(linkedPageIDs).
		Build()

	backlinkCount := int(viewmodel.PageBacklinkLimit) + 1
	for i := range backlinkCount {
		testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(model.PageNumber(200 + i)).
			WithTitle(fmt.Sprintf("Flow Backlink %02d", i)).
			WithLinkedPageIDs([]model.PageID{sourcePageID}).
			WithModifiedAt(baseTime.Add(time.Duration(100+i) * time.Hour)).
			Build()
	}

	spaceRepo := repository.NewSpaceRepository(queries)
	spaceMemberRepo := repository.NewSpaceMemberRepository(queries)
	pageRepo := repository.NewPageRepository(queries)
	topicRepo := repository.NewTopicRepository(queries)
	topicMemberRepo := repository.NewTopicMemberRepository(queries)
	draftPageRepo := repository.NewDraftPageRepository(queries)
	linkListHandler := pagelinklisthandler.NewHandler(usecase.NewGetLinkListUsecase(
		spaceRepo,
		spaceMemberRepo,
		pageRepo,
		topicRepo,
		topicMemberRepo,
		draftPageRepo,
	))
	pageBacklinksHandler := pagebacklinkshandler.NewHandler(usecase.NewGetPageBacklinksUsecase(
		spaceRepo,
		spaceMemberRepo,
		pageRepo,
		topicRepo,
		topicMemberRepo,
	))
	draftHandler := setupHandler(t, queries)

	linkQuery := "context=edit&page=2"
	linkRequest := newShowRequest(t, "/s/related-flow/pages/1/link_list?"+linkQuery, map[string]string{
		"space_identifier": "related-flow",
		"page_number":      "1",
	})
	linkRequest.URL.RawQuery = linkQuery
	linkRequest = linkRequest.WithContext(middleware.SetUserToContext(linkRequest.Context(), &model.User{ID: userID}))
	linkResponse := httptest.NewRecorder()
	linkListHandler.Show(linkResponse, linkRequest)
	if linkResponse.Code != http.StatusOK {
		t.Fatalf("link-list status = %d, want %d", linkResponse.Code, http.StatusOK)
	}
	linkPage := inputValueByID(t, linkResponse.Body.String(), "page-link-list-state")
	if linkPage != "2" {
		t.Fatalf("link-list OOB state = %q, want 2", linkPage)
	}

	backlinkQuery := "context=edit&page=2&links_page=" + linkPage
	backlinkRequest := newShowRequest(t, "/s/related-flow/pages/1/backlinks?"+backlinkQuery, map[string]string{
		"space_identifier": "related-flow",
		"page_number":      "1",
	})
	backlinkRequest.URL.RawQuery = backlinkQuery
	backlinkRequest = backlinkRequest.WithContext(middleware.SetUserToContext(backlinkRequest.Context(), &model.User{ID: userID}))
	backlinkResponse := httptest.NewRecorder()
	pageBacklinksHandler.Show(backlinkResponse, backlinkRequest)
	if backlinkResponse.Code != http.StatusOK {
		t.Fatalf("page-backlinks status = %d, want %d", backlinkResponse.Code, http.StatusOK)
	}
	backlinkPage := inputValueByID(t, backlinkResponse.Body.String(), "page-backlink-list-state")
	if backlinkPage != "2" {
		t.Fatalf("page-backlinks OOB state = %q, want 2", backlinkPage)
	}

	draftQuery := "context=edit&links_page=" + linkPage + "&backlinks_page=" + backlinkPage
	draftRequest := newShowRequest(t, "/s/related-flow/pages/1/draft_page?"+draftQuery, map[string]string{
		"space_identifier": "related-flow",
		"page_number":      "1",
	})
	draftRequest.URL.RawQuery = draftQuery
	draftRequest = draftRequest.WithContext(middleware.SetUserToContext(draftRequest.Context(), &model.User{ID: userID}))
	draftResponse := httptest.NewRecorder()
	draftHandler.Show(draftResponse, draftRequest)
	if draftResponse.Code != http.StatusOK {
		t.Fatalf("draft refresh status = %d, want %d", draftResponse.Code, http.StatusOK)
	}

	body := draftResponse.Body.String()
	for _, want := range []string{
		"Flow Link 00",
		fmt.Sprintf("Flow Link %02d", linkedCount-1),
		"Flow Backlink 00",
		fmt.Sprintf("Flow Backlink %02d", backlinkCount-1),
	} {
		if !strings.Contains(body, want) {
			t.Errorf("draft refresh does not contain %q", want)
		}
	}
	if got := inputValueByID(t, body, "page-link-list-state"); got != "2" {
		t.Errorf("draft link-list state = %q, want 2", got)
	}
	if got := inputValueByID(t, body, "page-backlink-list-state"); got != "2" {
		t.Errorf("draft page-backlink state = %q, want 2", got)
	}
}

func inputValueByID(t *testing.T, body string, id string) string {
	t.Helper()
	return inputAttributeValue(t, body, "id", id)
}

func inputValueByName(t *testing.T, body string, name string) string {
	t.Helper()
	return inputAttributeValue(t, body, "name", name)
}

func inputAttributeValue(t *testing.T, body string, key string, expected string) string {
	t.Helper()

	tokenizer := xhtml.NewTokenizer(strings.NewReader(body))
	for {
		switch tokenizer.Next() {
		case xhtml.ErrorToken:
			t.Fatalf("input %s=%q not found in response", key, expected)
		case xhtml.StartTagToken, xhtml.SelfClosingTagToken:
			token := tokenizer.Token()
			if token.Data != "input" {
				continue
			}

			attributes := make(map[string]string, len(token.Attr))
			for _, attribute := range token.Attr {
				attributes[attribute.Key] = attribute.Val
			}
			if attributes[key] == expected {
				return attributes["value"]
			}
		}
	}
}

// TestRelatedPageHandlers_NestedFallbackStaysReachableAfterTheLinkListAdvances drives the real
// handlers through the sequence that pairs a screen-wide link page with a card from an earlier one.
// The editor sends the shared state with every request, so after the link list advances, a nested
// listing on an earlier page is advanced with links_page naming a page that card is not on. The
// full-page fallback the response renders has to name the card's own page instead, because the
// editor renders a single link-list page and answers a request for an absent card with a 404.
//
// [Ja] TestRelatedPageHandlers_NestedFallbackStaysReachableAfterTheLinkListAdvances は、画面全体の
// リンクページと、それより前のページのカードが組み合わさる流れを実 Handler で辿る。編集画面は共有状態を
// 毎回のリクエストに載せるため、リンク一覧が進んだ後に前のページのカードのネスト一覧を進めると、その
// カードが載っていないページを指す links_page が届く。応答が描画するフルページフォールバックは代わりに
// カード自身のページを指す必要がある。編集画面はリンク一覧を 1 ページしか描画せず、存在しないカードを
// 指すリクエストには 404 を返すためである。
func TestRelatedPageHandlers_NestedFallbackStaysReachableAfterTheLinkListAdvances(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("nested-fallback@example.com").
		WithAtname("nestedfallback").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("nested-fallback").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID).
		Build()

	// The listing is ordered by modified_at descending, so the newest linked page opens the first
	// link-list page and one more page than the limit pushes the oldest onto the second.
	//
	// [Ja] 一覧は modified_at の降順のため、最も新しいリンク先ページが 1 ページ目の先頭に来る。件数上限
	// より 1 件多いので、最も古いページが 2 ページ目に押し出される。
	baseTime := time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC)
	linkedCount := int(viewmodel.LinkLimit) + 1
	linkedPageIDs := make([]model.PageID, 0, linkedCount)
	for i := range linkedCount {
		linkedPageIDs = append(linkedPageIDs, testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(model.PageNumber(100+i)).
			WithTitle(fmt.Sprintf("Nested Fallback Link %02d", i)).
			WithModifiedAt(baseTime.Add(-time.Duration(i)*time.Hour)).
			Build())
	}
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Nested Fallback Source").
		WithLinkedPageIDs(linkedPageIDs).
		Build()

	// The first card of the first link-list page gets three nested pages worth of backlinks, so its
	// second page still renders a "load more" link rather than the end-of-listing status.
	//
	// [Ja] リンク一覧 1 ページ目の先頭カードに、ネスト 3 ページ分のバックリンクを与える。2 ページ目でも
	// 一覧終端の状態ではなく「もっと見る」リンクが描画されるようにするためである。
	selectedLinkedPageNumber := int32(100)
	for i := range int(viewmodel.BacklinkLimit)*2 + 1 {
		testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(model.PageNumber(200 + i)).
			WithTitle(fmt.Sprintf("Nested Fallback Backlink %02d", i)).
			WithLinkedPageIDs([]model.PageID{linkedPageIDs[0]}).
			Build()
	}

	spaceRepo := repository.NewSpaceRepository(queries)
	spaceMemberRepo := repository.NewSpaceMemberRepository(queries)
	pageRepo := repository.NewPageRepository(queries)
	topicRepo := repository.NewTopicRepository(queries)
	topicMemberRepo := repository.NewTopicMemberRepository(queries)
	draftPageRepo := repository.NewDraftPageRepository(queries)

	linkListHandler := pagelinklisthandler.NewHandler(usecase.NewGetLinkListUsecase(
		spaceRepo,
		spaceMemberRepo,
		pageRepo,
		topicRepo,
		topicMemberRepo,
		draftPageRepo,
	))
	backlinkListHandler := pagebacklinklisthandler.NewHandler(usecase.NewGetBacklinkListUsecase(
		spaceRepo,
		spaceMemberRepo,
		pageRepo,
		topicRepo,
		topicMemberRepo,
	))
	editHandler := setupEditHandler(t, queries)

	chiParams := map[string]string{
		"space_identifier": "nested-fallback",
		"page_number":      "1",
	}
	focusID := fmt.Sprintf("page-backlink-list-%d-load-more", selectedLinkedPageNumber)

	// 1. Load the editor and keep the nested "load more" of a first-page card, together with the
	//    shared state every later request sends.
	//
	// [Ja] 1. 編集画面を読み込み、1 ページ目のカードのネスト「もっと見る」を、以降の各リクエストが送る
	//    共有状態と一緒に保持する。
	editorRequest := newShowRequest(t, "/s/nested-fallback/pages/1/edit", chiParams)
	editorRequest = editorRequest.WithContext(middleware.SetUserToContext(editorRequest.Context(), &model.User{ID: userID}))
	editorRecorder := httptest.NewRecorder()
	editHandler.Edit(editorRecorder, editorRequest)
	if editorRecorder.Code != http.StatusOK {
		t.Fatalf("editor status = %d, want %d", editorRecorder.Code, http.StatusOK)
	}
	editorBody := editorRecorder.Body.String()
	state := applyRelatedPageState(t, editorBody, nil)
	nestedFragmentURL := anchorAttributeByID(t, editorBody, focusID, "hx-get")

	// 2. Advance the link list to its second page, which moves the shared state to links_page=2.
	//
	// [Ja] 2. リンク一覧を 2 ページ目へ進め、共有状態を links_page=2 にする。
	linkFragmentURL := anchorAttributeByID(t, editorBody, "page-link-list-load-more", "hx-get")
	linkPath, linkQuery := htmxRequestQuery(t, linkFragmentURL, state)
	linkResponse := callRelatedPageHandler(t, linkListHandler.Show, userID, linkPath, linkQuery, chiParams)
	state = applyRelatedPageState(t, linkResponse, state)
	if got := state.Get(viewmodel.LinkPageQueryParam); got != "2" {
		t.Fatalf("shared links_page = %q, want 2", got)
	}

	// 3. Advance the nested listing of the card that stayed on the first link-list page. The shared
	//    state now says links_page=2 while the card's own link still names parent page 1.
	//
	// [Ja] 3. 1 ページ目に残っているカードのネスト一覧を進める。共有状態は links_page=2 を伝える一方、
	//    カード自身のリンクは親ページ 1 を指したままである。
	nestedPath, nestedQuery := htmxRequestQuery(t, nestedFragmentURL, state)
	nestedResponse := callRelatedPageHandler(t, backlinkListHandler.Show, userID, nestedPath, nestedQuery,
		map[string]string{
			"space_identifier":   "nested-fallback",
			"page_number":        "1",
			"linked_page_number": fmt.Sprintf("%d", selectedLinkedPageNumber),
		})

	fallbackHref := anchorHrefByID(t, nestedResponse, focusID)
	fallbackURL, err := url.Parse(fallbackHref)
	if err != nil {
		t.Fatalf("parse fallback href %q: %v", fallbackHref, err)
	}
	if got := fallbackURL.Query().Get(viewmodel.LinkPageQueryParam); got != "" {
		t.Errorf("fallback links_page = %q, want the card's own first page (omitted)", got)
	}

	// 4. Follow that fallback as a viewer without JavaScript would. The editor has to render it.
	//
	// [Ja] 4. JavaScript の無い閲覧者と同じようにそのフォールバックを辿る。編集画面はこれを描画できな
	//    ければならない。
	editResponse := httptest.NewRecorder()
	editRequest := newShowRequest(t, fallbackURL.String(), chiParams)
	editRequest.URL.RawQuery = fallbackURL.RawQuery
	editRequest = editRequest.WithContext(middleware.SetUserToContext(editRequest.Context(), &model.User{ID: userID}))
	editHandler.Edit(editResponse, editRequest)

	if editResponse.Code != http.StatusOK {
		t.Fatalf("editor status for %q = %d, want %d", fallbackHref, editResponse.Code, http.StatusOK)
	}
	body := editResponse.Body.String()
	if !strings.Contains(body, "Nested Fallback Link 00") {
		t.Error("the editor did not render the link-list page holding the selected card")
	}
	if !strings.Contains(body, "Nested Fallback Backlink 00") {
		t.Error("the editor did not render the selected card's second nested backlink page")
	}

	// 5. The same nested state paired with the link page the screen had reached is out of range, which
	//    is what makes step 4 a requirement rather than a preference.
	//
	// [Ja] 5. 同じネスト状態を、画面が到達していたリンクページと組み合わせると範囲外になる。これが
	//    ステップ 4 を好みではなく要件にしている。
	staleQuery := fallbackURL.Query()
	staleQuery.Set(viewmodel.LinkPageQueryParam, "2")
	staleResponse := httptest.NewRecorder()
	staleRequest := newShowRequest(t, "/s/nested-fallback/pages/1/edit", chiParams)
	staleRequest.URL.RawQuery = staleQuery.Encode()
	staleRequest = staleRequest.WithContext(middleware.SetUserToContext(staleRequest.Context(), &model.User{ID: userID}))
	editHandler.Edit(staleResponse, staleRequest)

	if staleResponse.Code != http.StatusNotFound {
		t.Errorf("editor status for the stale link page = %d, want %d", staleResponse.Code, http.StatusNotFound)
	}
}

// setupEditHandler builds the page handler this file drives. Only Edit is exercised, so the two
// usecases behind Show and Update are left out rather than wired up for a path never taken.
//
// [Ja] setupEditHandler は本ファイルが駆動するページ Handler を組み立てる。使うのは Edit だけのため、
// Show と Update が使う 2 つの UseCase は、通らない経路のために用意せず省く。
func setupEditHandler(t *testing.T, queries *query.Queries) *pagehandler.Handler {
	t.Helper()

	cfg := &config.Config{Env: "test", Port: "8080", Domain: "localhost"}
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	spaceRepo := repository.NewSpaceRepository(queries)
	spaceMemberRepo := repository.NewSpaceMemberRepository(queries)
	pageRepo := repository.NewPageRepository(queries)
	topicRepo := repository.NewTopicRepository(queries)
	topicMemberRepo := repository.NewTopicMemberRepository(queries)
	draftPageRepo := repository.NewDraftPageRepository(queries)
	draftPageRevisionRepo := repository.NewDraftPageRevisionRepository(queries)
	suggestionPageRepo := repository.NewSuggestionPageRepository(queries)
	suggestionRepo := repository.NewSuggestionRepository(queries)

	getPageDetailUC := usecase.NewGetPageDetailUsecase(
		spaceRepo,
		spaceMemberRepo,
		pageRepo,
		draftPageRepo,
		draftPageRevisionRepo,
		topicRepo,
		topicMemberRepo,
		suggestionPageRepo,
		suggestionRepo,
	)
	getEditLinkDataUC := usecase.NewGetEditLinkDataUsecase(pageRepo, topicRepo)

	return pagehandler.NewHandler(cfg, flashMgr, nil, getPageDetailUC, getEditLinkDataUC, nil)
}

// callRelatedPageHandler runs one related-page fragment handler as the signed-in member and returns
// its body, failing the test on any status other than 200.
//
// [Ja] callRelatedPageHandler は関連ページのフラグメント Handler を 1 つ、ログイン済みメンバーとして
// 実行し、本文を返す。200 以外のステータスはテストを失敗させる。
func callRelatedPageHandler(
	t *testing.T,
	handle func(http.ResponseWriter, *http.Request),
	userID model.UserID,
	path string,
	rawQuery string,
	params map[string]string,
) string {
	t.Helper()

	req := newShowRequest(t, path+"?"+rawQuery, params)
	req.URL.RawQuery = rawQuery
	req = req.WithContext(middleware.SetUserToContext(req.Context(), &model.User{ID: userID}))

	rr := httptest.NewRecorder()
	handle(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("%s status = %d, want %d", path, rr.Code, http.StatusOK)
	}

	return rr.Body.String()
}

// anchorHrefByID returns the href of the anchor carrying the given id, which is the stable focus id
// each listing gives its "load more" link.
//
// [Ja] anchorHrefByID は指定 id を持つアンカーの href を返す。この id は各一覧が「もっと見る」リンクに
// 与える安定したフォーカス用 id である。
func anchorHrefByID(t *testing.T, body string, id string) string {
	t.Helper()
	return anchorAttributeByID(t, body, id, "href")
}

func anchorAttributeByID(t *testing.T, body string, id string, key string) string {
	t.Helper()

	tokenizer := xhtml.NewTokenizer(strings.NewReader(body))
	for {
		switch tokenizer.Next() {
		case xhtml.ErrorToken:
			t.Fatalf("anchor id=%q not found in response", id)
		case xhtml.StartTagToken, xhtml.SelfClosingTagToken:
			token := tokenizer.Token()
			if token.Data != "a" {
				continue
			}

			attributes := make(map[string]string, len(token.Attr))
			for _, attribute := range token.Attr {
				attributes[attribute.Key] = attribute.Val
			}
			if attributes["id"] == id {
				return attributes[key]
			}
		}
	}
}

// relatedPageStateNames are the inputs of the editor's shared related-page state element, which
// every editor request sends alongside the URL of the link that started it.
//
// [Ja] relatedPageStateNames は編集画面で共有する関連ページ状態要素の入力。編集画面の各リクエストは、
// きっかけとなったリンクの URL と一緒にこれらを送る。
var relatedPageStateNames = []string{
	viewmodel.PageLinkContextQueryParam,
	viewmodel.LinkPageQueryParam,
	viewmodel.LinkedPageNumberQueryParam,
	viewmodel.LinkedBacklinkPageQueryParam,
	viewmodel.LinkedPageParentPageQueryParam,
	viewmodel.PageBacklinkPageQueryParam,
}

// applyRelatedPageState merges the shared-state inputs a response carries into the state the screen
// holds, which is what an out-of-band swap does to the element in the DOM. A response that advances
// one listing only carries that listing's inputs, so the others keep their previous values.
//
// [Ja] applyRelatedPageState は、応答が運ぶ共有状態の入力を画面が保持する状態へ統合する。DOM 上の
// 要素に対して OOB スワップが行うのと同じ操作である。1 つの一覧を進める応答はその一覧の入力しか
// 運ばないため、他の入力は以前の値のままになる。
func applyRelatedPageState(t *testing.T, body string, state url.Values) url.Values {
	t.Helper()

	if state == nil {
		state = url.Values{}
	}

	wanted := make(map[string]struct{}, len(relatedPageStateNames))
	for _, name := range relatedPageStateNames {
		wanted[name] = struct{}{}
	}

	tokenizer := xhtml.NewTokenizer(strings.NewReader(body))
	for {
		switch tokenizer.Next() {
		case xhtml.ErrorToken:
			return state
		case xhtml.StartTagToken, xhtml.SelfClosingTagToken:
			token := tokenizer.Token()
			if token.Data != "input" {
				continue
			}

			attributes := make(map[string]string, len(token.Attr))
			for _, attribute := range token.Attr {
				attributes[attribute.Key] = attribute.Val
			}
			if _, ok := wanted[attributes["name"]]; ok {
				state.Set(attributes["name"], attributes["value"])
			}
		}
	}
}

// htmxRequestQuery composes the query an editor request actually reaches the handler with. htmx
// drops the query parameters whose names the sent values also use and then appends those values, so
// a name the link's URL shares with the shared state arrives holding the screen-wide value. Building
// the request any other way would let a test pass on a link the browser cannot send.
//
// [Ja] htmxRequestQuery は、編集画面のリクエストが実際に Handler へ届くときのクエリを組み立てる。
// htmx は送信値と同名のクエリパラメータを落としてから送信値を追記するため、リンクの URL が共有状態と
// 共有している名前は画面全体の値として届く。これ以外の組み立て方をすると、ブラウザが送れないリンクで
// テストが通ってしまう。
func htmxRequestQuery(t *testing.T, fragmentURL string, state url.Values) (string, string) {
	t.Helper()

	parsed, err := url.Parse(fragmentURL)
	if err != nil {
		t.Fatalf("parse fragment url %q: %v", fragmentURL, err)
	}

	query := parsed.Query()
	for name := range state {
		query.Del(name)
	}
	for name, values := range state {
		for _, value := range values {
			query.Add(name, value)
		}
	}

	return parsed.Path, query.Encode()
}

// TestRelatedPageHandlers_NestedFallbackSurvivesTheSharedStateOnALaterLinkPage drives the sequence
// the previous test covers from the other side: the card being advanced sits on the link-list page
// the screen has reached, and the shared state holds no card at all. Every editor request sends that
// state, so the card's own link-list page only reaches the handler while it travels under a name the
// state does not use. Composing the request the way htmx does is what makes that a requirement here
// rather than an assumption about the link's URL.
//
// [Ja] TestRelatedPageHandlers_NestedFallbackSurvivesTheSharedStateOnALaterLinkPage は、直前のテストが
// 扱う流れを逆側から辿る。進めるカードは画面が到達しているリンク一覧ページにあり、共有状態はどのカードも
// 保持していない。編集画面の各リクエストはその状態を送るため、カード自身のリンク一覧ページは、状態が
// 使っていない名前で運ばれる間しか Handler へ届かない。リクエストを htmx と同じ方法で組み立てることが、
// これをリンクの URL に対する仮定ではなく要件にしている。
func TestRelatedPageHandlers_NestedFallbackSurvivesTheSharedStateOnALaterLinkPage(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	queries := testutil.QueriesWithTx(tx)

	userID := testutil.NewUserBuilder(t, tx).
		WithEmail("later-parent@example.com").
		WithAtname("laterparent").
		Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).
		WithIdentifier("later-parent").
		Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithUserID(userID).
		Build()
	topicID := testutil.NewTopicBuilder(t, tx).
		WithSpaceID(spaceID).
		WithNumber(1).
		WithName("General").
		Build()
	testutil.NewTopicMemberBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithSpaceMemberID(spaceMemberID).
		Build()

	// One more linked page than fits on a page, ordered so that the oldest one is the single card on
	// the second link-list page.
	//
	// [Ja] 1 ページに収まる件数より 1 つ多くリンク先ページを作り、最も古いものが 2 ページ目の唯一の
	// カードになる並びにする。
	baseTime := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	linkedCount := int(viewmodel.LinkLimit) + 1
	linkedPageIDs := make([]model.PageID, 0, linkedCount)
	secondPageCardNumber := int32(100)
	for i := range linkedCount {
		linkedPageIDs = append(linkedPageIDs, testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(model.PageNumber(100+i)).
			WithTitle(fmt.Sprintf("Later Parent Link %02d", i)).
			WithModifiedAt(baseTime.Add(time.Duration(i)*time.Hour)).
			Build())
	}
	testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithTopicID(topicID).
		WithNumber(1).
		WithTitle("Later Parent Source").
		WithLinkedPageIDs(linkedPageIDs).
		Build()

	// That card gets three nested pages worth of backlinks, so its second page still renders a "load
	// more" link whose fallback URL this test follows.
	//
	// [Ja] そのカードにネスト 3 ページ分のバックリンクを与える。2 ページ目でも「もっと見る」リンクが
	// 描画され、本テストがそのフォールバック URL を辿れるようにするためである。
	nestedCount := int(viewmodel.BacklinkLimit)*2 + 1
	for i := range nestedCount {
		testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(model.PageNumber(200 + i)).
			WithTitle(fmt.Sprintf("Later Parent Backlink %02d", i)).
			WithLinkedPageIDs([]model.PageID{linkedPageIDs[0]}).
			WithModifiedAt(baseTime.Add(time.Duration(200+i) * time.Hour)).
			Build()
	}

	spaceRepo := repository.NewSpaceRepository(queries)
	spaceMemberRepo := repository.NewSpaceMemberRepository(queries)
	pageRepo := repository.NewPageRepository(queries)
	topicRepo := repository.NewTopicRepository(queries)
	topicMemberRepo := repository.NewTopicMemberRepository(queries)
	draftPageRepo := repository.NewDraftPageRepository(queries)

	linkListHandler := pagelinklisthandler.NewHandler(usecase.NewGetLinkListUsecase(
		spaceRepo,
		spaceMemberRepo,
		pageRepo,
		topicRepo,
		topicMemberRepo,
		draftPageRepo,
	))
	backlinkListHandler := pagebacklinklisthandler.NewHandler(usecase.NewGetBacklinkListUsecase(
		spaceRepo,
		spaceMemberRepo,
		pageRepo,
		topicRepo,
		topicMemberRepo,
	))
	editHandler := setupEditHandler(t, queries)

	chiParams := map[string]string{
		"space_identifier": "later-parent",
		"page_number":      "1",
	}

	// 1. Load the editor, which seeds the shared state every later request sends.
	//
	// [Ja] 1. 編集画面を読み込み、以降の各リクエストが送る共有状態を用意する。
	editRequest := newShowRequest(t, "/s/later-parent/pages/1/edit", chiParams)
	editRequest = editRequest.WithContext(middleware.SetUserToContext(editRequest.Context(), &model.User{ID: userID}))
	editRecorder := httptest.NewRecorder()
	editHandler.Edit(editRecorder, editRequest)
	if editRecorder.Code != http.StatusOK {
		t.Fatalf("editor status = %d, want %d", editRecorder.Code, http.StatusOK)
	}
	state := applyRelatedPageState(t, editRecorder.Body.String(), nil)

	// 2. Advance the link list to its second page, the way its "load more" link does.
	//
	// [Ja] 2. 「もっと見る」リンクと同じ方法で、リンク一覧を 2 ページ目へ進める。
	linkFragmentURL := anchorAttributeByID(t, editRecorder.Body.String(), "page-link-list-load-more", "hx-get")
	linkPath, linkQuery := htmxRequestQuery(t, linkFragmentURL, state)
	linkResponse := callRelatedPageHandler(t, linkListHandler.Show, userID, linkPath, linkQuery, chiParams)
	state = applyRelatedPageState(t, linkResponse, state)
	if got := state.Get(viewmodel.LinkPageQueryParam); got != "2" {
		t.Fatalf("shared links_page = %q, want 2", got)
	}
	if got := state.Get(viewmodel.LinkedPageParentPageQueryParam); got != "" {
		t.Fatalf("shared linked_page_parent_page = %q, want it empty while no card is open", got)
	}

	// 3. Advance the nested listing of the card that page rendered.
	//
	// [Ja] 3. そのページが描画したカードのネスト一覧を進める。
	nestedFocusID := fmt.Sprintf("page-backlink-list-%d-load-more", secondPageCardNumber)
	nestedFragmentURL := anchorAttributeByID(t, linkResponse, nestedFocusID, "hx-get")
	nestedPath, nestedQuery := htmxRequestQuery(t, nestedFragmentURL, state)
	nestedResponse := callRelatedPageHandler(t, backlinkListHandler.Show, userID, nestedPath, nestedQuery,
		map[string]string{
			"space_identifier":   "later-parent",
			"page_number":        "1",
			"linked_page_number": fmt.Sprintf("%d", secondPageCardNumber),
		})

	// 4. Follow the response's full-page fallback as a viewer without JavaScript would.
	//
	// [Ja] 4. JavaScript の無い閲覧者と同じように、応答のフルページフォールバックを辿る。
	fallbackHref := anchorHrefByID(t, nestedResponse, nestedFocusID)
	fallbackURL, err := url.Parse(fallbackHref)
	if err != nil {
		t.Fatalf("parse fallback href %q: %v", fallbackHref, err)
	}
	if got := fallbackURL.Query().Get(viewmodel.LinkPageQueryParam); got != "2" {
		t.Errorf("fallback links_page = %q, want the card's own link-list page 2", got)
	}

	fallbackRequest := newShowRequest(t, fallbackURL.String(), chiParams)
	fallbackRequest.URL.RawQuery = fallbackURL.RawQuery
	fallbackRequest = fallbackRequest.WithContext(middleware.SetUserToContext(fallbackRequest.Context(), &model.User{ID: userID}))
	fallbackRecorder := httptest.NewRecorder()
	editHandler.Edit(fallbackRecorder, fallbackRequest)
	if fallbackRecorder.Code != http.StatusOK {
		t.Fatalf("editor status for %q = %d, want %d", fallbackHref, fallbackRecorder.Code, http.StatusOK)
	}

	fallbackBody := fallbackRecorder.Body.String()
	if !strings.Contains(fallbackBody, "Later Parent Link 00") {
		t.Error("the editor did not render the link-list page holding the selected card")
	}
	if !strings.Contains(fallbackBody, "Later Parent Backlink 00") {
		t.Error("the editor did not render the selected card's second nested backlink page")
	}
}
