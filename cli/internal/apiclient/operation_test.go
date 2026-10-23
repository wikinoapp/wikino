package apiclient

import (
	"slices"
	"testing"

	"github.com/wikinoapp/wikino/cli/internal/apigen"
)

func TestMatchOperation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		method string
		path   string
		wantID string
	}{
		{name: "パスのテンプレートに当たる", method: "GET", path: "/spaces/my-space/pages/12", wantID: "getPage"},
		{name: "メソッドの大文字小文字を区別しない", method: "patch", path: "/spaces/my-space/pages/12", wantID: "updatePage"},
		{name: "下の階層のパスと区別する", method: "DELETE", path: "/spaces/my-space/pages/12/draft_page", wantID: "deleteDraftPage"},
		{name: "セグメントの数が違えば当たらない", method: "GET", path: "/spaces/my-space/pages/12/extra"},
		{name: "空のセグメントは値に当たらない", method: "GET", path: "/spaces//pages/12"},
		{name: "メソッドが違えば当たらない", method: "PUT", path: "/spaces/my-space/pages/12"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			op, ok := MatchOperation(tt.method, tt.path)
			if tt.wantID == "" {
				if ok {
					t.Errorf("当たらないはずが %s に当たりました", op.ID)
				}
				return
			}
			if !ok || op.ID != tt.wantID {
				t.Errorf("当たるoperationが違います: 実際 %q (ok=%t), 期待 %q", op.ID, ok, tt.wantID)
			}
		})
	}
}

func TestLookupOperation(t *testing.T) {
	t.Parallel()

	op, ok := LookupOperation("createWikilinkPreview")
	if !ok || !op.ReadOnly || op.Method != "POST" {
		t.Errorf("Wikiリンクのプレビューは読み取り専用のPOSTのはずです: %+v (ok=%t)", op, ok)
	}

	if _, ok := LookupOperation("unknownOperation"); ok {
		t.Error("無いoperationが見つかりました")
	}
}

// 確認プロトコルの対象 (取り消せない操作と公開) が、OpenAPI記述の分類から決まることを確かめる。
func TestOperations_ConfirmationRequired(t *testing.T) {
	t.Parallel()

	var got []string
	for _, op := range apigen.Operations {
		if op.ConfirmationRequired {
			got = append(got, op.ID)
		}
		if op.ConfirmationRequired && op.ReadOnly {
			t.Errorf("%s: 読み取り専用のoperationに確認を求めています", op.ID)
		}
	}
	slices.Sort(got)

	want := []string{"createDraftPagePublication", "deleteDraftPage"}
	if !slices.Equal(got, want) {
		t.Errorf("確認が要るoperationが違います: 実際 %v, 期待 %v", got, want)
	}
}
