package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/api"
	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/apihandler"
	apiattachment "github.com/wikinoapp/wikino/go/internal/apihandler/attachment"
	apicurrentspacemember "github.com/wikinoapp/wikino/go/internal/apihandler/current_space_member"
	apicurrenttoken "github.com/wikinoapp/wikino/go/internal/apihandler/current_token"
	apidraftpage "github.com/wikinoapp/wikino/go/internal/apihandler/draft_page"
	apidraftpagebodyedit "github.com/wikinoapp/wikino/go/internal/apihandler/draft_page_body_edit"
	apidraftpagepublication "github.com/wikinoapp/wikino/go/internal/apihandler/draft_page_publication"
	apidraftpagerevision "github.com/wikinoapp/wikino/go/internal/apihandler/draft_page_revision"
	"github.com/wikinoapp/wikino/go/internal/apihandler/openapi_description"
	apipage "github.com/wikinoapp/wikino/go/internal/apihandler/page"
	apipagebacklink "github.com/wikinoapp/wikino/go/internal/apihandler/page_backlink"
	apipagebodyedit "github.com/wikinoapp/wikino/go/internal/apihandler/page_body_edit"
	apipagelink "github.com/wikinoapp/wikino/go/internal/apihandler/page_link"
	apispace "github.com/wikinoapp/wikino/go/internal/apihandler/space"
	apitopic "github.com/wikinoapp/wikino/go/internal/apihandler/topic"
	apiwikilinkpreview "github.com/wikinoapp/wikino/go/internal/apihandler/wikilink_preview"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/query"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
	"github.com/wikinoapp/wikino/go/internal/validator"
)

// pageTestAuthenticatorはスコープの異なるテスト用トークンを受け付ける。
type pageTestAuthenticator struct {
	principal *model.APIPrincipal
}

func (a pageTestAuthenticator) Execute(_ context.Context, token string) (*model.APIPrincipal, error) {
	p := *a.principal
	switch token {
	case "wkp_page_read":
		p.Scopes = []model.Scope{model.ScopePageRead}
	case "wkp_page_write":
		// page:writeはpage:readを含意する
		p.Scopes = []model.Scope{model.ScopePageRead, model.ScopePageWrite}
	case "wkp_topic_read":
		p.Scopes = []model.Scope{model.ScopeTopicRead}
	default:
		return nil, nil
	}
	return &p, nil
}

func TestAPIRouter_Pages(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	userID := testutil.NewUserBuilder(t, tx).
		WithAtname("api_http_pages").WithEmail("api_http_pages@example.com").Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("api-http-pages").Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).WithUserID(userID).Build()
	topicID := testutil.NewTopicBuilder(t, tx).WithSpaceID(spaceID).WithNumber(1).WithName("トピック").Build()
	baseTime := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	// 番号の小さいページが大きいページすべてにリンクする (1は2・3に、2は3にリンクする)。
	// リンク先のIDを先に決めるため、番号の大きいページから作る
	var linkedPageIDs []model.PageID
	for number := int32(3); number >= 1; number-- {
		pageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).WithTopicID(topicID).WithNumber(model.PageNumber(number)).
			WithTitle(fmt.Sprintf("ページ%d", number)).
			WithLinkedPageIDs(linkedPageIDs).
			WithModifiedAt(baseTime.Add(-time.Duration(number) * time.Hour)).
			Build()
		linkedPageIDs = append([]model.PageID{pageID}, linkedPageIDs...)
	}

	principal := &model.APIPrincipal{
		User:        &model.User{ID: userID},
		Space:       &model.Space{ID: spaceID, Identifier: "api-http-pages"},
		SpaceMember: &model.SpaceMember{ID: spaceMemberID, SpaceID: spaceID, UserID: userID, Role: model.SpaceRoleAdmin, Active: true},
		TokenKind:   model.APITokenKindPersonalAccessToken,
	}
	q := testutil.QueriesWithTx(tx)
	pageRepo := repository.NewPageRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	server := apihandler.NewServer(
		apiattachment.NewHandler(nil),
		apicurrentspacemember.NewHandler(usecase.NewGetAPICurrentSpaceMemberUsecase()),
		apicurrenttoken.NewHandler(usecase.NewGetAPICurrentTokenUsecase()),
		apidraftpage.NewHandler("https://example.com", usecase.NewListAPIDraftPagesUsecase(nil, nil, nil), usecase.NewGetAPIDraftPageUsecase(nil, nil, nil, nil), nil, nil, nil),
		apidraftpagebodyedit.NewHandler("https://example.com", nil),
		apidraftpagepublication.NewHandler("https://example.com", nil),
		apidraftpagerevision.NewHandler(nil, nil),
		openapi_description.NewHandler(api.OpenAPIDescription),
		apipage.NewHandler(
			"https://example.com",
			usecase.NewListAPIPagesUsecase(pageRepo, topicRepo, topicMemberRepo),
			usecase.NewGetAPIPageUsecase(pageRepo, topicRepo, topicMemberRepo),
			nil,
			nil,
		),
		apipagebacklink.NewHandler("https://example.com", usecase.NewListAPIPageBacklinksUsecase(pageRepo, topicRepo, topicMemberRepo)),
		apipagebodyedit.NewHandler("https://example.com", nil),
		apipagelink.NewHandler("https://example.com", usecase.NewListAPIPageLinksUsecase(pageRepo, topicRepo, topicMemberRepo)),
		apispace.NewHandler(usecase.NewGetAPISpaceUsecase()),
		apitopic.NewHandler(usecase.NewListAPITopicsUsecase(nil, nil), usecase.NewGetAPITopicUsecase(nil, nil)),
		apiwikilinkpreview.NewHandler("https://example.com", nil),
	)
	r := newTestAPIHandlerWith(t, server, pageTestAuthenticator{principal: principal}, topicTestRateLimiter{})

	request := func(target, token string, header http.Header) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, target, nil)
		for k, v := range header {
			req.Header[k] = v
		}
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		return rr
	}
	base := "/api/v1/spaces/api-http-pages/pages"

	// サブテストは親のトランザクションを共有するため、並列にしない。
	t.Run("一覧の本文とLinkで次のページを辿れる", func(t *testing.T) {
		first := request(base+"?limit=2", "wkp_page_read", nil)
		if first.Code != http.StatusOK {
			t.Fatalf("ステータス = %d、期待値 = 200 (本文: %s)", first.Code, first.Body.String())
		}
		var body apigen.PageList
		if err := json.Unmarshal(first.Body.Bytes(), &body); err != nil {
			t.Fatalf("本文を読めない: %v", err)
		}
		if len(body.Items) != 2 || body.Items[0].Number != 1 || body.Items[1].Number != 2 || body.NextCursor == nil {
			t.Fatalf("一覧 = %+v、期待値 = 1・2と次のカーソル", body)
		}
		if want := "https://example.com/s/api-http-pages/pages/1"; body.Items[0].Url != want {
			t.Errorf("url = %q、期待値 = %q", body.Items[0].Url, want)
		}
		if body.Items[0].TopicName != "トピック" {
			t.Errorf("topic_name = %q、期待値 = %q", body.Items[0].TopicName, "トピック")
		}
		// 省略した `status` は、リクエスト検証がOpenAPI記述の既定値で補ってから次のページのURLに引き継ぐ
		wantLink := "<" + base + "?cursor=" + url.QueryEscape(*body.NextCursor) + `&limit=2&status=published>; rel="next"`
		if got := first.Header().Get("Link"); got != wantLink {
			t.Errorf("Link = %q、期待値 = %q", got, wantLink)
		}
		second := request(base+"?limit=2&cursor="+url.QueryEscape(*body.NextCursor), "wkp_page_read", nil)
		if second.Code != http.StatusOK {
			t.Fatalf("次ページのステータス = %d、期待値 = 200 (本文: %s)", second.Code, second.Body.String())
		}
		if err := json.Unmarshal(second.Body.Bytes(), &body); err != nil {
			t.Fatalf("次ページの本文を読めない: %v", err)
		}
		if len(body.Items) != 1 || body.Items[0].Number != 3 || body.NextCursor != nil || second.Header().Get("Link") != "" {
			t.Errorf("次ページ = %+v、Link = %q、期待値 = 3のみ・続きなし", body, second.Header().Get("Link"))
		}
	})

	t.Run("qで絞り込み、次のページのURLにqを引き継ぐ", func(t *testing.T) {
		q := "ページ\u3000ジ"
		first := request(base+"?limit=1&q="+url.QueryEscape(q), "wkp_page_read", nil)
		if first.Code != http.StatusOK {
			t.Fatalf("ステータス = %d、期待値 = 200 (本文: %s)", first.Code, first.Body.String())
		}
		var body apigen.PageList
		if err := json.Unmarshal(first.Body.Bytes(), &body); err != nil {
			t.Fatalf("本文を読めない: %v", err)
		}
		if len(body.Items) != 1 || body.Items[0].Number != 1 || body.NextCursor == nil {
			t.Fatalf("一覧 = %+v、期待値 = 1と次のカーソル", body)
		}
		wantLink := "<" + base + "?cursor=" + url.QueryEscape(*body.NextCursor) + "&limit=1&q=" + url.QueryEscape(q) + `&status=published>; rel="next"`
		if got := first.Header().Get("Link"); got != wantLink {
			t.Errorf("Link = %q、期待値 = %q", got, wantLink)
		}

		rr := request(base+"?q="+url.QueryEscape("ページ2"), "wkp_page_read", nil)
		if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
			t.Fatalf("本文を読めない: %v (本文: %s)", err, rr.Body.String())
		}
		if len(body.Items) != 1 || body.Items[0].Number != 2 {
			t.Errorf("一覧 = %+v、期待値 = 2のみ", body.Items)
		}
	})

	t.Run("日時はZ付きのUTCで返す", func(t *testing.T) {
		rr := request(base+"?limit=1", "wkp_page_read", nil)
		var raw struct {
			Items []struct {
				ModifiedAt string `json:"modified_at"`
			} `json:"items"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &raw); err != nil || len(raw.Items) != 1 {
			t.Fatalf("本文を読めない: %v (本文: %s)", err, rr.Body.String())
		}
		if want := "2026-09-01T11:00:00Z"; raw.Items[0].ModifiedAt != want {
			t.Errorf("modified_at = %q、期待値 = %q", raw.Items[0].ModifiedAt, want)
		}
	})

	t.Run("modified_sinceはオフセット付きの日時を受け付ける", func(t *testing.T) {
		// `2026-09-01T10:00:00Z` と同じ時点
		rr := request(base+"?modified_since="+url.QueryEscape("2026-09-01T19:00:00+09:00"), "wkp_page_read", nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("ステータス = %d、期待値 = 200 (本文: %s)", rr.Code, rr.Body.String())
		}
		var body apigen.PageList
		if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
			t.Fatalf("本文を読めない: %v", err)
		}
		if len(body.Items) != 2 || body.Items[0].Number != 1 || body.Items[1].Number != 2 {
			t.Errorf("一覧 = %+v、期待値 = 1・2", body.Items)
		}
	})

	for _, target := range []string{
		base + "?modified_since=yesterday",
		base + "?topic_number=0",
		base + "?cursor=invalid!",
		base + "?status=draft",
		base + "?title=",
		base + "?q=",
		base + "?q=" + url.QueryEscape(" \u3000"),
		base + "?q=" + strings.Repeat("a", 201),
		base + "?q=" + url.QueryEscape(strings.Repeat("a ", 11)),
	} {
		t.Run("不正なクエリは422: "+target, func(t *testing.T) {
			assertProblem(t, request(target, "wkp_page_read", nil), http.StatusUnprocessableEntity)
		})
	}

	t.Run("ページをETag付きで返し、If-None-Matchが一致すれば304", func(t *testing.T) {
		rr := request(base+"/2", "wkp_page_write", nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("ステータス = %d、期待値 = 200 (本文: %s)", rr.Code, rr.Body.String())
		}
		var body apigen.Page
		if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
			t.Fatalf("本文を読めない: %v", err)
		}
		if body.Number != 2 || body.TopicNumber != 1 || body.TopicName != "トピック" {
			t.Errorf("ページ = %+v、期待値 = トピック1「トピック」のページ2", body)
		}
		if want := "https://example.com/s/api-http-pages/pages/2"; body.Url != want {
			t.Errorf("url = %q、期待値 = %q", body.Url, want)
		}
		etag := rr.Header().Get("ETag")
		if etag == "" {
			t.Fatal("ETag = 空、期待値 = エンティティタグ")
		}

		notModified := request(base+"/2", "wkp_page_write", http.Header{"If-None-Match": {etag}})
		if notModified.Code != http.StatusNotModified {
			t.Fatalf("ステータス = %d、期待値 = 304 (本文: %s)", notModified.Code, notModified.Body.String())
		}
		if notModified.Body.Len() != 0 {
			t.Errorf("本文 = %q、期待値 = 空", notModified.Body.String())
		}
		if got := notModified.Header().Get("ETag"); got != etag {
			t.Errorf("ETag = %q、期待値 = %q", got, etag)
		}

		stale := request(base+"/2", "wkp_page_write", http.Header{"If-None-Match": {`"stale"`}})
		if stale.Code != http.StatusOK {
			t.Errorf("ステータス = %d、期待値 = 200", stale.Code)
		}
	})

	linkTests := []struct {
		name   string
		target string
		want   []int32
	}{
		{name: "リンク先の一覧をLinkで辿れる", target: base + "/1/links", want: []int32{2, 3}},
		{name: "バックリンクの一覧をLinkで辿れる", target: base + "/3/backlinks", want: []int32{1, 2}},
	}
	for _, tt := range linkTests {
		t.Run(tt.name, func(t *testing.T) {
			first := request(tt.target+"?limit=1", "wkp_page_read", nil)
			if first.Code != http.StatusOK {
				t.Fatalf("ステータス = %d、期待値 = 200 (本文: %s)", first.Code, first.Body.String())
			}
			var body apigen.PageList
			if err := json.Unmarshal(first.Body.Bytes(), &body); err != nil {
				t.Fatalf("本文を読めない: %v", err)
			}
			if len(body.Items) != 1 || body.Items[0].Number != tt.want[0] || body.NextCursor == nil {
				t.Fatalf("一覧 = %+v、期待値 = %dと次のカーソル", body, tt.want[0])
			}
			if want := fmt.Sprintf("https://example.com/s/api-http-pages/pages/%d", tt.want[0]); body.Items[0].Url != want {
				t.Errorf("url = %q、期待値 = %q", body.Items[0].Url, want)
			}
			if body.Items[0].TopicName != "トピック" {
				t.Errorf("topic_name = %q、期待値 = %q", body.Items[0].TopicName, "トピック")
			}
			wantLink := "<" + tt.target + "?cursor=" + url.QueryEscape(*body.NextCursor) + `&limit=1>; rel="next"`
			if got := first.Header().Get("Link"); got != wantLink {
				t.Errorf("Link = %q、期待値 = %q", got, wantLink)
			}

			second := request(tt.target+"?limit=1&cursor="+url.QueryEscape(*body.NextCursor), "wkp_page_read", nil)
			if second.Code != http.StatusOK {
				t.Fatalf("次ページのステータス = %d、期待値 = 200 (本文: %s)", second.Code, second.Body.String())
			}
			if err := json.Unmarshal(second.Body.Bytes(), &body); err != nil {
				t.Fatalf("次ページの本文を読めない: %v", err)
			}
			if len(body.Items) != 1 || body.Items[0].Number != tt.want[1] || body.Items[0].Status != apigen.PageStatusPublished || body.NextCursor != nil || second.Header().Get("Link") != "" {
				t.Errorf("次ページ = %+v (Link: %q)、期待値 = 公開済みの%dだけで続き無し", body, second.Header().Get("Link"), tt.want[1])
			}
		})
	}

	for _, target := range []string{base + "/1/links?cursor=invalid!", base + "/1/backlinks?limit=0"} {
		t.Run("リンク一覧の不正なクエリは422: "+target, func(t *testing.T) {
			assertProblem(t, request(target, "wkp_page_read", nil), http.StatusUnprocessableEntity)
		})
	}

	for _, target := range []string{base + "/99/links", base + "/99/backlinks"} {
		t.Run("存在しないページのリンク一覧は404: "+target, func(t *testing.T) {
			assertProblem(t, request(target, "wkp_page_read", nil), http.StatusNotFound)
		})
	}

	for _, target := range []string{base, base + "/2", base + "/1/links", base + "/1/backlinks"} {
		t.Run("page:readの無いトークンは403: "+target, func(t *testing.T) {
			rr := request(target, "wkp_topic_read", nil)
			assertProblem(t, rr, http.StatusForbidden)
			if got, want := rr.Header().Get("WWW-Authenticate"), `Bearer error="insufficient_scope", scope="page:read", resource_metadata="https://example.com/.well-known/oauth-protected-resource/api/v1/spaces/api-http-pages"`; got != want {
				t.Errorf("WWW-Authenticate = %q、期待値 = %q", got, want)
			}
		})
	}

	t.Run("存在しないページは404", func(t *testing.T) {
		assertProblem(t, request(base+"/99", "wkp_page_read", nil), http.StatusNotFound)
	})

	t.Run("トークンが属さないスペースなら404", func(t *testing.T) {
		assertProblem(t, request("/api/v1/spaces/other-space/pages/1", "wkp_page_read", nil), http.StatusNotFound)
		assertProblem(t, request("/api/v1/spaces/other-space/pages?cursor=invalid!", "wkp_page_read", nil), http.StatusNotFound)
	})
}

func TestAPIRouter_CreatePage(t *testing.T) {
	t.Parallel()

	// ページの作成はUseCaseが自前でトランザクションを張るため、フィクスチャをテストDBへコミットする
	db := testutil.GetTestDB()
	userID := testutil.NewUserBuilderDB(t, db).
		WithAtname("api_http_page_create").WithEmail("api_http_page_create@example.com").Build()
	spaceID := testutil.NewSpaceBuilderDB(t, db).WithIdentifier("api-http-page-create").Build()
	spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).WithUserID(userID).Build()
	topicID := testutil.NewTopicBuilderDB(t, db).WithSpaceID(spaceID).WithNumber(1).WithName("トピック").Build()
	testutil.NewPageBuilderDB(t, db).WithSpaceID(spaceID).WithTopicID(topicID).WithNumber(1).WithTitle("既存").Build()

	principal := &model.APIPrincipal{
		User:        &model.User{ID: userID},
		Space:       &model.Space{ID: spaceID, Identifier: "api-http-page-create"},
		SpaceMember: &model.SpaceMember{ID: spaceMemberID, SpaceID: spaceID, UserID: userID, Role: model.SpaceRoleAdmin, Active: true},
		TokenKind:   model.APITokenKindPersonalAccessToken,
	}
	q := query.New(db)
	pageRepo := repository.NewPageRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	server := apihandler.NewServer(
		apiattachment.NewHandler(nil),
		apicurrentspacemember.NewHandler(usecase.NewGetAPICurrentSpaceMemberUsecase()),
		apicurrenttoken.NewHandler(usecase.NewGetAPICurrentTokenUsecase()),
		apidraftpage.NewHandler("https://example.com", usecase.NewListAPIDraftPagesUsecase(nil, nil, nil), usecase.NewGetAPIDraftPageUsecase(nil, nil, nil, nil), nil, nil, nil),
		apidraftpagebodyedit.NewHandler("https://example.com", nil),
		apidraftpagepublication.NewHandler("https://example.com", nil),
		apidraftpagerevision.NewHandler(nil, nil),
		openapi_description.NewHandler(api.OpenAPIDescription),
		apipage.NewHandler(
			"https://example.com",
			usecase.NewListAPIPagesUsecase(pageRepo, topicRepo, topicMemberRepo),
			usecase.NewGetAPIPageUsecase(pageRepo, topicRepo, topicMemberRepo),
			usecase.NewCreateAPIPageUsecase(
				db,
				repository.NewSpaceRepository(q),
				pageRepo,
				repository.NewPageRevisionRepository(q),
				repository.NewPageEditorRepository(q),
				topicRepo,
				topicMemberRepo,
				repository.NewAttachmentRepository(q),
				repository.NewPageAttachmentReferenceRepository(q),
				validator.NewPageCreateValidator(pageRepo),
			),
			nil,
		),
		apipagebacklink.NewHandler("https://example.com", nil),
		apipagebodyedit.NewHandler("https://example.com", nil),
		apipagelink.NewHandler("https://example.com", nil),
		apispace.NewHandler(usecase.NewGetAPISpaceUsecase()),
		apitopic.NewHandler(usecase.NewListAPITopicsUsecase(nil, nil), usecase.NewGetAPITopicUsecase(nil, nil)),
		apiwikilinkpreview.NewHandler("https://example.com", nil),
	)
	r := newTestAPIHandlerWith(t, server, pageTestAuthenticator{principal: principal}, topicTestRateLimiter{})

	target := "/api/v1/spaces/api-http-page-create/pages"
	request := func(token, contentType, body string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		return rr
	}
	// assertFieldProblemは422の問題が、pointerの指すリクエスト本文の項目についてのものであることを確かめる
	assertFieldProblem := func(t *testing.T, rr *httptest.ResponseRecorder, pointer string) {
		t.Helper()
		assertProblem(t, rr, http.StatusUnprocessableEntity)
		var problem struct {
			Errors []struct {
				Pointer *string `json:"pointer"`
			} `json:"errors"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &problem); err != nil {
			t.Fatalf("本文を読めない: %v", err)
		}
		if len(problem.Errors) == 0 || problem.Errors[0].Pointer == nil || *problem.Errors[0].Pointer != pointer {
			t.Errorf("errors = %s、期待値 = pointerが %s の問題", rr.Body.String(), pointer)
		}
	}

	// サブテストは同じスペースにページを作るため、並列にしない。
	t.Run("ページを作成し、LocationとETagを付けて201を返す", func(t *testing.T) {
		rr := request("wkp_page_write", "application/json", `{"topic_number": 1, "title": "新しいページ", "body": "本文"}`)
		if rr.Code != http.StatusCreated {
			t.Fatalf("ステータス = %d、期待値 = 201 (本文: %s)", rr.Code, rr.Body.String())
		}
		if got := rr.Header().Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q、期待値 = application/json", got)
		}
		var body apigen.Page
		if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
			t.Fatalf("本文を読めない: %v", err)
		}
		if body.Number != 2 || body.TopicNumber != 1 || body.TopicName != "トピック" || body.Title == nil || *body.Title != "新しいページ" {
			t.Errorf("ページ = %+v、期待値 = トピック1「トピック」のページ2「新しいページ」", body)
		}
		if want := "https://example.com/s/api-http-page-create/pages/2"; body.Url != want {
			t.Errorf("url = %q、期待値 = %q", body.Url, want)
		}
		location := rr.Header().Get("Location")
		if want := target + "/2"; location != want {
			t.Errorf("Location = %q、期待値 = %q", location, want)
		}

		// LocationのページをETagで条件付き取得すると304になる
		req := httptest.NewRequest(http.MethodGet, location, nil)
		req.Header.Set("Authorization", "Bearer wkp_page_write")
		req.Header.Set("If-None-Match", rr.Header().Get("ETag"))
		got := httptest.NewRecorder()
		r.ServeHTTP(got, req)
		if got.Code != http.StatusNotModified {
			t.Errorf("作成時のETagでの取得のステータス = %d、期待値 = 304", got.Code)
		}
	})

	t.Run("page:writeの無いトークンは403", func(t *testing.T) {
		rr := request("wkp_page_read", "application/json", `{"topic_number": 1, "title": "スコープ不足", "body": ""}`)
		assertProblem(t, rr, http.StatusForbidden)
		if got, want := rr.Header().Get("WWW-Authenticate"), `Bearer error="insufficient_scope", scope="page:write", resource_metadata="https://example.com/.well-known/oauth-protected-resource/api/v1/spaces/api-http-page-create"`; got != want {
			t.Errorf("WWW-Authenticate = %q、期待値 = %q", got, want)
		}
	})

	t.Run("JSON以外の本文は415", func(t *testing.T) {
		rr := request("wkp_page_write", "text/plain", "本文")
		assertProblem(t, rr, http.StatusUnsupportedMediaType)
		if got := rr.Header().Get("Accept-Post"); got != "application/json" {
			t.Errorf("Accept-Post = %q、期待値 = application/json", got)
		}
	})

	t.Run("JSONとして読めない本文は400", func(t *testing.T) {
		assertProblem(t, request("wkp_page_write", "application/json", `{"topic_number": 1,`), http.StatusBadRequest)
	})

	t.Run("タイトルが無ければ422", func(t *testing.T) {
		assertProblem(t, request("wkp_page_write", "application/json", `{"topic_number": 1, "body": "本文"}`), http.StatusUnprocessableEntity)
	})

	t.Run("タイトルが空文字列なら422", func(t *testing.T) {
		assertFieldProblem(t, request("wkp_page_write", "application/json", `{"topic_number": 1, "title": "", "body": "本文"}`), "#/title")
	})

	t.Run("タイトルに使えない文字があれば422", func(t *testing.T) {
		assertFieldProblem(t, request("wkp_page_write", "application/json", `{"topic_number": 1, "title": "a/b", "body": "本文"}`), "#/title")
	})

	t.Run("トピック内で重複するタイトルは422", func(t *testing.T) {
		assertFieldProblem(t, request("wkp_page_write", "application/json", `{"topic_number": 1, "title": "既存", "body": "本文"}`), "#/title")
	})

	t.Run("存在しないトピックは422", func(t *testing.T) {
		assertFieldProblem(t, request("wkp_page_write", "application/json", `{"topic_number": 99, "title": "迷子", "body": "本文"}`), "#/topic_number")
	})
}

func TestAPIRouter_UpdatePage(t *testing.T) {
	t.Parallel()

	// ページの更新はUseCaseが自前でトランザクションを張るため、フィクスチャをテストDBへコミットする
	db := testutil.GetTestDB()
	userID := testutil.NewUserBuilderDB(t, db).
		WithAtname("api_http_page_update").WithEmail("api_http_page_update@example.com").Build()
	spaceID := testutil.NewSpaceBuilderDB(t, db).WithIdentifier("api-http-page-update").Build()
	spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).WithUserID(userID).Build()
	topicID := testutil.NewTopicBuilderDB(t, db).WithSpaceID(spaceID).WithNumber(1).WithName("トピック").Build()
	testutil.NewPageBuilderDB(t, db).WithSpaceID(spaceID).WithTopicID(topicID).WithNumber(1).WithTitle("タイトル").WithBody("本文").Build()
	testutil.NewPageBuilderDB(t, db).WithSpaceID(spaceID).WithTopicID(topicID).WithNumber(2).WithTitle("既存").Build()

	principal := &model.APIPrincipal{
		User:        &model.User{ID: userID},
		Space:       &model.Space{ID: spaceID, Identifier: "api-http-page-update"},
		SpaceMember: &model.SpaceMember{ID: spaceMemberID, SpaceID: spaceID, UserID: userID, Role: model.SpaceRoleAdmin, Active: true},
		TokenKind:   model.APITokenKindPersonalAccessToken,
	}
	q := query.New(db)
	pageRepo := repository.NewPageRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	server := apihandler.NewServer(
		apiattachment.NewHandler(nil),
		apicurrentspacemember.NewHandler(usecase.NewGetAPICurrentSpaceMemberUsecase()),
		apicurrenttoken.NewHandler(usecase.NewGetAPICurrentTokenUsecase()),
		apidraftpage.NewHandler("https://example.com", usecase.NewListAPIDraftPagesUsecase(nil, nil, nil), usecase.NewGetAPIDraftPageUsecase(nil, nil, nil, nil), nil, nil, nil),
		apidraftpagebodyedit.NewHandler("https://example.com", nil),
		apidraftpagepublication.NewHandler("https://example.com", nil),
		apidraftpagerevision.NewHandler(nil, nil),
		openapi_description.NewHandler(api.OpenAPIDescription),
		apipage.NewHandler(
			"https://example.com",
			usecase.NewListAPIPagesUsecase(pageRepo, topicRepo, topicMemberRepo),
			usecase.NewGetAPIPageUsecase(pageRepo, topicRepo, topicMemberRepo),
			nil,
			usecase.NewUpdateAPIPageUsecase(
				db,
				repository.NewSpaceRepository(q),
				pageRepo,
				repository.NewPageRevisionRepository(q),
				repository.NewPageEditorRepository(q),
				topicRepo,
				topicMemberRepo,
				repository.NewAttachmentRepository(q),
				repository.NewPageAttachmentReferenceRepository(q),
				validator.NewAPIPageUpdateValidator(pageRepo),
			),
		),
		apipagebacklink.NewHandler("https://example.com", nil),
		apipagebodyedit.NewHandler("https://example.com", nil),
		apipagelink.NewHandler("https://example.com", nil),
		apispace.NewHandler(usecase.NewGetAPISpaceUsecase()),
		apitopic.NewHandler(usecase.NewListAPITopicsUsecase(nil, nil), usecase.NewGetAPITopicUsecase(nil, nil)),
		apiwikilinkpreview.NewHandler("https://example.com", nil),
	)
	r := newTestAPIHandlerWith(t, server, pageTestAuthenticator{principal: principal}, topicTestRateLimiter{})

	target := "/api/v1/spaces/api-http-page-update/pages/1"
	const mergePatch = "application/merge-patch+json"
	// currentETagはページを取得し、応答のETagを返す
	currentETag := func(t *testing.T) string {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, target, nil)
		req.Header.Set("Authorization", "Bearer wkp_page_read")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("取得のステータス = %d、期待値 = 200 (本文: %s)", rr.Code, rr.Body.String())
		}
		return rr.Header().Get("ETag")
	}
	request := func(token, contentType, ifMatch, body string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPatch, target, strings.NewReader(body))
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("Authorization", "Bearer "+token)
		if ifMatch != "" {
			req.Header.Set("If-Match", ifMatch)
		}
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		return rr
	}
	// assertFieldProblemは422の問題が、pointerの指すリクエスト本文の項目についてのものであることを確かめる
	assertFieldProblem := func(t *testing.T, rr *httptest.ResponseRecorder, pointer string) {
		t.Helper()
		assertProblem(t, rr, http.StatusUnprocessableEntity)
		var problem struct {
			Errors []struct {
				Pointer *string `json:"pointer"`
			} `json:"errors"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &problem); err != nil {
			t.Fatalf("本文を読めない: %v", err)
		}
		if len(problem.Errors) == 0 || problem.Errors[0].Pointer == nil || *problem.Errors[0].Pointer != pointer {
			t.Errorf("errors = %s、期待値 = pointerが %s の問題", rr.Body.String(), pointer)
		}
	}

	// サブテストは同じページを更新するため、並列にしない。
	t.Run("ページを更新し、新しいETagを付けて200を返す", func(t *testing.T) {
		before := currentETag(t)
		rr := request("wkp_page_write", mergePatch, before, `{"title": "新しいタイトル", "body": "新しい本文"}`)
		if rr.Code != http.StatusOK {
			t.Fatalf("ステータス = %d、期待値 = 200 (本文: %s)", rr.Code, rr.Body.String())
		}
		if got := rr.Header().Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q、期待値 = application/json", got)
		}
		var body apigen.Page
		if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
			t.Fatalf("本文を読めない: %v", err)
		}
		if body.Number != 1 || body.TopicName != "トピック" || body.Title == nil || *body.Title != "新しいタイトル" || body.Body != "新しい本文" {
			t.Errorf("ページ = %+v、期待値 = トピック「トピック」の更新したページ1", body)
		}
		if want := "https://example.com/s/api-http-page-update/pages/1"; body.Url != want {
			t.Errorf("url = %q、期待値 = %q", body.Url, want)
		}
		etag := rr.Header().Get("ETag")
		if etag == before || etag != currentETag(t) {
			t.Errorf("ETag = %q、期待値 = 更新前 (%q) と異なり、取得と同じ値", etag, before)
		}

		// 更新前のETagを前提にした更新は、先の更新と競合するため412になる
		assertProblem(t, request("wkp_page_write", mergePatch, before, `{"body": "古い版を元にした本文"}`), http.StatusPreconditionFailed)
	})

	t.Run("トピック名を変えてもETagは変わらず、変える前のETagで更新できる", func(t *testing.T) {
		before := currentETag(t)
		if _, err := topicRepo.UpdateNameAndDescription(t.Context(), repository.UpdateTopicNameAndDescriptionInput{
			ID: topicID, SpaceID: spaceID, Name: "改名したトピック",
		}); err != nil {
			t.Fatalf("トピック名を変えられない: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, target, nil)
		req.Header.Set("Authorization", "Bearer wkp_page_read")
		got := httptest.NewRecorder()
		r.ServeHTTP(got, req)
		var page apigen.Page
		if err := json.Unmarshal(got.Body.Bytes(), &page); err != nil {
			t.Fatalf("本文を読めない: %v (本文: %s)", err, got.Body.String())
		}
		if page.TopicName != "改名したトピック" {
			t.Errorf("topic_name = %q、期待値 = %q", page.TopicName, "改名したトピック")
		}
		if etag := got.Header().Get("ETag"); etag != before {
			t.Errorf("ETag = %q、期待値 = トピック名を変える前と同じ %q", etag, before)
		}

		rr := request("wkp_page_write", mergePatch, before, `{"body": "改名後の本文"}`)
		if rr.Code != http.StatusOK {
			t.Fatalf("ステータス = %d、期待値 = 200 (本文: %s)", rr.Code, rr.Body.String())
		}
		var body apigen.Page
		if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
			t.Fatalf("本文を読めない: %v", err)
		}
		if body.TopicName != "改名したトピック" {
			t.Errorf("topic_name = %q、期待値 = %q", body.TopicName, "改名したトピック")
		}
	})

	t.Run("If-Matchが無ければ428", func(t *testing.T) {
		assertProblem(t, request("wkp_page_write", mergePatch, "", `{"body": "本文"}`), http.StatusPreconditionRequired)
	})

	t.Run("page:writeの無いトークンは403", func(t *testing.T) {
		rr := request("wkp_page_read", mergePatch, currentETag(t), `{"body": "スコープ不足"}`)
		assertProblem(t, rr, http.StatusForbidden)
		if got, want := rr.Header().Get("WWW-Authenticate"), `Bearer error="insufficient_scope", scope="page:write", resource_metadata="https://example.com/.well-known/oauth-protected-resource/api/v1/spaces/api-http-page-update"`; got != want {
			t.Errorf("WWW-Authenticate = %q、期待値 = %q", got, want)
		}
	})

	t.Run("JSON Merge Patch以外の本文は415", func(t *testing.T) {
		rr := request("wkp_page_write", "application/json", currentETag(t), `{"body": "本文"}`)
		assertProblem(t, rr, http.StatusUnsupportedMediaType)
		if got := rr.Header().Get("Accept-Patch"); got != mergePatch {
			t.Errorf("Accept-Patch = %q、期待値 = %s", got, mergePatch)
		}
	})

	t.Run("JSONとして読めない本文は400", func(t *testing.T) {
		assertProblem(t, request("wkp_page_write", mergePatch, currentETag(t), `{"body":`), http.StatusBadRequest)
	})

	t.Run("タイトルをnullにする更新は422", func(t *testing.T) {
		assertFieldProblem(t, request("wkp_page_write", mergePatch, currentETag(t), `{"title": null}`), "#/title")
	})

	t.Run("定義していない項目を含む更新は422", func(t *testing.T) {
		assertProblem(t, request("wkp_page_write", mergePatch, currentETag(t), `{"topic_number": 2}`), http.StatusUnprocessableEntity)
	})

	t.Run("トピック内で重複するタイトルは422", func(t *testing.T) {
		assertFieldProblem(t, request("wkp_page_write", mergePatch, currentETag(t), `{"title": "既存"}`), "#/title")
	})

	t.Run("存在しないページは404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/spaces/api-http-page-update/pages/99", strings.NewReader(`{"body": "本文"}`))
		req.Header.Set("Content-Type", mergePatch)
		req.Header.Set("Authorization", "Bearer wkp_page_write")
		req.Header.Set("If-Match", "*")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		assertProblem(t, rr, http.StatusNotFound)
	})
}
