package page_test

import (
	"testing"
	"time"

	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestHandler_GetPage(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h, ctx := setupPageHandler(t, tx, 2)

	getPage := func(t *testing.T, number int32, ifNoneMatch *string) apigen.GetPageResponseObject {
		t.Helper()
		res, err := h.GetPage(ctx, apigen.GetPageRequestObject{
			SpaceIdentifier: "page-handler",
			PageNumber:      number,
			Params:          apigen.GetPageParams{IfNoneMatch: ifNoneMatch},
		})
		if err != nil {
			t.Fatalf("GetPage() error = %v", err)
		}
		return res
	}
	get200 := func(t *testing.T, number int32, ifNoneMatch *string) apigen.GetPage200JSONResponse {
		t.Helper()
		res := getPage(t, number, ifNoneMatch)
		jsonRes, ok := res.(apigen.GetPage200JSONResponse)
		if !ok {
			t.Fatalf("レスポンスの型 = %T、期待値 = GetPage200JSONResponse", res)
		}
		return jsonRes
	}

	// サブテストは親のトランザクションを共有するため、並列にしない

	t.Run("ページとETagを返す", func(t *testing.T) {
		res := get200(t, 2, nil)
		if res.Body.Number != 2 || res.Body.TopicNumber != 1 || res.Body.Title == nil || *res.Body.Title != "ページ2" || res.Body.Body != "本文2" {
			t.Errorf("ページ = %+v", res.Body)
		}
		if len(res.Headers.ETag) < 3 || res.Headers.ETag[0] != '"' || res.Headers.ETag[len(res.Headers.ETag)-1] != '"' {
			t.Errorf("ETag = %q、期待値 = 引用符で囲んだ強いエンティティタグ", res.Headers.ETag)
		}
		if other := get200(t, 1, nil); other.Headers.ETag == res.Headers.ETag {
			t.Errorf("内容の違うページのETagが同じ: %q", res.Headers.ETag)
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
			res := getPage(t, 2, &tt.ifNoneMatch)
			notModified, ok := res.(apigen.GetPage304Response)
			if !ok {
				t.Fatalf("レスポンスの型 = %T、期待値 = GetPage304Response", res)
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
		modifiedAt := pageHandlerBaseTime.Add(time.Hour)
		spaceID := middleware.APIPrincipalFromContext(ctx).Space.ID
		if _, err := tx.ExecContext(ctx,
			"UPDATE pages SET modified_at = $1 WHERE space_id = $2 AND number = $3",
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
		if res.Body.Body != "本文2" {
			t.Errorf("Body = %q、期待値 = 本文2", res.Body.Body)
		}
	})

	t.Run("存在しないページは未存在のエラーを返す", func(t *testing.T) {
		res, err := h.GetPage(ctx, apigen.GetPageRequestObject{SpaceIdentifier: "page-handler", PageNumber: 99})
		if res != nil {
			t.Errorf("レスポンス = %v、期待値 = nil", res)
		}
		if ae := model.AsAppError(err); ae == nil || ae.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("error = %v、期待値 = AppErrCodeResourceNotFound", err)
		}
	})
}
