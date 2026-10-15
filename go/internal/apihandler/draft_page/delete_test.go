package draft_page_test

import (
	"errors"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/apierror"
	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/model"
)

func TestHandler_DeleteDraftPage(t *testing.T) {
	t.Parallel()

	const identifier = "draft-delete-handler"
	h, ctx := newDraftPageWriteHandler(t, identifier)

	// サブテストは同じ下書きを削除するため、並列にしない。
	t.Run("If-Matchが無ければUseCaseを呼ばずにErrPreconditionRequiredを返す", func(t *testing.T) {
		_, err := h.DeleteDraftPage(ctx, apigen.DeleteDraftPageRequestObject{SpaceIdentifier: identifier, PageNumber: 1})
		if !errors.Is(err, apierror.ErrPreconditionRequired) {
			t.Errorf("エラー = %v、期待値 = ErrPreconditionRequired", err)
		}
	})

	t.Run("取得と同じETagなら削除して204を返す", func(t *testing.T) {
		etag := draftPageETag(t, h, ctx, identifier)
		res, err := h.DeleteDraftPage(ctx, apigen.DeleteDraftPageRequestObject{
			SpaceIdentifier: identifier,
			PageNumber:      1,
			Params:          apigen.DeleteDraftPageParams{IfMatch: &etag},
		})
		if err != nil {
			t.Fatalf("DeleteDraftPage()のエラー = %v", err)
		}
		if _, ok := res.(apigen.DeleteDraftPage204Response); !ok {
			t.Errorf("応答の型 = %T、期待値 = DeleteDraftPage204Response", res)
		}

		_, err = h.GetDraftPage(ctx, apigen.GetDraftPageRequestObject{SpaceIdentifier: identifier, PageNumber: 1})
		if ae := model.AsAppError(err); ae == nil || ae.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("削除後の取得のエラー = %v、期待値 = AppErrCodeResourceNotFound", err)
		}
	})
}
