package draft_page_test

import (
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestHandler_GetDraftPage(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h, ctx := setupDraftPageHandler(t, tx, 2)

	getDraftPage := func(t *testing.T, number int32, ifNoneMatch *string) apigen.GetDraftPageResponseObject {
		t.Helper()
		res, err := h.GetDraftPage(ctx, apigen.GetDraftPageRequestObject{
			SpaceIdentifier: "draft-page-handler",
			PageNumber:      number,
			Params:          apigen.GetDraftPageParams{IfNoneMatch: ifNoneMatch},
		})
		if err != nil {
			t.Fatalf("GetDraftPage()のエラー = %v", err)
		}
		return res
	}
	get200 := func(t *testing.T, number int32, ifNoneMatch *string) apigen.GetDraftPage200JSONResponse {
		t.Helper()
		res := getDraftPage(t, number, ifNoneMatch)
		jsonRes, ok := res.(apigen.GetDraftPage200JSONResponse)
		if !ok {
			t.Fatalf("レスポンスの型 = %T、期待値 = GetDraftPage200JSONResponse", res)
		}
		return jsonRes
	}

	// サブテストは親のトランザクションを共有するため、並列にしない

	t.Run("下書きとETagを返す", func(t *testing.T) {
		res := get200(t, 2, nil)
		if res.Body.PageNumber != 2 || res.Body.TopicNumber != 2 || res.Body.TopicName != "トピック2" || res.Body.Title == nil || *res.Body.Title != "下書き2" || res.Body.Body != "下書きの本文2" {
			t.Errorf("下書き = %+v", res.Body)
		}
		if len(res.Headers.ETag) < 3 || res.Headers.ETag[0] != '"' || res.Headers.ETag[len(res.Headers.ETag)-1] != '"' {
			t.Errorf("ETag = %q、期待値 = 引用符で囲んだ強いエンティティタグ", res.Headers.ETag)
		}
		if other := get200(t, 1, nil); other.Headers.ETag == res.Headers.ETag {
			t.Errorf("内容の違う下書きのETagが同じ: %q", res.Headers.ETag)
		}
	})

	etag := get200(t, 2, nil).Headers.ETag
	notModifiedTests := []struct {
		name        string
		ifNoneMatch string
	}{
		{name: "ETagが一致すれば304", ifNoneMatch: etag},
		{name: "弱いエンティティタグとしても一致すれば304", ifNoneMatch: "W/" + etag},
		{name: "並びのどれかに一致すれば304", ifNoneMatch: `"other", ` + etag},
		{name: "*は304", ifNoneMatch: "*"},
	}
	for _, tt := range notModifiedTests {
		t.Run(tt.name, func(t *testing.T) {
			res := getDraftPage(t, 2, &tt.ifNoneMatch)
			notModified, ok := res.(apigen.GetDraftPage304Response)
			if !ok {
				t.Fatalf("レスポンスの型 = %T、期待値 = GetDraftPage304Response", res)
			}
			if notModified.Headers.ETag != etag {
				t.Errorf("ETag = %q、期待値 = %q", notModified.Headers.ETag, etag)
			}
		})
	}

	t.Run("ETagが一致しなければ200", func(t *testing.T) {
		ifNoneMatch := `"stale"`
		if res := get200(t, 2, &ifNoneMatch); res.Headers.ETag != etag {
			t.Errorf("ETag = %q、期待値 = %q", res.Headers.ETag, etag)
		}
	})

	t.Run("本文が同じでも更新日時が変われば旧ETagに200を返す", func(t *testing.T) {
		// Web画面の自動保存は内容が同じでも下書きの更新日時を変える
		modifiedAt := draftPageHandlerBaseTime.Add(time.Hour)
		spaceID := middleware.APIPrincipalFromContext(ctx).Space.ID
		if _, err := tx.ExecContext(ctx,
			"UPDATE draft_pages SET modified_at = $1 WHERE space_id = $2 AND page_id = (SELECT id FROM pages WHERE space_id = $2 AND number = $3)",
			modifiedAt, spaceID, 2,
		); err != nil {
			t.Fatalf("更新日時の変更に失敗: %v", err)
		}

		res := get200(t, 2, &etag)
		if res.Headers.ETag == etag {
			t.Errorf("ETag = %q、期待値 = 更新前とは別の値", res.Headers.ETag)
		}
		if !res.Body.ModifiedAt.Equal(modifiedAt) {
			t.Errorf("ModifiedAt = %s、期待値 = %s", res.Body.ModifiedAt, modifiedAt)
		}
	})

	t.Run("下書きの無いページは未存在のエラーを返す", func(t *testing.T) {
		res, err := h.GetDraftPage(ctx, apigen.GetDraftPageRequestObject{SpaceIdentifier: "draft-page-handler", PageNumber: 99})
		if res != nil {
			t.Errorf("レスポンス = %v、期待値 = nil", res)
		}
		if ae := model.AsAppError(err); ae == nil || ae.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("error = %v、期待値 = AppErrCodeResourceNotFound", err)
		}
	})
}
