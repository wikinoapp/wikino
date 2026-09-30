package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
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
	"github.com/wikinoapp/wikino/go/internal/ratelimit"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// topicTestAuthenticatorはスコープの異なる2つのテスト用トークンを受け付ける。
type topicTestAuthenticator struct {
	principal *model.APIPrincipal
}

func (a topicTestAuthenticator) Execute(_ context.Context, token string) (*model.APIPrincipal, error) {
	p := *a.principal
	switch token {
	case "wkp_topic_read":
		p.Scopes = []model.Scope{model.ScopeTopicRead}
	case "wkp_topic_no_scope":
		p.Scopes = nil
	default:
		return nil, nil
	}
	return &p, nil
}

type topicTestRateLimiter struct{}

func (topicTestRateLimiter) Check(context.Context, ratelimit.CheckInput) (*ratelimit.CheckResult, error) {
	return &ratelimit.CheckResult{Allowed: true, Remaining: 4999, ResetAt: time.Now().Add(time.Hour)}, nil
}

func TestAPIRouter_Topics(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	userID := testutil.NewUserBuilder(t, tx).
		WithAtname("api_http_topics").WithEmail("api_http_topics@example.com").Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("api-http-topics").Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).
		WithSpaceID(spaceID).WithUserID(userID).Build()
	for number := int32(1); number <= 3; number++ {
		testutil.NewTopicBuilder(t, tx).
			WithSpaceID(spaceID).WithNumber(number).WithName(fmt.Sprintf("トピック%d", number)).Build()
	}

	principal := &model.APIPrincipal{
		User:        &model.User{ID: userID},
		Space:       &model.Space{ID: spaceID, Identifier: "api-http-topics"},
		SpaceMember: &model.SpaceMember{ID: spaceMemberID, SpaceID: spaceID, UserID: userID, Scopes: []model.Scope{model.ScopeSpaceAdmin}, Active: true},
		TokenKind:   model.APITokenKindPersonalAccessToken,
	}
	q := testutil.QueriesWithTx(tx)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	server := apihandler.NewServer(
		apicurrentspacemember.NewHandler(usecase.NewGetAPICurrentSpaceMemberUsecase()),
		openapi_description.NewHandler(api.OpenAPIDescription),
		apipage.NewHandler(usecase.NewListAPIPagesUsecase(nil, nil, nil), usecase.NewGetAPIPageUsecase(nil, nil, nil), nil, nil),
		apispace.NewHandler(usecase.NewGetAPISpaceUsecase()),
		apitopic.NewHandler(
			usecase.NewListAPITopicsUsecase(topicRepo, topicMemberRepo),
			usecase.NewGetAPITopicUsecase(topicRepo, topicMemberRepo),
		),
	)
	r := newTestAPIHandlerWith(t, server, topicTestAuthenticator{principal: principal}, topicTestRateLimiter{})

	request := func(target, token string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, target, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		return rr
	}
	base := "/api/v1/spaces/api-http-topics/topics"

	// サブテストは親のトランザクションを共有するため、並列にしない。
	t.Run("一覧の本文とLinkで次のページを辿れる", func(t *testing.T) {
		first := request(base+"?limit=2", "wkp_topic_read")
		if first.Code != http.StatusOK {
			t.Fatalf("ステータス = %d、期待値 = 200 (本文: %s)", first.Code, first.Body.String())
		}
		if got := first.Header().Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q、期待値 = application/json", got)
		}
		var body apigen.TopicList
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
		second := request(base+"?limit=2&cursor="+url.QueryEscape(*body.NextCursor), "wkp_topic_read")
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

	t.Run("個別のトピックを返す", func(t *testing.T) {
		rr := request(base+"/2", "wkp_topic_read")
		if rr.Code != http.StatusOK {
			t.Fatalf("ステータス = %d、期待値 = 200 (本文: %s)", rr.Code, rr.Body.String())
		}
		var body apigen.Topic
		if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
			t.Fatalf("本文を読めない: %v", err)
		}
		if body.Number != 2 || body.Visibility != apigen.TopicVisibility("public") {
			t.Errorf("トピック = %+v、期待値 = 番号2の公開トピック", body)
		}
	})

	for _, target := range []string{base, base + "/2"} {
		t.Run("スコープの無いトークンは403: "+target, func(t *testing.T) {
			rr := request(target, "wkp_topic_no_scope")
			assertProblem(t, rr, http.StatusForbidden)
			if got, want := rr.Header().Get("WWW-Authenticate"), `Bearer error="insufficient_scope", scope="topic:read", resource_metadata="https://example.com/.well-known/oauth-protected-resource/api/v1/spaces/api-http-topics"`; got != want {
				t.Errorf("WWW-Authenticate = %q、期待値 = %q", got, want)
			}
		})
	}

	t.Run("存在しないトピックは404", func(t *testing.T) {
		assertProblem(t, request(base+"/99", "wkp_topic_read"), http.StatusNotFound)
	})

	t.Run("壊れたカーソルは422", func(t *testing.T) {
		assertProblem(t, request(base+"?cursor=invalid!", "wkp_topic_read"), http.StatusUnprocessableEntity)
	})

	t.Run("束縛先が違えば壊れたカーソルより先に404", func(t *testing.T) {
		assertProblem(t, request("/api/v1/spaces/other-space/topics?cursor=invalid!", "wkp_topic_read"), http.StatusNotFound)
	})
}
