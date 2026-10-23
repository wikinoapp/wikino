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
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

func TestAPIRouter_CreateWikilinkPreview(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	userID := testutil.NewUserBuilder(t, tx).
		WithAtname("api_http_wikilink_preview").WithEmail("api_http_wikilink_preview@example.com").Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("api-http-wikilink-preview").Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).WithUserID(userID).Build()
	topicID := testutil.NewTopicBuilder(t, tx).WithSpaceID(spaceID).WithNumber(1).WithName("トピック").Build()
	pageID := testutil.NewPageBuilder(t, tx).WithSpaceID(spaceID).WithTopicID(topicID).WithNumber(1).WithTitle("既存").Build()

	principal := &model.APIPrincipal{
		User:        &model.User{ID: userID},
		Space:       &model.Space{ID: spaceID, Identifier: "api-http-wikilink-preview"},
		SpaceMember: &model.SpaceMember{ID: spaceMemberID, SpaceID: spaceID, UserID: userID, Role: model.SpaceRoleAdmin, Active: true},
		TokenKind:   model.APITokenKindPersonalAccessToken,
	}
	q := testutil.QueriesWithTx(tx)
	server := apihandler.NewServer(
		apiattachment.NewHandler(nil),
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
		apiwikilinkpreview.NewHandler("https://example.com", usecase.NewGetAPIWikilinkPreviewUsecase(
			repository.NewPageRepository(q),
			repository.NewTopicRepository(q),
			repository.NewTopicMemberRepository(q),
		)),
	)
	r := newTestAPIHandlerWith(t, server, pageTestAuthenticator{principal: principal}, topicTestRateLimiter{})

	request := func(target, token, body string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		return rr
	}
	target := "/api/v1/spaces/api-http-wikilink-preview/wikilink_previews"

	// サブテストは親のトランザクションを共有するため、並列にしない。
	t.Run("各Wikiリンクの解決の結果を200で返す", func(t *testing.T) {
		rr := request(target, "wkp_page_read", `{"topic_number": 1, "body": "[[既存]] [[新規]] [[無いトピック/ページ]]"}`)
		if rr.Code != http.StatusOK {
			t.Fatalf("ステータス = %d、期待値 = 200 (本文: %s)", rr.Code, rr.Body.String())
		}

		// 値の無い `page` もキーを省かずnullで返す
		var raw struct {
			Links []map[string]json.RawMessage `json:"links"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &raw); err != nil {
			t.Fatalf("本文を読めない: %v", err)
		}
		if len(raw.Links) != 3 || string(raw.Links[1]["page"]) != "null" {
			t.Fatalf("本文 = %s、期待値 = 3件のリンクと、2件目の `page` がnull", rr.Body.String())
		}

		var body apigen.WikilinkPreview
		if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
			t.Fatalf("本文を読めない: %v", err)
		}
		existing := body.Links[0]
		if existing.Text != "既存" || existing.TopicName != "トピック" || existing.PageTitle != "既存" ||
			existing.Resolution != apigen.WikilinkPreviewLinkResolutionExistingPage || existing.Page == nil ||
			existing.Page.Id.String() != string(pageID) || existing.Page.Number != 1 || existing.Page.TopicNumber != 1 ||
			existing.Page.Title == nil || *existing.Page.Title != "既存" || existing.Page.Status != apigen.WikilinkPreviewPageStatusPublished {
			t.Errorf("既存のページへのリンク = %+v、期待値 = ページ1を指す既存のページ", existing)
		}
		if want := "https://example.com/s/api-http-wikilink-preview/pages/1"; existing.Page != nil && existing.Page.Url != want {
			t.Errorf("url = %q、期待値 = %q", existing.Page.Url, want)
		}
		if got := body.Links[1]; got.Text != "新規" || got.Resolution != apigen.WikilinkPreviewLinkResolutionNewPage || got.Page != nil {
			t.Errorf("存在しないページへのリンク = %+v、期待値 = 作られるページ", got)
		}
		if got := body.Links[2]; got.TopicName != "無いトピック" || got.PageTitle != "ページ" || got.Resolution != apigen.WikilinkPreviewLinkResolutionTopicNotFound || got.Page != nil {
			t.Errorf("存在しないトピックへのリンク = %+v、期待値 = トピックが無い", got)
		}
	})

	t.Run("存在しないトピックの番号は422", func(t *testing.T) {
		assertProblem(t, request(target, "wkp_page_read", `{"topic_number": 99, "body": "[[既存]]"}`), http.StatusUnprocessableEntity)
	})

	t.Run("本文の無いリクエストは422", func(t *testing.T) {
		assertProblem(t, request(target, "wkp_page_read", `{"topic_number": 1}`), http.StatusUnprocessableEntity)
	})

	t.Run("page:readの無いトークンは403", func(t *testing.T) {
		assertProblem(t, request(target, "wkp_topic_read", `{"topic_number": 1, "body": ""}`), http.StatusForbidden)
	})

	t.Run("トークンが属さないスペースなら404", func(t *testing.T) {
		assertProblem(t, request("/api/v1/spaces/other-space/wikilink_previews", "wkp_page_read", `{"topic_number": 1, "body": ""}`), http.StatusNotFound)
	})
}
