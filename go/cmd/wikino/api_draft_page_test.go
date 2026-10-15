package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

// draftPageTestAuthenticatorはスコープの異なるテスト用トークンを受け付ける。
type draftPageTestAuthenticator struct {
	principal *model.APIPrincipal
}

func (a draftPageTestAuthenticator) Execute(_ context.Context, token string) (*model.APIPrincipal, error) {
	p := *a.principal
	switch token {
	case "wkp_draft_page_read":
		p.Scopes = []model.Scope{model.ScopeDraftPageRead}
	case "wkp_draft_page_write":
		// draft_page:writeはdraft_page:readを含意する
		p.Scopes = []model.Scope{model.ScopeDraftPageRead, model.ScopeDraftPageWrite}
	case "wkp_page_write":
		p.Scopes = []model.Scope{model.ScopePageRead, model.ScopePageWrite}
	case "wkp_draft_page_delete":
		p.Scopes = []model.Scope{model.ScopeDraftPageDelete}
	case "wkp_draft_page_and_page_write":
		p.Scopes = []model.Scope{model.ScopeDraftPageRead, model.ScopeDraftPageWrite, model.ScopePageRead, model.ScopePageWrite}
	default:
		return nil, nil
	}
	return &p, nil
}

func TestAPIRouter_DraftPages(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	userID := testutil.NewUserBuilder(t, tx).
		WithAtname("api_http_drafts").WithEmail("api_http_drafts@example.com").Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("api-http-drafts").Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).WithUserID(userID).Build()
	topicID := testutil.NewTopicBuilder(t, tx).WithSpaceID(spaceID).WithNumber(1).WithName("トピック").Build()
	pageID := testutil.NewPageBuilder(t, tx).
		WithSpaceID(spaceID).WithTopicID(topicID).WithNumber(1).WithTitle("ページ1").Build()
	testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).WithPageID(pageID).WithSpaceMemberID(spaceMemberID).WithTopicID(topicID).
		WithTitle("下書き1").WithBody("下書きの本文").
		WithModifiedAt(time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)).Build()

	principal := &model.APIPrincipal{
		User:        &model.User{ID: userID},
		Space:       &model.Space{ID: spaceID, Identifier: "api-http-drafts"},
		SpaceMember: &model.SpaceMember{ID: spaceMemberID, SpaceID: spaceID, UserID: userID, Role: model.SpaceRoleAdmin, Active: true},
		TokenKind:   model.APITokenKindPersonalAccessToken,
	}
	q := testutil.QueriesWithTx(tx)
	pageRepo := repository.NewPageRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	server := apihandler.NewServer(
		apiattachment.NewHandler(nil),
		apicurrentspacemember.NewHandler(usecase.NewGetAPICurrentSpaceMemberUsecase()),
		apicurrenttoken.NewHandler(usecase.NewGetAPICurrentTokenUsecase()),
		apidraftpage.NewHandler(
			"https://example.com",
			usecase.NewListAPIDraftPagesUsecase(draftPageRepo, topicRepo, topicMemberRepo),
			usecase.NewGetAPIDraftPageUsecase(pageRepo, draftPageRepo, topicRepo, topicMemberRepo),
			nil,
			nil,
			nil,
		),
		apidraftpagebodyedit.NewHandler("https://example.com", nil),
		apidraftpagepublication.NewHandler("https://example.com", nil),
		apidraftpagerevision.NewHandler(nil, nil),
		openapi_description.NewHandler(api.OpenAPIDescription),
		apipage.NewHandler("https://example.com", usecase.NewListAPIPagesUsecase(nil, nil, nil), usecase.NewGetAPIPageUsecase(nil, nil, nil), nil, nil),
		apipagebacklink.NewHandler("https://example.com", nil),
		apipagebodyedit.NewHandler("https://example.com", nil),
		apipagelink.NewHandler("https://example.com", nil),
		apispace.NewHandler(usecase.NewGetAPISpaceUsecase()),
		apitopic.NewHandler(usecase.NewListAPITopicsUsecase(nil, nil), usecase.NewGetAPITopicUsecase(nil, nil)),
		apiwikilinkpreview.NewHandler("https://example.com", nil),
	)
	r := newTestAPIHandlerWith(t, server, draftPageTestAuthenticator{principal: principal}, topicTestRateLimiter{})

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
	listPath := "/api/v1/spaces/api-http-drafts/draft_pages"
	showPath := "/api/v1/spaces/api-http-drafts/pages/1/draft_page"

	// サブテストは親のトランザクションを共有するため、並列にしない。
	t.Run("下書きの一覧を返す", func(t *testing.T) {
		rr := request(listPath, "wkp_draft_page_write", nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("ステータス = %d、期待値 = 200 (本文: %s)", rr.Code, rr.Body.String())
		}
		var body apigen.DraftPageList
		if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
			t.Fatalf("本文を読めない: %v", err)
		}
		if len(body.Items) != 1 || body.Items[0].PageNumber != 1 || body.NextCursor != nil {
			t.Errorf("一覧 = %+v、期待値 = ページ1の下書きのみ", body)
		}
		if want := "https://example.com/s/api-http-drafts/pages/1/edit"; len(body.Items) == 1 && body.Items[0].Url != want {
			t.Errorf("url = %q、期待値 = 編集画面の %q", body.Items[0].Url, want)
		}
	})

	t.Run("下書きをETag付きで返し、If-None-Matchが一致すれば304", func(t *testing.T) {
		rr := request(showPath, "wkp_draft_page_read", nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("ステータス = %d、期待値 = 200 (本文: %s)", rr.Code, rr.Body.String())
		}
		var raw map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &raw); err != nil {
			t.Fatalf("本文を読めない: %v", err)
		}
		if raw["page_number"] != float64(1) || raw["topic_number"] != float64(1) || raw["title"] != "下書き1" || raw["modified_at"] != "2026-09-01T12:00:00Z" {
			t.Errorf("下書き = %v", raw)
		}
		if want := "https://example.com/s/api-http-drafts/pages/1/edit"; raw["url"] != want {
			t.Errorf("url = %v、期待値 = 編集画面の %q", raw["url"], want)
		}
		etag := rr.Header().Get("ETag")
		if etag == "" {
			t.Fatal("ETag = 空、期待値 = エンティティタグ")
		}

		notModified := request(showPath, "wkp_draft_page_read", http.Header{"If-None-Match": {etag}})
		if notModified.Code != http.StatusNotModified {
			t.Fatalf("ステータス = %d、期待値 = 304 (本文: %s)", notModified.Code, notModified.Body.String())
		}
		if notModified.Body.Len() != 0 {
			t.Errorf("本文 = %q、期待値 = 空", notModified.Body.String())
		}
		if got := notModified.Header().Get("ETag"); got != etag {
			t.Errorf("ETag = %q、期待値 = %q", got, etag)
		}
	})

	for _, target := range []string{listPath, showPath} {
		t.Run("draft_page:readの無いトークンは403: "+target, func(t *testing.T) {
			rr := request(target, "wkp_page_write", nil)
			assertProblem(t, rr, http.StatusForbidden)
			if got, want := rr.Header().Get("WWW-Authenticate"), `Bearer error="insufficient_scope", scope="draft_page:read", resource_metadata="https://example.com/.well-known/oauth-protected-resource/api/v1/spaces/api-http-drafts"`; got != want {
				t.Errorf("WWW-Authenticate = %q、期待値 = %q", got, want)
			}
		})
	}

	t.Run("下書きの無いページは404", func(t *testing.T) {
		assertProblem(t, request("/api/v1/spaces/api-http-drafts/pages/99/draft_page", "wkp_draft_page_read", nil), http.StatusNotFound)
	})

	t.Run("不正なカーソルは422", func(t *testing.T) {
		assertProblem(t, request(listPath+"?cursor=invalid!", "wkp_draft_page_read", nil), http.StatusUnprocessableEntity)
	})

	t.Run("トークンが属さないスペースなら404", func(t *testing.T) {
		assertProblem(t, request("/api/v1/spaces/other-space/draft_pages", "wkp_draft_page_read", nil), http.StatusNotFound)
		assertProblem(t, request("/api/v1/spaces/other-space/pages/1/draft_page", "wkp_draft_page_read", nil), http.StatusNotFound)
	})
}

func TestAPIRouter_CreateDraftPage(t *testing.T) {
	t.Parallel()

	// 下書きの作成はUseCaseが自前でトランザクションを張るため、フィクスチャをテストDBへコミットする
	db := testutil.GetTestDB()
	userID := testutil.NewUserBuilderDB(t, db).
		WithAtname("api_http_draft_create").WithEmail("api_http_draft_create@example.com").Build()
	spaceID := testutil.NewSpaceBuilderDB(t, db).WithIdentifier("api-http-draft-create").Build()
	spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).WithUserID(userID).Build()
	topicID := testutil.NewTopicBuilderDB(t, db).WithSpaceID(spaceID).WithNumber(1).WithName("トピック").Build()
	testutil.NewPageBuilderDB(t, db).WithSpaceID(spaceID).WithTopicID(topicID).WithNumber(1).WithTitle("既存").WithBody("既存の本文").Build()

	principal := &model.APIPrincipal{
		User:        &model.User{ID: userID},
		Space:       &model.Space{ID: spaceID, Identifier: "api-http-draft-create"},
		SpaceMember: &model.SpaceMember{ID: spaceMemberID, SpaceID: spaceID, UserID: userID, Role: model.SpaceRoleAdmin, Active: true},
		TokenKind:   model.APITokenKindPersonalAccessToken,
	}
	q := query.New(db)
	pageRepo := repository.NewPageRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	server := apihandler.NewServer(
		apiattachment.NewHandler(nil),
		apicurrentspacemember.NewHandler(usecase.NewGetAPICurrentSpaceMemberUsecase()),
		apicurrenttoken.NewHandler(usecase.NewGetAPICurrentTokenUsecase()),
		apidraftpage.NewHandler(
			"https://example.com",
			usecase.NewListAPIDraftPagesUsecase(draftPageRepo, topicRepo, topicMemberRepo),
			usecase.NewGetAPIDraftPageUsecase(pageRepo, draftPageRepo, topicRepo, topicMemberRepo),
			usecase.NewCreateAPIDraftPageUsecase(
				db,
				repository.NewSpaceRepository(q),
				pageRepo,
				repository.NewPageEditorRepository(q),
				draftPageRepo,
				repository.NewDraftPageRevisionRepository(q),
				topicRepo,
				topicMemberRepo,
				repository.NewAttachmentRepository(q),
				validator.NewAPIDraftPageCreateValidator(),
			),
			nil,
			nil,
		),
		apidraftpagebodyedit.NewHandler("https://example.com", nil),
		apidraftpagepublication.NewHandler("https://example.com", nil),
		apidraftpagerevision.NewHandler(nil, nil),
		openapi_description.NewHandler(api.OpenAPIDescription),
		apipage.NewHandler("https://example.com", usecase.NewListAPIPagesUsecase(nil, nil, nil), usecase.NewGetAPIPageUsecase(nil, nil, nil), nil, nil),
		apipagebacklink.NewHandler("https://example.com", nil),
		apipagebodyedit.NewHandler("https://example.com", nil),
		apipagelink.NewHandler("https://example.com", nil),
		apispace.NewHandler(usecase.NewGetAPISpaceUsecase()),
		apitopic.NewHandler(usecase.NewListAPITopicsUsecase(nil, nil), usecase.NewGetAPITopicUsecase(nil, nil)),
		apiwikilinkpreview.NewHandler("https://example.com", nil),
	)
	r := newTestAPIHandlerWith(t, server, draftPageTestAuthenticator{principal: principal}, topicTestRateLimiter{})

	target := "/api/v1/spaces/api-http-draft-create/draft_pages"
	request := func(token, body string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		return rr
	}
	// problemOfは問題の本文から、問題タイプとフィールド単位の問題の箇所を読む
	problemOf := func(t *testing.T, rr *httptest.ResponseRecorder) (string, []*string) {
		t.Helper()
		var problem struct {
			Type   string `json:"type"`
			Errors []struct {
				Pointer *string `json:"pointer"`
			} `json:"errors"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &problem); err != nil {
			t.Fatalf("本文を読めない: %v", err)
		}
		pointers := make([]*string, 0, len(problem.Errors))
		for _, e := range problem.Errors {
			pointers = append(pointers, e.Pointer)
		}
		return problem.Type, pointers
	}

	// サブテストは同じスペースに下書きを作るため、並列にしない。
	t.Run("既存のページの下書きを作成し、LocationとETagを付けて201を返す", func(t *testing.T) {
		rr := request("wkp_draft_page_write", `{"page_number": 1, "body": "書き換えた本文"}`)
		if rr.Code != http.StatusCreated {
			t.Fatalf("ステータス = %d、期待値 = 201 (本文: %s)", rr.Code, rr.Body.String())
		}
		var body apigen.DraftPage
		if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
			t.Fatalf("本文を読めない: %v", err)
		}
		if body.PageNumber != 1 || body.TopicNumber != 1 || body.Title == nil || *body.Title != "既存" || body.Body != "書き換えた本文" {
			t.Errorf("下書き = %+v、期待値 = ページ1のタイトル「既存」と指定した本文", body)
		}
		if want := "https://example.com/s/api-http-draft-create/pages/1/edit"; body.Url != want {
			t.Errorf("url = %q、期待値 = 編集画面の %q", body.Url, want)
		}
		location := rr.Header().Get("Location")
		if want := "/api/v1/spaces/api-http-draft-create/pages/1/draft_page"; location != want {
			t.Errorf("Location = %q、期待値 = %q", location, want)
		}

		// Locationの下書きを作成時のETagで条件付き取得すると304になる
		req := httptest.NewRequest(http.MethodGet, location, nil)
		req.Header.Set("Authorization", "Bearer wkp_draft_page_read")
		req.Header.Set("If-None-Match", rr.Header().Get("ETag"))
		got := httptest.NewRecorder()
		r.ServeHTTP(got, req)
		if got.Code != http.StatusNotModified {
			t.Errorf("作成時のETagでの取得のステータス = %d、期待値 = 304", got.Code)
		}
	})

	t.Run("下書きが既にあれば409でalready-existsの問題を返す", func(t *testing.T) {
		rr := request("wkp_draft_page_write", `{"page_number": 1}`)
		assertProblem(t, rr, http.StatusConflict)
		if typ, _ := problemOf(t, rr); typ != "https://example.com/api/problems/already-exists" {
			t.Errorf("type = %q、期待値 = already-existsの問題タイプ", typ)
		}
	})

	t.Run("新しいページの下書きを作成する", func(t *testing.T) {
		rr := request("wkp_draft_page_and_page_write", `{"topic_number": 1, "title": "新しいページ"}`)
		if rr.Code != http.StatusCreated {
			t.Fatalf("ステータス = %d、期待値 = 201 (本文: %s)", rr.Code, rr.Body.String())
		}
		var body apigen.DraftPage
		if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
			t.Fatalf("本文を読めない: %v", err)
		}
		if body.PageNumber != 2 || body.Title == nil || *body.Title != "新しいページ" || body.Body != "" {
			t.Errorf("下書き = %+v、期待値 = ページ2の「新しいページ」", body)
		}
		if want := "https://example.com/s/api-http-draft-create/pages/2/edit"; body.Url != want {
			t.Errorf("url = %q、期待値 = 編集画面の %q", body.Url, want)
		}
	})

	t.Run("page:writeの無いトークンでは新しいページの下書きは404", func(t *testing.T) {
		assertProblem(t, request("wkp_draft_page_write", `{"topic_number": 1}`), http.StatusNotFound)
	})

	t.Run("draft_page:writeの無いトークンは403", func(t *testing.T) {
		rr := request("wkp_page_write", `{"topic_number": 1}`)
		assertProblem(t, rr, http.StatusForbidden)
		if got, want := rr.Header().Get("WWW-Authenticate"), `Bearer error="insufficient_scope", scope="draft_page:write", resource_metadata="https://example.com/.well-known/oauth-protected-resource/api/v1/spaces/api-http-draft-create"`; got != want {
			t.Errorf("WWW-Authenticate = %q、期待値 = %q", got, want)
		}
	})

	pointerTests := []struct {
		name         string
		body         string
		wantPointers []string
	}{
		{name: "ページとトピックの両方を指定すると422", body: `{"page_number": 1, "topic_number": 1}`, wantPointers: []string{"#/page_number", "#/topic_number"}},
		{name: "ページもトピックも無ければ422", body: `{"title": "迷子"}`, wantPointers: []string{""}},
		{name: "存在しないページは422", body: `{"page_number": 99}`, wantPointers: []string{"#/page_number"}},
		{name: "存在しないトピックは422", body: `{"topic_number": 99}`, wantPointers: []string{"#/topic_number"}},
		{name: "空のタイトルは422", body: `{"page_number": 1, "title": ""}`, wantPointers: []string{"#/title"}},
	}
	for _, tt := range pointerTests {
		t.Run(tt.name, func(t *testing.T) {
			rr := request("wkp_draft_page_and_page_write", tt.body)
			assertProblem(t, rr, http.StatusUnprocessableEntity)
			_, pointers := problemOf(t, rr)
			got := make([]string, 0, len(pointers))
			for _, p := range pointers {
				if p == nil {
					got = append(got, "")
					continue
				}
				got = append(got, *p)
			}
			if strings.Join(got, ",") != strings.Join(tt.wantPointers, ",") {
				t.Errorf("問題の箇所 = %v、期待値 = %v (本文: %s)", got, tt.wantPointers, rr.Body.String())
			}
		})
	}
}

// newDraftPageWriteTestRouterは、下書きの更新・削除のテスト用に、トークンの持ち主の下書きを持つページ1と、
// 下書きの無いページ2のあるスペースを作り、APIのルーターを返す。
// 下書きの更新・削除はUseCaseが自前でトランザクションを張るため、フィクスチャをテストDBへコミットする
func newDraftPageWriteTestRouter(t *testing.T, identifier string) http.Handler {
	t.Helper()

	db := testutil.GetTestDB()
	atname := strings.ReplaceAll(identifier, "-", "_")
	userID := testutil.NewUserBuilderDB(t, db).WithAtname(atname).WithEmail(atname + "@example.com").Build()
	spaceID := testutil.NewSpaceBuilderDB(t, db).WithIdentifier(identifier).Build()
	spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).WithSpaceID(spaceID).WithUserID(userID).Build()
	topicID := testutil.NewTopicBuilderDB(t, db).WithSpaceID(spaceID).WithNumber(1).WithName("トピック").Build()
	pageID := testutil.NewPageBuilderDB(t, db).WithSpaceID(spaceID).WithTopicID(topicID).WithNumber(1).WithTitle("ページ").Build()
	testutil.NewPageBuilderDB(t, db).WithSpaceID(spaceID).WithTopicID(topicID).WithNumber(2).WithTitle("下書きの無いページ").Build()
	testutil.NewDraftPageBuilderDB(t, db).
		WithSpaceID(spaceID).WithPageID(pageID).WithSpaceMemberID(spaceMemberID).WithTopicID(topicID).
		WithTitle("下書き").WithBody("下書きの本文").Build()

	principal := &model.APIPrincipal{
		User:        &model.User{ID: userID},
		Space:       &model.Space{ID: spaceID, Identifier: model.SpaceIdentifier(identifier)},
		SpaceMember: &model.SpaceMember{ID: spaceMemberID, SpaceID: spaceID, UserID: userID, Role: model.SpaceRoleAdmin, Active: true},
		TokenKind:   model.APITokenKindPersonalAccessToken,
	}
	q := query.New(db)
	spaceRepo := repository.NewSpaceRepository(q)
	pageRepo := repository.NewPageRepository(q)
	pageEditorRepo := repository.NewPageEditorRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	draftPageRevisionRepo := repository.NewDraftPageRevisionRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	server := apihandler.NewServer(
		apiattachment.NewHandler(nil),
		apicurrentspacemember.NewHandler(usecase.NewGetAPICurrentSpaceMemberUsecase()),
		apicurrenttoken.NewHandler(usecase.NewGetAPICurrentTokenUsecase()),
		apidraftpage.NewHandler(
			"https://example.com",
			usecase.NewListAPIDraftPagesUsecase(draftPageRepo, topicRepo, topicMemberRepo),
			usecase.NewGetAPIDraftPageUsecase(pageRepo, draftPageRepo, topicRepo, topicMemberRepo),
			nil,
			usecase.NewUpdateAPIDraftPageUsecase(db, spaceRepo, pageRepo, pageEditorRepo, draftPageRepo, draftPageRevisionRepo, topicRepo, topicMemberRepo, repository.NewAttachmentRepository(q)),
			usecase.NewDeleteAPIDraftPageUsecase(db, pageRepo, draftPageRepo, draftPageRevisionRepo, topicRepo, topicMemberRepo),
		),
		apidraftpagebodyedit.NewHandler("https://example.com", usecase.NewCreateAPIDraftPageBodyEditUsecase(
			db, spaceRepo, pageRepo, pageEditorRepo, draftPageRepo, draftPageRevisionRepo, topicRepo, topicMemberRepo, repository.NewAttachmentRepository(q),
		)),
		apidraftpagepublication.NewHandler("https://example.com", usecase.NewCreateAPIDraftPagePublicationUsecase(
			db, spaceRepo, pageRepo, repository.NewPageRevisionRepository(q), pageEditorRepo, draftPageRepo, draftPageRevisionRepo,
			topicRepo, topicMemberRepo, repository.NewAttachmentRepository(q), repository.NewPageAttachmentReferenceRepository(q),
			validator.NewAPIPageUpdateValidator(pageRepo),
		)),
		apidraftpagerevision.NewHandler(nil, nil),
		openapi_description.NewHandler(api.OpenAPIDescription),
		apipage.NewHandler("https://example.com", usecase.NewListAPIPagesUsecase(nil, nil, nil), usecase.NewGetAPIPageUsecase(nil, nil, nil), nil, nil),
		apipagebacklink.NewHandler("https://example.com", nil),
		apipagebodyedit.NewHandler("https://example.com", nil),
		apipagelink.NewHandler("https://example.com", nil),
		apispace.NewHandler(usecase.NewGetAPISpaceUsecase()),
		apitopic.NewHandler(usecase.NewListAPITopicsUsecase(nil, nil), usecase.NewGetAPITopicUsecase(nil, nil)),
		apiwikilinkpreview.NewHandler("https://example.com", nil),
	)
	return newTestAPIHandlerWith(t, server, draftPageTestAuthenticator{principal: principal}, topicTestRateLimiter{})
}

// requestDraftPageは下書きのAPIに、tokenのトークンで要求を送る。ifMatchが空なら `If-Match` を付けない
func requestDraftPage(t *testing.T, r http.Handler, method, target, token, ifMatch, contentType, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if ifMatch != "" {
		req.Header.Set("If-Match", ifMatch)
	}
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

// currentDraftPageETagは下書きを取得し、応答のETagを返す
func currentDraftPageETag(t *testing.T, r http.Handler, target string) string {
	t.Helper()

	rr := requestDraftPage(t, r, http.MethodGet, target, "wkp_draft_page_read", "", "", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("取得のステータス = %d、期待値 = 200 (本文: %s)", rr.Code, rr.Body.String())
	}
	return rr.Header().Get("ETag")
}

func TestAPIRouter_UpdateDraftPage(t *testing.T) {
	t.Parallel()

	r := newDraftPageWriteTestRouter(t, "api-http-draft-update")
	target := "/api/v1/spaces/api-http-draft-update/pages/1/draft_page"
	const mergePatch = "application/merge-patch+json"
	update := func(token, ifMatch, contentType, body string) *httptest.ResponseRecorder {
		return requestDraftPage(t, r, http.MethodPatch, target, token, ifMatch, contentType, body)
	}

	// サブテストは同じ下書きを更新するため、並列にしない。
	t.Run("下書きを更新し、新しいETagを付けて200を返す", func(t *testing.T) {
		before := currentDraftPageETag(t, r, target)
		rr := update("wkp_draft_page_write", before, mergePatch, `{"title": "新しいタイトル", "body": "新しい本文"}`)
		if rr.Code != http.StatusOK {
			t.Fatalf("ステータス = %d、期待値 = 200 (本文: %s)", rr.Code, rr.Body.String())
		}
		var body apigen.DraftPage
		if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
			t.Fatalf("本文を読めない: %v", err)
		}
		if body.PageNumber != 1 || body.Title == nil || *body.Title != "新しいタイトル" || body.Body != "新しい本文" {
			t.Errorf("下書き = %+v、期待値 = 更新したページ1の下書き", body)
		}
		if want := "https://example.com/s/api-http-draft-update/pages/1/edit"; body.Url != want {
			t.Errorf("url = %q、期待値 = 編集画面の %q", body.Url, want)
		}
		etag := rr.Header().Get("ETag")
		if etag == before || etag != currentDraftPageETag(t, r, target) {
			t.Errorf("ETag = %q、期待値 = 更新前 (%q) と異なり、取得と同じ値", etag, before)
		}

		// 更新前のETagを前提にした更新は、先の更新と競合するため412になる
		assertProblem(t, update("wkp_draft_page_write", before, mergePatch, `{"body": "古い版を元にした本文"}`), http.StatusPreconditionFailed)
	})

	t.Run("タイトルをnullにするとタイトルを外す", func(t *testing.T) {
		rr := update("wkp_draft_page_write", currentDraftPageETag(t, r, target), mergePatch, `{"title": null}`)
		if rr.Code != http.StatusOK {
			t.Fatalf("ステータス = %d、期待値 = 200 (本文: %s)", rr.Code, rr.Body.String())
		}
		var raw map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &raw); err != nil {
			t.Fatalf("本文を読めない: %v", err)
		}
		if title, ok := raw["title"]; !ok || title != nil || raw["body"] != "新しい本文" {
			t.Errorf("下書き = %v、期待値 = タイトルがnullで本文は変えない", raw)
		}
	})

	t.Run("If-Matchが無ければ428", func(t *testing.T) {
		assertProblem(t, update("wkp_draft_page_write", "", mergePatch, `{"body": "本文"}`), http.StatusPreconditionRequired)
	})

	t.Run("draft_page:writeの無いトークンは403", func(t *testing.T) {
		rr := update("wkp_draft_page_read", currentDraftPageETag(t, r, target), mergePatch, `{"body": "スコープ不足"}`)
		assertProblem(t, rr, http.StatusForbidden)
		if got, want := rr.Header().Get("WWW-Authenticate"), `Bearer error="insufficient_scope", scope="draft_page:write", resource_metadata="https://example.com/.well-known/oauth-protected-resource/api/v1/spaces/api-http-draft-update"`; got != want {
			t.Errorf("WWW-Authenticate = %q、期待値 = %q", got, want)
		}
	})

	t.Run("JSON Merge Patch以外の本文は415", func(t *testing.T) {
		rr := update("wkp_draft_page_write", currentDraftPageETag(t, r, target), "application/json", `{"body": "本文"}`)
		assertProblem(t, rr, http.StatusUnsupportedMediaType)
		if got := rr.Header().Get("Accept-Patch"); got != mergePatch {
			t.Errorf("Accept-Patch = %q、期待値 = %s", got, mergePatch)
		}
	})

	t.Run("空のタイトルは422", func(t *testing.T) {
		assertProblem(t, update("wkp_draft_page_write", currentDraftPageETag(t, r, target), mergePatch, `{"title": ""}`), http.StatusUnprocessableEntity)
	})

	t.Run("定義していない項目を含む更新は422", func(t *testing.T) {
		assertProblem(t, update("wkp_draft_page_write", currentDraftPageETag(t, r, target), mergePatch, `{"topic_number": 2}`), http.StatusUnprocessableEntity)
	})

	t.Run("下書きの無いページは404", func(t *testing.T) {
		rr := requestDraftPage(t, r, http.MethodPatch, "/api/v1/spaces/api-http-draft-update/pages/2/draft_page", "wkp_draft_page_write", "*", mergePatch, `{"body": "本文"}`)
		assertProblem(t, rr, http.StatusNotFound)
	})
}

func TestAPIRouter_DeleteDraftPage(t *testing.T) {
	t.Parallel()

	r := newDraftPageWriteTestRouter(t, "api-http-draft-delete")
	target := "/api/v1/spaces/api-http-draft-delete/pages/1/draft_page"
	remove := func(token, ifMatch string) *httptest.ResponseRecorder {
		return requestDraftPage(t, r, http.MethodDelete, target, token, ifMatch, "", "")
	}

	// サブテストは同じ下書きを削除するため、並列にしない。
	t.Run("If-Matchが無ければ428", func(t *testing.T) {
		assertProblem(t, remove("wkp_draft_page_delete", ""), http.StatusPreconditionRequired)
	})

	t.Run("今の版と異なるETagは412", func(t *testing.T) {
		assertProblem(t, remove("wkp_draft_page_delete", `"古い版"`), http.StatusPreconditionFailed)
	})

	t.Run("draft_page:deleteの無いトークンは403", func(t *testing.T) {
		rr := remove("wkp_draft_page_write", currentDraftPageETag(t, r, target))
		assertProblem(t, rr, http.StatusForbidden)
		if got, want := rr.Header().Get("WWW-Authenticate"), `Bearer error="insufficient_scope", scope="draft_page:delete", resource_metadata="https://example.com/.well-known/oauth-protected-resource/api/v1/spaces/api-http-draft-delete"`; got != want {
			t.Errorf("WWW-Authenticate = %q、期待値 = %q", got, want)
		}
	})

	t.Run("下書きを削除して本文の無い204を返し、続く取得と削除は404", func(t *testing.T) {
		rr := remove("wkp_draft_page_delete", currentDraftPageETag(t, r, target))
		if rr.Code != http.StatusNoContent {
			t.Fatalf("ステータス = %d、期待値 = 204 (本文: %s)", rr.Code, rr.Body.String())
		}
		if rr.Body.Len() != 0 {
			t.Errorf("本文 = %q、期待値 = 空", rr.Body.String())
		}

		assertProblem(t, requestDraftPage(t, r, http.MethodGet, target, "wkp_draft_page_read", "", "", ""), http.StatusNotFound)
		assertProblem(t, remove("wkp_draft_page_delete", "*"), http.StatusNotFound)
	})
}

func TestAPIRouter_CreateDraftPagePublication(t *testing.T) {
	t.Parallel()

	r := newDraftPageWriteTestRouter(t, "api-http-draft-publish")
	draftTarget := "/api/v1/spaces/api-http-draft-publish/pages/1/draft_page"
	target := draftTarget + "/publication"
	const mergePatch = "application/merge-patch+json"
	publish := func(token, ifMatch string) *httptest.ResponseRecorder {
		return requestDraftPage(t, r, http.MethodPost, target, token, ifMatch, "", "")
	}
	updateTitle := func(title string) {
		t.Helper()

		rr := requestDraftPage(t, r, http.MethodPatch, draftTarget, "wkp_draft_page_write", currentDraftPageETag(t, r, draftTarget), mergePatch, `{"title": `+title+`}`)
		if rr.Code != http.StatusOK {
			t.Fatalf("下書きの更新のステータス = %d、期待値 = 200 (本文: %s)", rr.Code, rr.Body.String())
		}
	}

	// サブテストは同じ下書きを公開するため、並列にしない。
	t.Run("If-Matchが無ければ428", func(t *testing.T) {
		assertProblem(t, publish("wkp_draft_page_and_page_write", ""), http.StatusPreconditionRequired)
	})

	t.Run("今の版と異なるETagは412", func(t *testing.T) {
		assertProblem(t, publish("wkp_draft_page_and_page_write", `"古い版"`), http.StatusPreconditionFailed)
	})

	t.Run("draft_page:writeだけのトークンは403", func(t *testing.T) {
		rr := publish("wkp_draft_page_write", currentDraftPageETag(t, r, draftTarget))
		assertProblem(t, rr, http.StatusForbidden)
		if got, want := rr.Header().Get("WWW-Authenticate"), `Bearer error="insufficient_scope", scope="draft_page:write page:write", resource_metadata="https://example.com/.well-known/oauth-protected-resource/api/v1/spaces/api-http-draft-publish"`; got != want {
			t.Errorf("WWW-Authenticate = %q、期待値 = %q", got, want)
		}
	})

	t.Run("タイトルの無い下書きは、titleの問題として422", func(t *testing.T) {
		updateTitle("null")
		rr := publish("wkp_draft_page_and_page_write", currentDraftPageETag(t, r, draftTarget))
		assertProblem(t, rr, http.StatusUnprocessableEntity)
		var problem struct {
			Errors []struct {
				Pointer *string `json:"pointer"`
			} `json:"errors"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &problem); err != nil {
			t.Fatalf("本文を読めない: %v", err)
		}
		if len(problem.Errors) != 1 || problem.Errors[0].Pointer == nil || *problem.Errors[0].Pointer != "#/title" {
			t.Errorf("errors = %s、期待値 = pointerが #/title の問題1件", rr.Body.String())
		}
	})

	t.Run("下書きの内容でページを公開し、Content-LocationとETagを付けて200を返す", func(t *testing.T) {
		updateTitle(`"公開するタイトル"`)
		rr := publish("wkp_draft_page_and_page_write", currentDraftPageETag(t, r, draftTarget))
		if rr.Code != http.StatusOK {
			t.Fatalf("ステータス = %d、期待値 = 200 (本文: %s)", rr.Code, rr.Body.String())
		}
		var body apigen.Page
		if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
			t.Fatalf("本文を読めない: %v", err)
		}
		if body.Number != 1 || body.Title == nil || *body.Title != "公開するタイトル" || body.Body != "下書きの本文" || body.Status != apigen.PageStatusPublished {
			t.Errorf("ページ = %+v、期待値 = 下書きの内容で公開したページ1", body)
		}
		// 公開の応答はページのため、編集画面ではなくページのURLを返す
		if want := "https://example.com/s/api-http-draft-publish/pages/1"; body.Url != want {
			t.Errorf("url = %q、期待値 = %q", body.Url, want)
		}
		if got, want := rr.Header().Get("Content-Location"), "/api/v1/spaces/api-http-draft-publish/pages/1"; got != want {
			t.Errorf("Content-Location = %q、期待値 = %q", got, want)
		}
		if got := rr.Header().Get("ETag"); !strings.HasPrefix(got, `"`) || !strings.HasSuffix(got, `"`) || len(got) <= 2 {
			t.Errorf("ETag = %q、期待値 = 引用符で囲んだ強いETag", got)
		}

		// 公開すると下書きが消えるため、続く取得と公開は404になる
		assertProblem(t, requestDraftPage(t, r, http.MethodGet, draftTarget, "wkp_draft_page_read", "", "", ""), http.StatusNotFound)
		assertProblem(t, publish("wkp_draft_page_and_page_write", "*"), http.StatusNotFound)
	})
}
