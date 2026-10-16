package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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

func TestAPIRouter_CreatePageBodyEdit(t *testing.T) {
	t.Parallel()

	// 本文の検索置換はUseCaseが自前でトランザクションを張るため、フィクスチャをテストDBへコミットする
	db := testutil.GetTestDB()
	userID := testutil.NewUserBuilderDB(t, db).
		WithAtname("api_http_page_body_edit").WithEmail("api_http_page_body_edit@example.com").Build()
	spaceID := testutil.NewSpaceBuilderDB(t, db).WithIdentifier("api-http-page-body-edit").Build()
	spaceMemberID := testutil.NewSpaceMemberBuilderDB(t, db).
		WithSpaceID(spaceID).WithUserID(userID).Build()
	topicID := testutil.NewTopicBuilderDB(t, db).WithSpaceID(spaceID).WithNumber(1).WithName("トピック").Build()
	testutil.NewPageBuilderDB(t, db).WithSpaceID(spaceID).WithTopicID(topicID).WithNumber(1).WithTitle("タイトル").WithBody("一つ目 二つ目 二つ目").Build()

	principal := &model.APIPrincipal{
		User:        &model.User{ID: userID},
		Space:       &model.Space{ID: spaceID, Identifier: "api-http-page-body-edit"},
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
			nil,
		),
		apipagebacklink.NewHandler("https://example.com", nil),
		apipagebodyedit.NewHandler("https://example.com", usecase.NewCreateAPIPageBodyEditUsecase(
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
		)),
		apipagelink.NewHandler("https://example.com", nil),
		apispace.NewHandler(usecase.NewGetAPISpaceUsecase()),
		apitopic.NewHandler(usecase.NewListAPITopicsUsecase(nil, nil), usecase.NewGetAPITopicUsecase(nil, nil)),
		apiwikilinkpreview.NewHandler("https://example.com", nil),
	)
	r := newTestAPIHandlerWith(t, server, pageTestAuthenticator{principal: principal}, topicTestRateLimiter{})

	const pagePath = "/api/v1/spaces/api-http-page-body-edit/pages/1"
	target := pagePath + "/body_edits"
	// currentPageはページを取得し、応答の本文とETagを返す
	currentPage := func(t *testing.T) (apigen.Page, string) {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, pagePath, nil)
		req.Header.Set("Authorization", "Bearer wkp_page_read")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("取得のステータス = %d、期待値 = 200 (本文: %s)", rr.Code, rr.Body.String())
		}
		var page apigen.Page
		if err := json.Unmarshal(rr.Body.Bytes(), &page); err != nil {
			t.Fatalf("本文を読めない: %v", err)
		}
		return page, rr.Header().Get("ETag")
	}
	request := func(token, ifMatch, body string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		if ifMatch != "" {
			req.Header.Set("If-Match", ifMatch)
		}
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		return rr
	}
	// assertFieldProblemは422の問題が、pointerの指すリクエスト本文の項目についてのものであることを確かめ、detailを返す
	assertFieldProblem := func(t *testing.T, rr *httptest.ResponseRecorder, pointer string) string {
		t.Helper()
		assertProblem(t, rr, http.StatusUnprocessableEntity)
		var problem struct {
			Errors []struct {
				Detail  string  `json:"detail"`
				Pointer *string `json:"pointer"`
			} `json:"errors"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &problem); err != nil {
			t.Fatalf("本文を読めない: %v", err)
		}
		if len(problem.Errors) == 0 || problem.Errors[0].Pointer == nil || *problem.Errors[0].Pointer != pointer {
			t.Errorf("errors = %s、期待値 = pointerが %s の問題", rr.Body.String(), pointer)
			return ""
		}
		return problem.Errors[0].Detail
	}

	// サブテストは同じページを更新するため、並列にしない。
	t.Run("If-Match無しで本文を置換し、ページのURLとETagを付けて200を返す", func(t *testing.T) {
		rr := request("wkp_page_write", "", `{"edits": [{"old_str": "一つ目", "new_str": "1つ目"}]}`)
		if rr.Code != http.StatusOK {
			t.Fatalf("ステータス = %d、期待値 = 200 (本文: %s)", rr.Code, rr.Body.String())
		}
		if got := rr.Header().Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q、期待値 = application/json", got)
		}
		if got := rr.Header().Get("Content-Location"); got != pagePath {
			t.Errorf("Content-Location = %q、期待値 = %q", got, pagePath)
		}
		var body apigen.Page
		if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
			t.Fatalf("本文を読めない: %v", err)
		}
		if body.Number != 1 || body.Title == nil || *body.Title != "タイトル" || body.Body != "1つ目 二つ目 二つ目" {
			t.Errorf("ページ = %+v、期待値 = 本文を置換したページ1", body)
		}
		if want := "https://example.com/s/api-http-page-body-edit/pages/1"; body.Url != want {
			t.Errorf("url = %q、期待値 = %q", body.Url, want)
		}
		if _, etag := currentPage(t); rr.Header().Get("ETag") != etag {
			t.Errorf("ETag = %q、期待値 = 取得と同じ %q", rr.Header().Get("ETag"), etag)
		}
	})

	t.Run("今のETagをIf-Matchで送れば置換する", func(t *testing.T) {
		_, etag := currentPage(t)
		rr := request("wkp_page_write", etag, `{"edits": [{"old_str": "二つ目", "new_str": "2つ目", "replace_all": true}]}`)
		if rr.Code != http.StatusOK {
			t.Fatalf("ステータス = %d、期待値 = 200 (本文: %s)", rr.Code, rr.Body.String())
		}

		// 置換前のETagを前提にした置換は、先の置換と競合するため412になる
		assertProblem(t, request("wkp_page_write", etag, `{"edits": [{"old_str": "1つ目", "new_str": "x"}]}`), http.StatusPreconditionFailed)
	})

	t.Run("どのETagとも一致しないIf-Match (W/付き) は、省略と区別して412", func(t *testing.T) {
		assertProblem(t, request("wkp_page_write", `W/"古い版"`, `{"edits": [{"old_str": "1つ目", "new_str": "x"}]}`), http.StatusPreconditionFailed)
	})

	t.Run("今のETagにW/が付いていても置換する", func(t *testing.T) {
		_, etag := currentPage(t)
		rr := request("wkp_page_write", "W/"+etag, `{"edits": [{"old_str": "1つ目", "new_str": "一つ目"}]}`)
		if rr.Code != http.StatusOK {
			t.Fatalf("ステータス = %d、期待値 = 200 (本文: %s)", rr.Code, rr.Body.String())
		}
		rr = request("wkp_page_write", "", `{"edits": [{"old_str": "一つ目", "new_str": "1つ目"}]}`)
		if rr.Code != http.StatusOK {
			t.Fatalf("元に戻す置換のステータス = %d、期待値 = 200 (本文: %s)", rr.Code, rr.Body.String())
		}
	})

	t.Run("一致しない編集は、何番目かを示して422にし、前の編集も適用しない", func(t *testing.T) {
		before, _ := currentPage(t)
		rr := request("wkp_page_write", "", `{"edits": [{"old_str": "1つ目", "new_str": "x"}, {"old_str": "無い", "new_str": "y"}]}`)
		assertFieldProblem(t, rr, "#/edits/1/old_str")
		if after, _ := currentPage(t); after.Body != before.Body {
			t.Errorf("本文 = %q、期待値 = %q のまま", after.Body, before.Body)
		}
	})

	t.Run("replace_allでなく複数件に一致する編集は、一致の件数を示して422", func(t *testing.T) {
		detail := assertFieldProblem(t, request("wkp_page_write", "", `{"edits": [{"old_str": "2つ目", "new_str": "x"}]}`), "#/edits/0/old_str")
		if !strings.Contains(detail, "2件") {
			t.Errorf("detail = %q、期待値 = 一致の件数 (2件) を含む", detail)
		}
	})

	t.Run("小さなリクエストでも本文が上限を超える連続置換は422", func(t *testing.T) {
		before, etag := currentPage(t)
		edits := make([]apigen.BodyEdit, 100)
		replaceAll := true
		for i := range edits {
			edits[i] = apigen.BodyEdit{OldStr: "1", NewStr: "11", ReplaceAll: &replaceAll}
		}
		body, err := json.Marshal(apigen.NewBodyEdit{Edits: edits})
		if err != nil {
			t.Fatalf("リクエストのエンコードに失敗: %v", err)
		}
		if len(body) >= 1<<20 {
			t.Fatalf("リクエストのバイト数 = %d、期待値 = 1MiB未満", len(body))
		}
		rr := request("wkp_page_write", "", string(body))
		detail := assertFieldProblem(t, rr, "#/edits/19/new_str")
		if !strings.Contains(detail, "1048576") {
			t.Errorf("detail = %q、期待値 = 上限のバイト数を含む", detail)
		}
		after, afterETag := currentPage(t)
		if after.Body != before.Body || afterETag != etag {
			t.Error("サイズ超過で拒否した編集が保存された")
		}
	})

	t.Run("編集が空なら422", func(t *testing.T) {
		assertProblem(t, request("wkp_page_write", "", `{"edits": []}`), http.StatusUnprocessableEntity)
	})

	t.Run("old_strが空なら422", func(t *testing.T) {
		assertProblem(t, request("wkp_page_write", "", `{"edits": [{"old_str": "", "new_str": "x"}]}`), http.StatusUnprocessableEntity)
	})

	t.Run("定義していない項目を含む編集は422", func(t *testing.T) {
		assertProblem(t, request("wkp_page_write", "", `{"edits": [{"old_str": "1つ目", "new_str": "x", "title": "t"}]}`), http.StatusUnprocessableEntity)
	})

	t.Run("page:writeの無いトークンは403", func(t *testing.T) {
		rr := request("wkp_page_read", "", `{"edits": [{"old_str": "1つ目", "new_str": "x"}]}`)
		assertProblem(t, rr, http.StatusForbidden)
		if got, want := rr.Header().Get("WWW-Authenticate"), `Bearer error="insufficient_scope", scope="page:write", resource_metadata="https://example.com/.well-known/oauth-protected-resource/api/v1/spaces/api-http-page-body-edit"`; got != want {
			t.Errorf("WWW-Authenticate = %q、期待値 = %q", got, want)
		}
	})

	t.Run("存在しないページは404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/spaces/api-http-page-body-edit/pages/99/body_edits", strings.NewReader(`{"edits": [{"old_str": "a", "new_str": "b"}]}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer wkp_page_write")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		assertProblem(t, rr, http.StatusNotFound)
	})
}
