package draft_page_revision_test

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
	"github.com/wikinoapp/wikino/go/internal/apihandler/draft_page_revision"
	"github.com/wikinoapp/wikino/go/internal/apipagination"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/testutil"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// revisionHandlerBaseTimeはリビジョンの作成日時の基準。n件目のリビジョンの作成日時は基準のn分後にする
var revisionHandlerBaseTime = time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)

// setupDraftPageRevisionHandlerは、ページ1にトークンの持ち主の下書きとそのリビジョンをrevisionCount件持つスペースと、
// そのスペースの管理者が `draft_page:read` を持つトークンで呼び出したcontextを用意する。
// n件目のリビジョンのタイトルは「リビジョンn」で、0件目だけはタイトルの無い下書きの記録 (空のタイトル) にする
func setupDraftPageRevisionHandler(t *testing.T, tx *sql.Tx, revisionCount int) (*draft_page_revision.Handler, context.Context, []model.DraftPageRevisionID) {
	t.Helper()

	userID := testutil.NewUserBuilder(t, tx).Build()
	spaceID := testutil.NewSpaceBuilder(t, tx).WithIdentifier("draft-page-revision-handler").Build()
	spaceMemberID := testutil.NewSpaceMemberBuilder(t, tx).WithSpaceID(spaceID).WithUserID(userID).Build()
	topicID := testutil.NewTopicBuilder(t, tx).WithSpaceID(spaceID).WithNumber(1).WithName("トピック").Build()
	pageID := testutil.NewPageBuilder(t, tx).WithSpaceID(spaceID).WithTopicID(topicID).WithNumber(1).WithTitle("ページ1").Build()
	draftPageID := testutil.NewDraftPageBuilder(t, tx).
		WithSpaceID(spaceID).
		WithPageID(pageID).
		WithSpaceMemberID(spaceMemberID).
		WithTopicID(topicID).
		Build()

	revisionIDs := make([]model.DraftPageRevisionID, revisionCount)
	for i := range revisionCount {
		title := fmt.Sprintf("リビジョン%d", i)
		if i == 0 {
			title = ""
		}
		revisionIDs[i] = testutil.NewDraftPageRevisionBuilder(t, tx).
			WithDraftPageID(draftPageID).
			WithSpaceID(spaceID).
			WithSpaceMemberID(spaceMemberID).
			WithTitle(title).
			WithBody(fmt.Sprintf("本文%d", i)).
			WithCreatedAt(revisionHandlerBaseTime.Add(time.Duration(i) * time.Minute)).
			Build()
	}

	ctx := middleware.SetAPIPrincipalToContext(context.Background(), &model.APIPrincipal{
		User:        &model.User{ID: userID},
		Space:       &model.Space{ID: spaceID, Identifier: "draft-page-revision-handler"},
		SpaceMember: &model.SpaceMember{ID: spaceMemberID, SpaceID: spaceID, UserID: userID, Role: model.SpaceRoleAdmin, Active: true},
		TokenKind:   model.APITokenKindPersonalAccessToken,
		Scopes:      []model.Scope{model.ScopeDraftPageRead},
	})
	requestURL, err := url.Parse("/api/v1/spaces/draft-page-revision-handler/pages/1/draft_page_revisions")
	if err != nil {
		t.Fatalf("url.Parse()のエラー = %v", err)
	}
	ctx = apipagination.WithRequestURL(ctx, requestURL)

	q := testutil.QueriesWithTx(tx)
	pageRepo := repository.NewPageRepository(q)
	draftPageRepo := repository.NewDraftPageRepository(q)
	draftPageRevisionRepo := repository.NewDraftPageRevisionRepository(q)
	topicRepo := repository.NewTopicRepository(q)
	topicMemberRepo := repository.NewTopicMemberRepository(q)
	h := draft_page_revision.NewHandler(
		usecase.NewListAPIDraftPageRevisionsUsecase(pageRepo, draftPageRepo, draftPageRevisionRepo, topicRepo, topicMemberRepo),
		usecase.NewGetAPIDraftPageRevisionUsecase(pageRepo, draftPageRepo, draftPageRevisionRepo, topicRepo, topicMemberRepo),
	)
	return h, ctx, revisionIDs
}

func listDraftPageRevisions(t *testing.T, h *draft_page_revision.Handler, ctx context.Context, params apigen.ListDraftPageRevisionsParams) apigen.ListDraftPageRevisions200JSONResponse {
	t.Helper()

	res, err := h.ListDraftPageRevisions(ctx, apigen.ListDraftPageRevisionsRequestObject{
		SpaceIdentifier: "draft-page-revision-handler",
		PageNumber:      1,
		Params:          params,
	})
	if err != nil {
		t.Fatalf("ListDraftPageRevisions()のエラー = %v", err)
	}
	jsonRes, ok := res.(apigen.ListDraftPageRevisions200JSONResponse)
	if !ok {
		t.Fatalf("レスポンスの型 = %T、期待値 = ListDraftPageRevisions200JSONResponse", res)
	}
	return jsonRes
}

func TestHandler_ListDraftPageRevisions(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h, ctx, revisionIDs := setupDraftPageRevisionHandler(t, tx, 3)

	// サブテストは親のトランザクションを共有するため、並列にしない

	t.Run("次のページのカーソルとLinkヘッダーを返し、カーソルで続きを辿れる", func(t *testing.T) {
		limit := int32(2)
		first := listDraftPageRevisions(t, h, ctx, apigen.ListDraftPageRevisionsParams{Limit: &limit})
		if len(first.Body.Items) != 2 || first.Body.Items[0].Id.String() != string(revisionIDs[2]) || first.Body.Items[1].Id.String() != string(revisionIDs[1]) {
			t.Fatalf("一覧 = %+v、期待値 = リビジョン2・1", first.Body.Items)
		}
		if first.Body.NextCursor == nil {
			t.Fatal("NextCursor = nil、期待値 = 次のページのカーソル")
		}
		wantLink := `</api/v1/spaces/draft-page-revision-handler/pages/1/draft_page_revisions?cursor=` + url.QueryEscape(*first.Body.NextCursor) + `>; rel="next"`
		if first.Headers.Link == nil || *first.Headers.Link != wantLink {
			t.Errorf("Link = %v、期待値 = %q", first.Headers.Link, wantLink)
		}

		second := listDraftPageRevisions(t, h, ctx, apigen.ListDraftPageRevisionsParams{Limit: &limit, Cursor: first.Body.NextCursor})
		if len(second.Body.Items) != 1 || second.Body.Items[0].Id.String() != string(revisionIDs[0]) {
			t.Errorf("次のページ = %+v、期待値 = リビジョン0のみ", second.Body.Items)
		}
		if second.Body.NextCursor != nil {
			t.Errorf("NextCursor = %q、期待値 = nil", *second.Body.NextCursor)
		}
		if second.Headers.Link != nil {
			t.Errorf("Link = %q、期待値 = 無し", *second.Headers.Link)
		}
	})

	t.Run("リビジョンの項目を返し、空のタイトルはnullにする", func(t *testing.T) {
		items := listDraftPageRevisions(t, h, ctx, apigen.ListDraftPageRevisionsParams{}).Body.Items
		if len(items) != 3 {
			t.Fatalf("len(items) = %d、期待値 = 3", len(items))
		}
		newest := items[0]
		if newest.Title == nil || *newest.Title != "リビジョン2" || newest.Excerpt != "本文2" || newest.Body != nil {
			t.Errorf("リビジョン = %+v、期待値 = 抜粋を含み本文を含まないリビジョン2", newest)
		}
		if want := revisionHandlerBaseTime.Add(2 * time.Minute); !newest.CreatedAt.Equal(want) || newest.CreatedAt.Location() != time.UTC {
			t.Errorf("CreatedAt = %s、期待値 = UTCの%s", newest.CreatedAt, want)
		}
		if oldest := items[2]; oldest.Title != nil {
			t.Errorf("空のタイトルのリビジョンのTitle = %q、期待値 = nil", *oldest.Title)
		}
	})

	t.Run("includeにbodyを指定すると本文を返す", func(t *testing.T) {
		newest := listDraftPageRevisions(t, h, ctx, apigen.ListDraftPageRevisionsParams{Include: &apigen.ListInclude{apigen.Body}}).Body.Items[0]
		if newest.Body == nil || *newest.Body != "本文2" {
			t.Errorf("リビジョン = %+v、期待値 = 本文を含むリビジョン2", newest)
		}
	})

	nonUUIDCursor, err := apipagination.EncodeCursor(map[string]string{"c": "2026-09-01T00:00:00Z", "i": "not-a-uuid"})
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
			res, err := h.ListDraftPageRevisions(ctx, apigen.ListDraftPageRevisionsRequestObject{
				SpaceIdentifier: "draft-page-revision-handler",
				PageNumber:      1,
				Params:          apigen.ListDraftPageRevisionsParams{Cursor: &tt.cursor},
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
		_, err := h.ListDraftPageRevisions(ctx, apigen.ListDraftPageRevisionsRequestObject{
			SpaceIdentifier: "other-space",
			PageNumber:      1,
			Params:          apigen.ListDraftPageRevisionsParams{Cursor: &cursor},
		})
		if ae := model.AsAppError(err); ae == nil || ae.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("error = %v、期待値 = AppErrCodeResourceNotFound", err)
		}
	})
}
