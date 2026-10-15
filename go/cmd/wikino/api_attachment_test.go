package main

import (
	"context"
	"encoding/json"
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
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/storage"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// attachmentTestAuthenticatorはスコープの異なるテスト用トークンを受け付ける。
type attachmentTestAuthenticator struct {
	principal *model.APIPrincipal
}

func (a attachmentTestAuthenticator) Execute(_ context.Context, token string) (*model.APIPrincipal, error) {
	p := *a.principal
	switch token {
	case "wkp_attachment_read":
		p.Scopes = []model.Scope{model.ScopeAttachmentRead}
	case "wkp_page_read":
		p.Scopes = []model.Scope{model.ScopePageRead, model.ScopeTopicRead}
	default:
		return nil, nil
	}
	return &p, nil
}

func TestAPIRouter_GetAttachment(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	userID := testutil.NewUserBuilder(t, tx).
		WithAtname("api_http_attachment").WithEmail("api_http_attachment@example.com").Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("api-http-attachment").Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).WithUserID(userID).Build()
	attachmentID := testutil.NewAttachmentBuilder(t, tx).
		WithSpaceID(spaceID).WithSpaceMemberID(spaceMemberID).
		WithFilename("図.png").WithContentType("image/png").WithByteSize(1234).WithBlobKey("api-http-attachment-key").
		Build()
	otherSpaceID, otherSpaceMemberID := testutil.SetupSpaceWithMember(t, tx, "api-http-attachment-other")
	otherAttachmentID := testutil.NewAttachmentBuilder(t, tx).
		WithSpaceID(otherSpaceID).WithSpaceMemberID(otherSpaceMemberID).WithBlobKey("api-http-attachment-other-key").
		Build()

	principal := &model.APIPrincipal{
		User:        &model.User{ID: userID},
		Space:       &model.Space{ID: spaceID, Identifier: "api-http-attachment"},
		SpaceMember: &model.SpaceMember{ID: spaceMemberID, SpaceID: spaceID, UserID: userID, Role: model.SpaceRoleEditor, Active: true},
		TokenKind:   model.APITokenKindPersonalAccessToken,
	}
	q := testutil.QueriesWithTx(tx)
	server := apihandler.NewServer(
		apiattachment.NewHandler(usecase.NewGetAPIAttachmentUsecase(repository.NewAttachmentRepository(q), storage.NewFakeObjectStorage())),
		apicurrentspacemember.NewHandler(usecase.NewGetAPICurrentSpaceMemberUsecase()),
		apicurrenttoken.NewHandler(usecase.NewGetAPICurrentTokenUsecase()),
		apidraftpage.NewHandler("https://example.com", usecase.NewListAPIDraftPagesUsecase(nil, nil, nil), usecase.NewGetAPIDraftPageUsecase(nil, nil, nil, nil), nil, nil, nil),
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
	// 応答の `Cache-Control` を確かめるため、本番と同じく共通のヘッダーを付けるミドルウェアを外側に置く
	r := middleware.APIResponseHeader(newTestAPIHandlerWith(t, server, attachmentTestAuthenticator{principal: principal}, topicTestRateLimiter{}))

	request := func(target, token string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, target, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		return rr
	}
	const basePath = "/api/v1/spaces/api-http-attachment/attachments/"

	// サブテストは親のトランザクションを共有するため、並列にしない。
	t.Run("添付ファイルの情報と署名付きURLを返す", func(t *testing.T) {
		before := time.Now().UTC().Truncate(time.Second)
		rr := request(basePath+string(attachmentID), "wkp_attachment_read")
		if rr.Code != http.StatusOK {
			t.Fatalf("ステータス = %d、期待値 = 200 (本文: %s)", rr.Code, rr.Body.String())
		}
		// 署名付きURLを含むため、応答をキャッシュに残させない
		if got := rr.Header().Get("Cache-Control"); got != "no-store" {
			t.Errorf("Cache-Control = %q、期待値 = no-store", got)
		}
		var raw map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &raw); err != nil {
			t.Fatalf("本文を読めない: %v", err)
		}
		if raw["id"] != string(attachmentID) || raw["filename"] != "図.png" || raw["content_type"] != "image/png" || raw["byte_size"] != float64(1234) {
			t.Errorf("添付ファイル = %v", raw)
		}
		if raw["download_url"] != "https://storage.example.com/api-http-attachment-key?expires_in=300" {
			t.Errorf("download_url = %v", raw["download_url"])
		}
		expiresAt, err := time.Parse(time.RFC3339, raw["download_url_expires_at"].(string))
		if err != nil {
			t.Fatalf("download_url_expires_atを読めない: %v", err)
		}
		if expiresAt.Before(before.Add(5*time.Minute)) || expiresAt.After(time.Now().Add(5*time.Minute)) {
			t.Errorf("download_url_expires_at = %v、期待値 = 取得から5分後", expiresAt)
		}
	})

	t.Run("attachment:readの無いトークンは403", func(t *testing.T) {
		rr := request(basePath+string(attachmentID), "wkp_page_read")
		assertProblem(t, rr, http.StatusForbidden)
		if got, want := rr.Header().Get("WWW-Authenticate"), `Bearer error="insufficient_scope", scope="attachment:read", resource_metadata="https://example.com/.well-known/oauth-protected-resource/api/v1/spaces/api-http-attachment"`; got != want {
			t.Errorf("WWW-Authenticate = %q、期待値 = %q", got, want)
		}
	})

	t.Run("別のスペースの添付ファイルは404", func(t *testing.T) {
		assertProblem(t, request(basePath+string(otherAttachmentID), "wkp_attachment_read"), http.StatusNotFound)
	})

	t.Run("存在しない添付ファイルは404", func(t *testing.T) {
		assertProblem(t, request(basePath+"00000000-0000-0000-0000-000000000000", "wkp_attachment_read"), http.StatusNotFound)
	})

	t.Run("トークンが属さないスペースのパスは404", func(t *testing.T) {
		assertProblem(t, request("/api/v1/spaces/api-http-attachment-other/attachments/"+string(otherAttachmentID), "wkp_attachment_read"), http.StatusNotFound)
	})

	t.Run("UUIDでない添付ファイルのIDは422", func(t *testing.T) {
		rr := request(basePath+"not-a-uuid", "wkp_attachment_read")
		assertProblem(t, rr, http.StatusUnprocessableEntity)
		var problem struct {
			Errors []struct {
				Parameter *string `json:"parameter"`
			} `json:"errors"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &problem); err != nil {
			t.Fatalf("本文を読めない: %v", err)
		}
		if len(problem.Errors) != 1 || problem.Errors[0].Parameter == nil || *problem.Errors[0].Parameter != "attachment_id" {
			t.Errorf("errors = %s、期待値 = attachment_idの問題", rr.Body.String())
		}
	})
}
