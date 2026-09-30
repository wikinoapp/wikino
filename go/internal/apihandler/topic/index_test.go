package topic_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/apierror"
	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/apihandler/topic"
	"github.com/wikinoapp/wikino/go/internal/apipagination"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// setupTopicHandlerは、公開トピックをtopicCount件持つスペースと、そのスペースの管理者が
// `topic:read` を持つトークンで呼び出したcontextを用意する
func setupTopicHandler(t *testing.T, tx *sql.Tx, topicCount int32) (*topic.Handler, context.Context) {
	t.Helper()

	userID := testutil.NewUserBuilder(t, tx).Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("topic-handler").Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).WithSpaceID(spaceID).WithUserID(userID).Build()
	for number := int32(1); number <= topicCount; number++ {
		testutil.NewTopicBuilder(t, tx).WithSpaceID(spaceID).WithNumber(number).WithName(fmt.Sprintf("トピック%d", number)).Build()
	}

	ctx := middleware.SetAPIPrincipalToContext(context.Background(), &model.APIPrincipal{
		User:        &model.User{ID: userID},
		Space:       &model.Space{ID: spaceID, Identifier: "topic-handler"},
		SpaceMember: &model.SpaceMember{ID: spaceMemberID, SpaceID: spaceID, UserID: userID, Scopes: []model.Scope{model.ScopeSpaceAdmin}, Active: true},
		TokenKind:   model.APITokenKindPersonalAccessToken,
		Scopes:      []model.Scope{model.ScopeTopicRead},
	})
	requestURL, err := url.Parse("/api/v1/spaces/topic-handler/topics")
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	ctx = apipagination.WithRequestURL(ctx, requestURL)

	q := testutil.QueriesWithTx(tx)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	h := topic.NewHandler(
		usecase.NewListAPITopicsUsecase(topicRepo, topicMemberRepo),
		usecase.NewGetAPITopicUsecase(topicRepo, topicMemberRepo),
	)
	return h, ctx
}

func listTopics(t *testing.T, h *topic.Handler, ctx context.Context, params apigen.ListTopicsParams) apigen.ListTopics200JSONResponse {
	t.Helper()

	res, err := h.ListTopics(ctx, apigen.ListTopicsRequestObject{SpaceIdentifier: "topic-handler", Params: params})
	if err != nil {
		t.Fatalf("ListTopics() error = %v", err)
	}
	jsonRes, ok := res.(apigen.ListTopics200JSONResponse)
	if !ok {
		t.Fatalf("レスポンスの型 = %T、期待値 = ListTopics200JSONResponse", res)
	}
	return jsonRes
}

func TestHandler_ListTopics(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h, ctx := setupTopicHandler(t, tx, 3)

	// サブテストは親のトランザクションを共有するため、並列にしない

	t.Run("次のページのカーソルとLinkヘッダーを返し、カーソルで続きを辿れる", func(t *testing.T) {
		limit := int32(2)
		first := listTopics(t, h, ctx, apigen.ListTopicsParams{Limit: &limit})
		if len(first.Body.Items) != 2 || first.Body.Items[0].Number != 1 || first.Body.Items[1].Number != 2 {
			t.Fatalf("1ページ目 = %+v、期待値 = トピック1と2", first.Body.Items)
		}
		if first.Body.NextCursor == nil {
			t.Fatal("next_cursor = null、期待値 = 次のページのカーソル")
		}
		wantLink := `</api/v1/spaces/topic-handler/topics?cursor=` + url.QueryEscape(*first.Body.NextCursor) + `>; rel="next"`
		if first.Headers.Link == nil || *first.Headers.Link != wantLink {
			t.Errorf("Link = %v、期待値 = %q", first.Headers.Link, wantLink)
		}

		second := listTopics(t, h, ctx, apigen.ListTopicsParams{Cursor: first.Body.NextCursor, Limit: &limit})
		if len(second.Body.Items) != 1 || second.Body.Items[0].Number != 3 {
			t.Fatalf("2ページ目 = %+v、期待値 = トピック3", second.Body.Items)
		}
		if second.Body.NextCursor != nil {
			t.Errorf("next_cursor = %q、期待値 = null", *second.Body.NextCursor)
		}
		if second.Headers.Link != nil {
			t.Errorf("Link = %q、期待値 = 無し", *second.Headers.Link)
		}
	})

	t.Run("トピックの項目を返す", func(t *testing.T) {
		res := listTopics(t, h, ctx, apigen.ListTopicsParams{})
		if len(res.Body.Items) != 3 {
			t.Fatalf("件数 = %d、期待値 = 3", len(res.Body.Items))
		}
		item := res.Body.Items[0]
		if item.Name != "トピック1" || item.Visibility != apigen.TopicVisibility("public") || item.Id.String() == "" {
			t.Errorf("トピック = %+v", item)
		}
	})

	invalidCursorTests := []struct {
		name   string
		cursor string
	}{
		{name: "読めないカーソルはcursorパラメーターのエラー", cursor: "invalid!"},
		{name: "番号を含まないカーソルもcursorパラメーターのエラー", cursor: "e30"}, // {}
	}
	for _, tt := range invalidCursorTests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := h.ListTopics(ctx, apigen.ListTopicsRequestObject{
				SpaceIdentifier: "topic-handler",
				Params:          apigen.ListTopicsParams{Cursor: &tt.cursor},
			})
			if res != nil {
				t.Errorf("レスポンス = %v、期待値 = nil", res)
			}
			var pe *apierror.ParameterError
			if !errors.As(err, &pe) || pe.Parameter != "cursor" {
				t.Errorf("error = %v、期待値 = cursorのParameterError", err)
			}
		})
	}

	t.Run("束縛先と異なるスペースは未存在のエラーを返す", func(t *testing.T) {
		_, err := h.ListTopics(ctx, apigen.ListTopicsRequestObject{SpaceIdentifier: "other-space"})
		if ae := model.AsAppError(err); ae == nil || ae.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("error = %v、期待値 = AppErrCodeResourceNotFound", err)
		}
	})
}

func TestHandler_ListTopics_LimitIsCapped(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h, ctx := setupTopicHandler(t, tx, apipagination.MaxLimit+1)

	limit := apipagination.MaxLimit * 5
	res := listTopics(t, h, ctx, apigen.ListTopicsParams{Limit: &limit})
	if got := int32(len(res.Body.Items)); got != apipagination.MaxLimit {
		t.Errorf("件数 = %d、期待値 = %d", got, apipagination.MaxLimit)
	}
	if res.Body.NextCursor == nil {
		t.Error("next_cursor = null、期待値 = 次のページのカーソル")
	}
	if res.Headers.Link == nil {
		t.Error("Link = 無し、期待値 = 次のページのURL")
	}
}
