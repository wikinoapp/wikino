package page_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/apierror"
	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/apihandler/page"
	"github.com/wikinoapp/wikino/go/internal/apipagination"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// pageHandlerBaseTimeはページの更新日時の基準。ページ番号nの更新日時は基準のn時間前にする
var pageHandlerBaseTime = time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)

// setupPageHandlerは、公開トピック (番号1) に公開済みのページをpageCount件持つスペースと、
// そのスペースの管理者が `page:read` を持つトークンで呼び出したcontextを用意する
func setupPageHandler(t *testing.T, tx *sql.Tx, pageCount int32) (*page.Handler, context.Context) {
	t.Helper()

	userID := testutil.NewUserBuilder(t, tx).Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("page-handler").Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).WithSpaceID(spaceID).WithUserID(userID).Build()
	topicID := testutil.NewTopicBuilder(t, tx).WithSpaceID(spaceID).WithNumber(1).WithName("トピック1").Build()
	for number := int32(1); number <= pageCount; number++ {
		testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(model.PageNumber(number)).
			WithTitle(fmt.Sprintf("ページ%d", number)).
			WithBody(fmt.Sprintf("本文%d", number)).
			WithModifiedAt(pageHandlerBaseTime.Add(-time.Duration(number) * time.Hour)).
			Build()
	}

	ctx := middleware.SetAPIPrincipalToContext(context.Background(), &model.APIPrincipal{
		User:        &model.User{ID: userID},
		Space:       &model.Space{ID: spaceID, Identifier: "page-handler"},
		SpaceMember: &model.SpaceMember{ID: spaceMemberID, SpaceID: spaceID, UserID: userID, Role: model.SpaceRoleAdmin, Active: true},
		TokenKind:   model.APITokenKindPersonalAccessToken,
		Scopes:      []model.Scope{model.ScopePageRead},
	})
	requestURL, err := url.Parse("/api/v1/spaces/page-handler/pages?topic_number=1")
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	ctx = apipagination.WithRequestURL(ctx, requestURL)

	q := testutil.QueriesWithTx(tx)
	pageRepo := repository.NewPageRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	h := page.NewHandler(
		usecase.NewListAPIPagesUsecase(pageRepo, topicRepo, topicMemberRepo),
		usecase.NewGetAPIPageUsecase(pageRepo, topicRepo, topicMemberRepo),
		nil,
		nil,
	)
	return h, ctx
}

func listPages(t *testing.T, h *page.Handler, ctx context.Context, params apigen.ListPagesParams) apigen.ListPages200JSONResponse {
	t.Helper()

	res, err := h.ListPages(ctx, apigen.ListPagesRequestObject{SpaceIdentifier: "page-handler", Params: params})
	if err != nil {
		t.Fatalf("ListPages() error = %v", err)
	}
	jsonRes, ok := res.(apigen.ListPages200JSONResponse)
	if !ok {
		t.Fatalf("レスポンスの型 = %T、期待値 = ListPages200JSONResponse", res)
	}
	return jsonRes
}

func TestHandler_ListPages(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h, ctx := setupPageHandler(t, tx, 3)

	// サブテストは親のトランザクションを共有するため、並列にしない

	t.Run("次のページのカーソルとLinkヘッダーを返し、カーソルで続きを辿れる", func(t *testing.T) {
		limit := int32(2)
		first := listPages(t, h, ctx, apigen.ListPagesParams{Limit: &limit})
		if len(first.Body.Items) != 2 || first.Body.Items[0].Number != 1 || first.Body.Items[1].Number != 2 {
			t.Fatalf("1ページ目 = %+v、期待値 = ページ1と2", first.Body.Items)
		}
		if first.Body.NextCursor == nil {
			t.Fatal("next_cursor = null、期待値 = 次のページのカーソル")
		}
		// 絞り込みのクエリはそのまま引き継ぐ
		wantLink := `</api/v1/spaces/page-handler/pages?cursor=` + url.QueryEscape(*first.Body.NextCursor) + `&topic_number=1>; rel="next"`
		if first.Headers.Link == nil || *first.Headers.Link != wantLink {
			t.Errorf("Link = %v、期待値 = %q", first.Headers.Link, wantLink)
		}

		second := listPages(t, h, ctx, apigen.ListPagesParams{Cursor: first.Body.NextCursor, Limit: &limit})
		if len(second.Body.Items) != 1 || second.Body.Items[0].Number != 3 {
			t.Fatalf("2ページ目 = %+v、期待値 = ページ3", second.Body.Items)
		}
		if second.Body.NextCursor != nil {
			t.Errorf("next_cursor = %q、期待値 = null", *second.Body.NextCursor)
		}
		if second.Headers.Link != nil {
			t.Errorf("Link = %q、期待値 = 無し", *second.Headers.Link)
		}
	})

	t.Run("ページの項目を返す", func(t *testing.T) {
		res := listPages(t, h, ctx, apigen.ListPagesParams{})
		if len(res.Body.Items) != 3 {
			t.Fatalf("件数 = %d、期待値 = 3", len(res.Body.Items))
		}
		item := res.Body.Items[0]
		if item.Id.String() == "" || item.TopicNumber != 1 || item.Title == nil || *item.Title != "ページ1" || item.Body != "本文1" {
			t.Errorf("ページ = %+v", item)
		}
		if want := pageHandlerBaseTime.Add(-time.Hour); !item.ModifiedAt.Equal(want) || item.ModifiedAt.Location() != time.UTC {
			t.Errorf("modified_at = %v、期待値 = UTCの %v", item.ModifiedAt, want)
		}
	})

	t.Run("modified_sinceで絞り込む", func(t *testing.T) {
		since := pageHandlerBaseTime.Add(-2 * time.Hour)
		res := listPages(t, h, ctx, apigen.ListPagesParams{ModifiedSince: &since})
		if len(res.Body.Items) != 2 || res.Body.Items[0].Number != 1 || res.Body.Items[1].Number != 2 {
			t.Errorf("一覧 = %+v、期待値 = ページ1と2", res.Body.Items)
		}
	})

	nonUUIDCursor, err := apipagination.EncodeCursor(map[string]string{"m": "2026-09-01T00:00:00Z", "i": "not-a-uuid"})
	if err != nil {
		t.Fatalf("EncodeCursor() error = %v", err)
	}
	invalidCursorTests := []struct {
		name   string
		cursor string
	}{
		{name: "読めないカーソルはcursorパラメーターのエラー", cursor: "invalid!"},
		{name: "位置を含まないカーソルもcursorパラメーターのエラー", cursor: "e30"}, // {}
		{name: "UUIDでないIDを含むカーソルもcursorパラメーターのエラー", cursor: nonUUIDCursor},
	}
	for _, tt := range invalidCursorTests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := h.ListPages(ctx, apigen.ListPagesRequestObject{
				SpaceIdentifier: "page-handler",
				Params:          apigen.ListPagesParams{Cursor: &tt.cursor},
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
		_, err := h.ListPages(ctx, apigen.ListPagesRequestObject{SpaceIdentifier: "other-space"})
		if ae := model.AsAppError(err); ae == nil || ae.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("error = %v、期待値 = AppErrCodeResourceNotFound", err)
		}
	})
}

func TestHandler_ListPages_LimitIsCapped(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h, ctx := setupPageHandler(t, tx, apipagination.MaxLimit+1)

	limit := apipagination.MaxLimit * 5
	res := listPages(t, h, ctx, apigen.ListPagesParams{Limit: &limit})
	if got := int32(len(res.Body.Items)); got != apipagination.MaxLimit {
		t.Errorf("件数 = %d、期待値 = %d", got, apipagination.MaxLimit)
	}
	if res.Body.NextCursor == nil {
		t.Error("next_cursor = null、期待値 = 次のページのカーソル")
	}
}
