package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/api"
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
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

func TestAPIRouter_DraftPageRevisions(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	userID := testutil.NewUserBuilder(t, tx).
		WithAtname("api_http_draft_revisions").WithEmail("api_http_draft_revisions@example.com").Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("api-http-draft-revisions").Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).WithUserID(userID).Build()
	topicID := testutil.NewTopicBuilder(t, tx).WithSpaceID(spaceID).WithNumber(1).WithName("トピック").Build()
	// ページ1と2の両方にトークンの持ち主の下書きとリビジョンを1件ずつ作る
	revisionIDs := map[model.PageNumber]model.DraftPageRevisionID{}
	for _, number := range []model.PageNumber{1, 2} {
		pageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).WithTopicID(topicID).WithNumber(number).WithTitle(fmt.Sprintf("ページ%d", number)).Build()
		draftPageID := testutil.NewDraftPageBuilder(t, tx).
			WithSpaceID(spaceID).WithPageID(pageID).WithSpaceMemberID(spaceMemberID).WithTopicID(topicID).Build()
		revisionIDs[number] = testutil.NewDraftPageRevisionBuilder(t, tx).
			WithDraftPageID(draftPageID).WithSpaceID(spaceID).WithSpaceMemberID(spaceMemberID).
			WithTitle("リビジョン").WithBody("リビジョンの本文").
			WithCreatedAt(time.Date(2026, 9, 1, 12, 0, 0, 123456000, time.UTC)).Build()
	}

	principal := &model.APIPrincipal{
		User:        &model.User{ID: userID},
		Space:       &model.Space{ID: spaceID, Identifier: "api-http-draft-revisions"},
		SpaceMember: &model.SpaceMember{ID: spaceMemberID, SpaceID: spaceID, UserID: userID, Role: model.SpaceRoleAdmin, Active: true},
		TokenKind:   model.APITokenKindPersonalAccessToken,
	}
	q := testutil.QueriesWithTx(tx)
	pageRepo := repository.NewPageRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	draftPageRevisionRepo := repository.NewDraftPageRevisionRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	server := apihandler.NewServer(
		apiattachment.NewHandler(nil),
		apicurrentspacemember.NewHandler(usecase.NewGetAPICurrentSpaceMemberUsecase()),
		apicurrenttoken.NewHandler(usecase.NewGetAPICurrentTokenUsecase()),
		apidraftpage.NewHandler("https://example.com", usecase.NewListAPIDraftPagesUsecase(nil, nil, nil), usecase.NewGetAPIDraftPageUsecase(nil, nil, nil, nil), nil, nil, nil),
		apidraftpagebodyedit.NewHandler("https://example.com", nil),
		apidraftpagepublication.NewHandler("https://example.com", nil),
		apidraftpagerevision.NewHandler(
			usecase.NewListAPIDraftPageRevisionsUsecase(pageRepo, draftPageRepo, draftPageRevisionRepo, topicRepo, topicMemberRepo),
			usecase.NewGetAPIDraftPageRevisionUsecase(pageRepo, draftPageRepo, draftPageRevisionRepo, topicRepo, topicMemberRepo),
		),
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

	request := func(target, token string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, target, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		return rr
	}
	listPath := "/api/v1/spaces/api-http-draft-revisions/pages/1/draft_page_revisions"
	showPath := listPath + "/" + string(revisionIDs[1])

	// サブテストは親のトランザクションを共有するため、並列にしない。
	t.Run("下書きの編集履歴を返す", func(t *testing.T) {
		rr := request(listPath, "wkp_draft_page_read")
		if rr.Code != http.StatusOK {
			t.Fatalf("ステータス = %d、期待値 = 200 (本文: %s)", rr.Code, rr.Body.String())
		}
		var body struct {
			Items      []map[string]any `json:"items"`
			NextCursor *string          `json:"next_cursor"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
			t.Fatalf("本文を読めない: %v", err)
		}
		if len(body.Items) != 1 || body.Items[0]["id"] != string(revisionIDs[1]) || body.NextCursor != nil {
			t.Errorf("一覧 = %+v、期待値 = ページ1の下書きのリビジョンのみ", body)
		}
	})

	t.Run("下書きのリビジョンを返す", func(t *testing.T) {
		rr := request(showPath, "wkp_draft_page_write")
		if rr.Code != http.StatusOK {
			t.Fatalf("ステータス = %d、期待値 = 200 (本文: %s)", rr.Code, rr.Body.String())
		}
		var raw map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &raw); err != nil {
			t.Fatalf("本文を読めない: %v", err)
		}
		if raw["id"] != string(revisionIDs[1]) || raw["title"] != "リビジョン" || raw["body"] != "リビジョンの本文" || raw["created_at"] != "2026-09-01T12:00:00.123456Z" {
			t.Errorf("リビジョン = %v", raw)
		}
	})

	for _, target := range []string{listPath, showPath} {
		t.Run("draft_page:readの無いトークンは403: "+target, func(t *testing.T) {
			rr := request(target, "wkp_page_write")
			assertProblem(t, rr, http.StatusForbidden)
			if got, want := rr.Header().Get("WWW-Authenticate"), `Bearer error="insufficient_scope", scope="draft_page:read", resource_metadata="https://example.com/.well-known/oauth-protected-resource/api/v1/spaces/api-http-draft-revisions"`; got != want {
				t.Errorf("WWW-Authenticate = %q、期待値 = %q", got, want)
			}
		})
	}

	t.Run("別のページの下書きのリビジョンのIDを指定すると404", func(t *testing.T) {
		assertProblem(t, request(listPath+"/"+string(revisionIDs[2]), "wkp_draft_page_read"), http.StatusNotFound)
	})

	t.Run("UUIDでないリビジョンのIDは422", func(t *testing.T) {
		rr := request(listPath+"/not-a-uuid", "wkp_draft_page_read")
		assertProblem(t, rr, http.StatusUnprocessableEntity)
		var problem struct {
			Errors []struct {
				Parameter *string `json:"parameter"`
			} `json:"errors"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &problem); err != nil {
			t.Fatalf("本文を読めない: %v", err)
		}
		if len(problem.Errors) != 1 || problem.Errors[0].Parameter == nil || *problem.Errors[0].Parameter != "draft_page_revision_id" {
			t.Errorf("errors = %s、期待値 = draft_page_revision_idの問題", rr.Body.String())
		}
	})

	t.Run("不正なカーソルは422", func(t *testing.T) {
		assertProblem(t, request(listPath+"?cursor=invalid!", "wkp_draft_page_read"), http.StatusUnprocessableEntity)
	})

	t.Run("下書きの無いページは404", func(t *testing.T) {
		assertProblem(t, request("/api/v1/spaces/api-http-draft-revisions/pages/99/draft_page_revisions", "wkp_draft_page_read"), http.StatusNotFound)
	})
}
