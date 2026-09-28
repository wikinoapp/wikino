package topic_test

import (
	"testing"

	"github.com/wikinoapp/wikino/go/internal/apigen"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/testutil"
)

func TestHandler_GetTopic(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	h, ctx := setupTopicHandler(t, tx, 2)

	// サブテストは親のトランザクションを共有するため、並列にしない

	t.Run("トピックを返す", func(t *testing.T) {
		res, err := h.GetTopic(ctx, apigen.GetTopicRequestObject{SpaceIdentifier: "topic-handler", TopicNumber: 2})
		if err != nil {
			t.Fatalf("GetTopic() error = %v", err)
		}
		jsonRes, ok := res.(apigen.GetTopic200JSONResponse)
		if !ok {
			t.Fatalf("レスポンスの型 = %T、期待値 = GetTopic200JSONResponse", res)
		}
		if jsonRes.Body.Number != 2 || jsonRes.Body.Name != "トピック2" || jsonRes.Body.Visibility != apigen.TopicVisibility("public") {
			t.Errorf("トピック = %+v", jsonRes.Body)
		}
	})

	t.Run("存在しないトピックは未存在のエラーを返す", func(t *testing.T) {
		res, err := h.GetTopic(ctx, apigen.GetTopicRequestObject{SpaceIdentifier: "topic-handler", TopicNumber: 99})
		if res != nil {
			t.Errorf("レスポンス = %v、期待値 = nil", res)
		}
		if ae := model.AsAppError(err); ae == nil || ae.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("error = %v、期待値 = AppErrCodeResourceNotFound", err)
		}
	})
}
