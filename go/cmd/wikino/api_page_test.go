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
	apicurrentspacemember "github.com/wikinoapp/wikino/go/internal/apihandler/current_space_member"
	"github.com/wikinoapp/wikino/go/internal/apihandler/openapi_description"
	apipage "github.com/wikinoapp/wikino/go/internal/apihandler/page"
	apispace "github.com/wikinoapp/wikino/go/internal/apihandler/space"
	apitopic "github.com/wikinoapp/wikino/go/internal/apihandler/topic"
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
	for number := int32(1); number <= 3; number++ {
		testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).WithTopicID(topicID).WithNumber(model.PageNumber(number)).
			WithTitle(fmt.Sprintf("ページ%d", number)).
			WithModifiedAt(baseTime.Add(-time.Duration(number) * time.Hour)).
			Build()
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
		apicurrentspacemember.NewHandler(usecase.NewGetAPICurrentSpaceMemberUsecase()),
		openapi_description.NewHandler(api.OpenAPIDescription),
		apipage.NewHandler(
			usecase.NewListAPIPagesUsecase(pageRepo, topicRepo, topicMemberRepo),
			usecase.NewGetAPIPageUsecase(pageRepo, topicRepo, topicMemberRepo),
			nil,
			nil,
		),
		apispace.NewHandler(usecase.NewGetAPISpaceUsecase()),
		apitopic.NewHandler(usecase.NewListAPITopicsUsecase(nil, nil), usecase.NewGetAPITopicUsecase(nil, nil)),
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
		wantLink := "<" + base + "?cursor=" + url.QueryEscape(*body.NextCursor) + `&limit=2>; rel="next"`
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
		if body.Number != 2 || body.TopicNumber != 1 {
			t.Errorf("ページ = %+v、期待値 = トピック1のページ2", body)
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

	for _, target := range []string{base, base + "/2"} {
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

	t.Run("束縛先が違えば404", func(t *testing.T) {
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
		apicurrentspacemember.NewHandler(usecase.NewGetAPICurrentSpaceMemberUsecase()),
		openapi_description.NewHandler(api.OpenAPIDescription),
		apipage.NewHandler(
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
		apispace.NewHandler(usecase.NewGetAPISpaceUsecase()),
		apitopic.NewHandler(usecase.NewListAPITopicsUsecase(nil, nil), usecase.NewGetAPITopicUsecase(nil, nil)),
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
		if body.Number != 2 || body.TopicNumber != 1 || body.Title == nil || *body.Title != "新しいページ" {
			t.Errorf("ページ = %+v、期待値 = トピック1のページ2「新しいページ」", body)
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
		apicurrentspacemember.NewHandler(usecase.NewGetAPICurrentSpaceMemberUsecase()),
		openapi_description.NewHandler(api.OpenAPIDescription),
		apipage.NewHandler(
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
		apispace.NewHandler(usecase.NewGetAPISpaceUsecase()),
		apitopic.NewHandler(usecase.NewListAPITopicsUsecase(nil, nil), usecase.NewGetAPITopicUsecase(nil, nil)),
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
		if body.Number != 1 || body.Title == nil || *body.Title != "新しいタイトル" || body.Body != "新しい本文" {
			t.Errorf("ページ = %+v、期待値 = 更新したページ1", body)
		}
		etag := rr.Header().Get("ETag")
		if etag == before || etag != currentETag(t) {
			t.Errorf("ETag = %q、期待値 = 更新前 (%q) と異なり、取得と同じ値", etag, before)
		}

		// 更新前のETagを前提にした更新は、先の更新と競合するため412になる
		assertProblem(t, request("wkp_page_write", mergePatch, before, `{"body": "古い版を元にした本文"}`), http.StatusPreconditionFailed)
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
