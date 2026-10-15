package main

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	t.Parallel()

	spec := `
paths:
  /spaces/{space_identifier}/pages/{page_number}:
    parameters: []
    patch:
      operationId: updatePage
      x-wikino-operation:
        read_only: false
        irreversible: false
        idempotent: true
        confirmation_required: false
    get:
      operationId: getPage
      x-wikino-operation:
        read_only: true
        irreversible: false
        idempotent: true
        confirmation_required: false
  /spaces/{space_identifier}/pages/{page_number}/draft_page:
    delete:
      operationId: deleteDraftPage
      x-wikino-operation:
        read_only: false
        irreversible: true
        idempotent: true
        confirmation_required: true
`

	ops, err := parse([]byte(spec))
	if err != nil {
		t.Fatalf("解析に失敗しました: %v", err)
	}

	want := []operation{
		{id: "getPage", method: "GET", path: "/spaces/{space_identifier}/pages/{page_number}", readOnly: true, idempotent: true},
		{id: "updatePage", method: "PATCH", path: "/spaces/{space_identifier}/pages/{page_number}", idempotent: true},
		{id: "deleteDraftPage", method: "DELETE", path: "/spaces/{space_identifier}/pages/{page_number}/draft_page", irreversible: true, idempotent: true, confirmationRequired: true},
	}
	if len(ops) != len(want) {
		t.Fatalf("operationの数が違います: 実際 %d, 期待 %d", len(ops), len(want))
	}
	for i := range want {
		if ops[i] != want[i] {
			t.Errorf("%d番目のoperationが違います:\n実際  %+v\n期待 %+v", i, ops[i], want[i])
		}
	}
}

func TestParse_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		spec string
		want string
	}{
		{
			name: "拡張項目が無い",
			spec: `
paths:
  /a:
    get:
      operationId: getA
`,
			want: "x-wikino-operationがありません",
		},
		{
			name: "拡張項目の値が欠けている",
			spec: `
paths:
  /a:
    get:
      operationId: getA
      x-wikino-operation:
        read_only: true
        irreversible: false
        idempotent: true
`,
			want: "すべて書いてください",
		},
		{
			name: "operationIdが無い",
			spec: `
paths:
  /a:
    get:
      x-wikino-operation: {read_only: true, irreversible: false, idempotent: true, confirmation_required: false}
`,
			want: "operationIdがありません",
		},
		{
			name: "operationが無い",
			spec: `
paths: {}
`,
			want: "operationが1つもありません",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := parse([]byte(tt.spec))
			if err == nil {
				t.Fatal("エラーになるはずが成功しました")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("エラーの内容が違います: 実際 %q, 期待 %qを含む", err.Error(), tt.want)
			}
		})
	}
}
