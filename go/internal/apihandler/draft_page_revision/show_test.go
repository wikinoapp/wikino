package draft_page_revision_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestHandler_GetDraftPageRevision(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h, ctx, revisionIDs := setupDraftPageRevisionHandler(t, tx, 2)

	// サブテストは親のトランザクションを共有するため、並列にしない

	t.Run("リビジョンを返す", func(t *testing.T) {
		res, err := h.GetDraftPageRevision(ctx, apigen.GetDraftPageRevisionRequestObject{
			SpaceIdentifier:     "draft-page-revision-handler",
			PageNumber:          1,
			DraftPageRevisionId: uuid.MustParse(string(revisionIDs[1])),
		})
		if err != nil {
			t.Fatalf("GetDraftPageRevision()のエラー = %v", err)
		}
		jsonRes, ok := res.(apigen.GetDraftPageRevision200JSONResponse)
		if !ok {
			t.Fatalf("レスポンスの型 = %T、期待値 = GetDraftPageRevision200JSONResponse", res)
		}
		if got := jsonRes.Body; got.Id.String() != string(revisionIDs[1]) || got.Title == nil || *got.Title != "リビジョン1" || got.Body != "本文1" {
			t.Errorf("リビジョン = %+v、期待値 = リビジョン1", got)
		}
	})

	t.Run("存在しないリビジョンは未存在のエラーを返す", func(t *testing.T) {
		res, err := h.GetDraftPageRevision(ctx, apigen.GetDraftPageRevisionRequestObject{
			SpaceIdentifier:     "draft-page-revision-handler",
			PageNumber:          1,
			DraftPageRevisionId: uuid.New(),
		})
		if res != nil {
			t.Errorf("レスポンス = %v、期待値 = nil", res)
		}
		if ae := model.AsAppError(err); ae == nil || ae.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("error = %v、期待値 = AppErrCodeResourceNotFound", err)
		}
	})
}
