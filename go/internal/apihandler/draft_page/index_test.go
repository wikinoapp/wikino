package draft_page_test

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
	"github.com/wikinoapp/wikino/go/internal/apihandler/draft_page"
	"github.com/wikinoapp/wikino/go/internal/apipagination"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// draftPageHandlerBaseTimeは下書きの更新日時の基準。ページ番号nの下書きの更新日時は基準のn時間前にする
var draftPageHandlerBaseTime = time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)

// setupDraftPageHandlerは、公開トピック (番号2) にページをdraftCount件と、その全ページの
// トークンの持ち主の下書きを持つスペースと、そのスペースの管理者が `draft_page:read` を持つトークンで
// 呼び出したcontextを用意する
func setupDraftPageHandler(t *testing.T, tx *sql.Tx, draftCount int32) (*draft_page.Handler, context.Context) {
	t.Helper()

	userID := testutil.NewUserBuilder(t, tx).Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("draft-page-handler").Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).WithSpaceID(spaceID).WithUserID(userID).Build()
	topicID := testutil.NewTopicBuilder(t, tx).WithSpaceID(spaceID).WithNumber(2).WithName("トピック2").Build()
	for number := int32(1); number <= draftCount; number++ {
		modifiedAt := draftPageHandlerBaseTime.Add(-time.Duration(number) * time.Hour)
		pageID := testutil.NewPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithTopicID(topicID).
			WithNumber(model.PageNumber(number)).
			WithTitle(fmt.Sprintf("ページ%d", number)).
			WithModifiedAt(modifiedAt).
			Build()
		testutil.NewDraftPageBuilder(t, tx).
			WithSpaceID(spaceID).
			WithPageID(pageID).
			WithSpaceMemberID(spaceMemberID).
			WithTopicID(topicID).
			WithTitle(fmt.Sprintf("下書き%d", number)).
			WithBody(fmt.Sprintf("下書きの本文%d", number)).
			WithModifiedAt(modifiedAt).
			Build()
	}

	ctx := middleware.SetAPIPrincipalToContext(context.Background(), &model.APIPrincipal{
		User:        &model.User{ID: userID},
		Space:       &model.Space{ID: spaceID, Identifier: "draft-page-handler"},
		SpaceMember: &model.SpaceMember{ID: spaceMemberID, SpaceID: spaceID, UserID: userID, Role: model.SpaceRoleAdmin, Active: true},
		TokenKind:   model.APITokenKindPersonalAccessToken,
		Scopes:      []model.Scope{model.ScopeDraftPageRead},
	})
	requestURL, err := url.Parse("/api/v1/spaces/draft-page-handler/draft_pages")
	if err != nil {
		t.Fatalf("url.Parse()のエラー = %v", err)
	}
	ctx = apipagination.WithRequestURL(ctx, requestURL)

	q := testutil.QueriesWithTx(tx)
	pageRepo := repository.NewPageRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	h := draft_page.NewHandler(
		"https://example.com",
		usecase.NewListAPIDraftPagesUsecase(draftPageRepo, topicRepo, topicMemberRepo),
		usecase.NewGetAPIDraftPageUsecase(pageRepo, draftPageRepo, topicRepo, topicMemberRepo),
		nil,
		nil,
		nil,
	)
	return h, ctx
}

func listDraftPages(t *testing.T, h *draft_page.Handler, ctx context.Context, params apigen.ListDraftPagesParams) apigen.ListDraftPages200JSONResponse {
	t.Helper()

	res, err := h.ListDraftPages(ctx, apigen.ListDraftPagesRequestObject{SpaceIdentifier: "draft-page-handler", Params: params})
	if err != nil {
		t.Fatalf("ListDraftPages()のエラー = %v", err)
	}
	jsonRes, ok := res.(apigen.ListDraftPages200JSONResponse)
	if !ok {
		t.Fatalf("レスポンスの型 = %T、期待値 = ListDraftPages200JSONResponse", res)
	}
	return jsonRes
}

func TestHandler_ListDraftPages(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h, ctx := setupDraftPageHandler(t, tx, 3)

	// サブテストは親のトランザクションを共有するため、並列にしない

	t.Run("次のページのカーソルとLinkヘッダーを返し、カーソルで続きを辿れる", func(t *testing.T) {
		limit := int32(2)
		first := listDraftPages(t, h, ctx, apigen.ListDraftPagesParams{Limit: &limit})
		if len(first.Body.Items) != 2 || first.Body.Items[0].PageNumber != 1 || first.Body.Items[1].PageNumber != 2 {
			t.Fatalf("一覧 = %+v、期待値 = 1・2", first.Body.Items)
		}
		if first.Body.NextCursor == nil {
			t.Fatal("NextCursor = nil、期待値 = 次のページのカーソル")
		}
		wantLink := `</api/v1/spaces/draft-page-handler/draft_pages?cursor=` + url.QueryEscape(*first.Body.NextCursor) + `>; rel="next"`
		if first.Headers.Link == nil || *first.Headers.Link != wantLink {
			t.Errorf("Link = %v、期待値 = %q", first.Headers.Link, wantLink)
		}

		second := listDraftPages(t, h, ctx, apigen.ListDraftPagesParams{Limit: &limit, Cursor: first.Body.NextCursor})
		if len(second.Body.Items) != 1 || second.Body.Items[0].PageNumber != 3 {
			t.Errorf("次のページ = %+v、期待値 = 3のみ", second.Body.Items)
		}
		if second.Body.NextCursor != nil {
			t.Errorf("NextCursor = %q、期待値 = nil", *second.Body.NextCursor)
		}
		if second.Headers.Link != nil {
			t.Errorf("Link = %q、期待値 = 無し", *second.Headers.Link)
		}
	})

	t.Run("下書きの項目を返す", func(t *testing.T) {
		item := listDraftPages(t, h, ctx, apigen.ListDraftPagesParams{}).Body.Items[0]
		if item.PageNumber != 1 || item.TopicNumber != 2 || item.TopicName != "トピック2" || item.Title == nil || *item.Title != "下書き1" || item.Body != "下書きの本文1" {
			t.Errorf("下書き = %+v", item)
		}
		if want := draftPageHandlerBaseTime.Add(-time.Hour); !item.ModifiedAt.Equal(want) || item.ModifiedAt.Location() != time.UTC {
			t.Errorf("ModifiedAt = %s、期待値 = UTCの%s", item.ModifiedAt, want)
		}
	})

	nonUUIDCursor, err := apipagination.EncodeCursor(map[string]string{"m": "2026-09-01T00:00:00Z", "i": "not-a-uuid"})
	if err != nil {
		t.Fatalf("EncodeCursor()のエラー = %v", err)
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
			res, err := h.ListDraftPages(ctx, apigen.ListDraftPagesRequestObject{
				SpaceIdentifier: "draft-page-handler",
				Params:          apigen.ListDraftPagesParams{Cursor: &tt.cursor},
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

	t.Run("トークンが属さないスペースは、カーソルが不正でも未存在のエラーを返す", func(t *testing.T) {
		cursor := "invalid!"
		_, err := h.ListDraftPages(ctx, apigen.ListDraftPagesRequestObject{
			SpaceIdentifier: "other-space",
			Params:          apigen.ListDraftPagesParams{Cursor: &cursor},
		})
		if ae := model.AsAppError(err); ae == nil || ae.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("error = %v、期待値 = AppErrCodeResourceNotFound", err)
		}
	})
}
