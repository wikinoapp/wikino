package mcpserver

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// loadTestSpecは、公開APIのOpenAPI記述を読み込む
func loadTestSpec(t *testing.T) *openapi3.T {
	t.Helper()

	spec, err := openapi3.NewLoader().LoadFromData(testOpenAPIDescription(t))
	if err != nil {
		t.Fatalf("OpenAPI記述の読み込みに失敗: %v", err)
	}
	return spec
}

// findToolは、名前nameのツールを返す
func findTool(t *testing.T, tools []*operationTool, name string) *operationTool {
	t.Helper()

	for _, tool := range tools {
		if tool.tool.Name == name {
			return tool
		}
	}
	t.Fatalf("ツール%sが無い", name)
	return nil
}

// schemaPropertiesは、JSON Schemaのプロパティの名前を順に並べて返す
func schemaProperties(t *testing.T, schema any) []string {
	t.Helper()

	properties, ok := schema.(map[string]any)["properties"].(map[string]any)
	if !ok {
		t.Fatalf("スキーマにpropertiesが無い: %v", schema)
	}
	names := make([]string, 0, len(properties))
	for name := range properties {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func TestNewOperationTools_スペース配下のoperationをすべてツールにする(t *testing.T) {
	t.Parallel()

	spec := loadTestSpec(t)
	tools, err := newOperationTools(spec)
	if err != nil {
		t.Fatalf("newOperationTools()のエラー = %v", err)
	}

	var want []string
	for path, item := range spec.Paths.Map() {
		if !strings.HasPrefix(path, spaceOperationPath) {
			continue
		}
		for _, op := range item.Operations() {
			want = append(want, op.OperationID)
		}
	}
	slices.Sort(want)
	got := make([]string, len(tools))
	for i, tool := range tools {
		got[i] = tool.tool.Name
	}
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Errorf("ツールの名前 = %v、期待値 = %v (スペース配下のoperationId)", got, want)
	}
	if slices.Contains(got, "getOpenAPIDescription") {
		t.Error("スペース配下でないoperationがツールになっている")
	}

	for _, tool := range tools {
		data, err := json.Marshal(tool.tool)
		if err != nil {
			t.Fatalf("%sの書き出しに失敗: %v", tool.tool.Name, err)
		}
		// ツールのスキーマは単独で解釈されるため、OpenAPI記述の中への参照を残さない
		if strings.Contains(string(data), `"$ref"`) {
			t.Errorf("%sのスキーマに参照が残っている", tool.tool.Name)
		}
		// スペースはトークンから決めるため、入力に含めない
		if slices.Contains(schemaProperties(t, tool.tool.InputSchema), spaceIdentifierParam) {
			t.Errorf("%sの入力にspace_identifierがある", tool.tool.Name)
		}
		if tool.tool.Title == "" || tool.tool.Description == "" {
			t.Errorf("%sのタイトルか説明が空", tool.tool.Name)
		}
	}
}

func TestNewOperationTools_OpenAPI記述からツールの項目を決める(t *testing.T) {
	t.Parallel()

	spec := loadTestSpec(t)
	tools, err := newOperationTools(spec)
	if err != nil {
		t.Fatalf("newOperationTools()のエラー = %v", err)
	}

	t.Run("読み取りのoperation", func(t *testing.T) {
		t.Parallel()

		tool := findTool(t, tools, "listPages")
		op := spec.Paths.Value("/spaces/{space_identifier}/pages").Get
		if tool.tool.Title != op.Summary || tool.tool.Description != op.Description {
			t.Errorf("タイトル・説明 = (%q, %q)、期待値 = operationのsummary・description", tool.tool.Title, tool.tool.Description)
		}
		if want := []string{"cursor", "include", "limit", "modified_since", "q", "status", "title", "topic_number"}; !slices.Equal(schemaProperties(t, tool.tool.InputSchema), want) {
			t.Errorf("入力のプロパティ = %v、期待値 = %v", schemaProperties(t, tool.tool.InputSchema), want)
		}
		a := tool.tool.Annotations
		if !a.ReadOnlyHint || !a.IdempotentHint || *a.DestructiveHint || *a.OpenWorldHint {
			t.Errorf("注釈 = %+v、期待値 = 読み取り専用・冪等・破壊的でない・閉じた世界", a)
		}
		if !slices.Equal(tool.scopes, []model.Scope{model.ScopePageRead}) {
			t.Errorf("スコープ = %v、期待値 = [page:read]", tool.scopes)
		}
		if tool.method != "GET" || tool.path != "/spaces/{space_identifier}/pages" || tool.bodyMediaType != "" {
			t.Errorf("要求 = %s %s (本文 %q)、期待値 = GET /spaces/{space_identifier}/pages (本文なし)", tool.method, tool.path, tool.bodyMediaType)
		}
		if tool.tool.OutputSchema == nil || !slices.Contains(schemaProperties(t, tool.tool.OutputSchema), "next_cursor") {
			t.Errorf("出力のスキーマ = %v、期待値 = ページの一覧のスキーマ", tool.tool.OutputSchema)
		}
	})

	t.Run("パス・ヘッダー・本文をまとめるoperation", func(t *testing.T) {
		t.Parallel()

		tool := findTool(t, tools, "updatePage")
		if want := []string{"body", "if_match", "page_number", "title"}; !slices.Equal(schemaProperties(t, tool.tool.InputSchema), want) {
			t.Errorf("入力のプロパティ = %v、期待値 = %v", schemaProperties(t, tool.tool.InputSchema), want)
		}
		if required := tool.tool.InputSchema.(map[string]any)["required"]; !slices.Equal(required.([]string), []string{"page_number"}) {
			t.Errorf("必須のプロパティ = %v、期待値 = [page_number]", required)
		}
		wantParams := []toolParam{
			{property: "page_number", in: "path", name: "page_number"},
			{property: "if_match", in: "header", name: "If-Match"},
		}
		if !slices.Equal(tool.params, wantParams) {
			t.Errorf("パラメーター = %+v、期待値 = %+v", tool.params, wantParams)
		}
		if !slices.Equal(tool.bodyProperties, []string{"body", "title"}) || tool.bodyMediaType != "application/merge-patch+json" {
			t.Errorf("本文 = %v (%s)、期待値 = [body title] (application/merge-patch+json)", tool.bodyProperties, tool.bodyMediaType)
		}
		if a := tool.tool.Annotations; a.ReadOnlyHint || !a.IdempotentHint || *a.DestructiveHint {
			t.Errorf("注釈 = %+v、期待値 = 読み取り専用でない・冪等・破壊的でない", a)
		}
	})

	t.Run("本文の必須の項目を入力でも必須にする", func(t *testing.T) {
		t.Parallel()

		tool := findTool(t, tools, "createPage")
		required := tool.tool.InputSchema.(map[string]any)["required"].([]string)
		slices.Sort(required)
		if want := []string{"body", "title", "topic_number"}; !slices.Equal(required, want) {
			t.Errorf("必須のプロパティ = %v、期待値 = %v", required, want)
		}
	})

	t.Run("取り消せないoperationは破壊的とし、本文の無い応答には出力のスキーマを付けない", func(t *testing.T) {
		t.Parallel()

		tool := findTool(t, tools, "deleteDraftPage")
		if !*tool.tool.Annotations.DestructiveHint {
			t.Errorf("DestructiveHint = false、期待値 = true")
		}
		if tool.tool.OutputSchema != nil {
			t.Errorf("出力のスキーマ = %v、期待値 = nil (204の応答)", tool.tool.OutputSchema)
		}
	})

	t.Run("複数のスコープが要るoperation", func(t *testing.T) {
		t.Parallel()

		tool := findTool(t, tools, "createDraftPagePublication")
		if !slices.Equal(tool.scopes, []model.Scope{model.ScopeDraftPageWrite, model.ScopePageWrite}) {
			t.Errorf("スコープ = %v、期待値 = [draft_page:write page:write]", tool.scopes)
		}
	})
}

func TestOperationTool_isAvailableTo(t *testing.T) {
	t.Parallel()

	tool := &operationTool{scopes: []model.Scope{model.ScopeDraftPageWrite, model.ScopePageWrite}}
	tests := []struct {
		name   string
		scopes []model.Scope
		want   bool
	}{
		{name: "要るスコープをすべて持つ", scopes: []model.Scope{model.ScopeDraftPageWrite, model.ScopePageWrite, model.ScopePageRead}, want: true},
		{name: "要るスコープの一部しか持たない", scopes: []model.Scope{model.ScopePageWrite}, want: false},
		{name: "スコープが無い", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tool.isAvailableTo(tt.scopes); got != tt.want {
				t.Errorf("isAvailableTo(%v) = %v、期待値 = %v", tt.scopes, got, tt.want)
			}
		})
	}

	if !(&operationTool{}).isAvailableTo(nil) {
		t.Error("スコープの要らないツールを、スコープの無いトークンで呼べない")
	}
}

// testSpecTemplateは、変換できないoperationを検出するテストの記述。OPERATIONに1つのoperationを入れる
const testSpecTemplate = `
openapi: 3.1.1
info:
  title: test
  version: 1.0.0
paths:
  /spaces/{space_identifier}/items/{item_number}:
    parameters:
      - name: space_identifier
        in: path
        required: true
        schema:
          type: string
      - name: item_number
        in: path
        required: true
        schema:
          type: integer
OPERATION
components:
  schemas:
    ItemPatch:
      type: object
      properties:
        item_number:
          type: integer
    Item:
      type: object
      properties:
        item_number:
          type: integer
        child:
          $ref: "#/components/schemas/Node"
    Node:
      type: object
      properties:
        next:
          $ref: "#/components/schemas/Node"
`

// testOperationExtensionは、変換できるoperationの操作の分類
const testOperationExtension = `      x-wikino-operation:
        read_only: false
        irreversible: false
        idempotent: true
        confirmation_required: false
`

// loadSpecWithOperationは、testSpecTemplateにoperationを入れた記述を読み込む
func loadSpecWithOperation(t *testing.T, operation string) *openapi3.T {
	t.Helper()

	spec, err := openapi3.NewLoader().LoadFromData([]byte(strings.Replace(testSpecTemplate, "OPERATION", operation, 1)))
	if err != nil {
		t.Fatalf("記述の読み込みに失敗: %v", err)
	}
	return spec
}

func TestNewOperationTools_ツールに変換できないoperationを検出する(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		operation string
		// wantはエラーの文面に含まれることを期待する理由
		want string
	}{
		{
			name: "パスのパラメーターと本文のプロパティの名前がぶつかる",
			want: "重複",
			operation: `    patch:
      operationId: updateItem
` + testOperationExtension + `      requestBody:
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/ItemPatch"
      responses:
        "204":
          description: ok
`,
		},
		{
			name: "JSONでない本文",
			want: "メディアタイプ",
			operation: `    put:
      operationId: putItem
` + testOperationExtension + `      requestBody:
        content:
          text/plain:
            schema:
              type: string
      responses:
        "204":
          description: ok
`,
		},
		{
			name: "Cookieのパラメーター",
			want: "cookie",
			operation: `    get:
      operationId: getItem
` + testOperationExtension + `      parameters:
        - name: session
          in: cookie
          schema:
            type: string
      responses:
        "204":
          description: ok
`,
		},
		{
			name: "操作の分類が無い",
			want: "x-wikino-operation",
			operation: `    get:
      operationId: getItem
      responses:
        "204":
          description: ok
`,
		},
		{
			name: "operationIdが無い",
			want: "operationId",
			operation: `    get:
` + testOperationExtension + `      responses:
        "204":
          description: ok
`,
		},
		{
			name: "入力のスキーマが循環する",
			want: "表せない",
			operation: `    post:
      operationId: createNode
` + testOperationExtension + `      requestBody:
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/Node"
      responses:
        "204":
          description: ok
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tools, err := newOperationTools(loadSpecWithOperation(t, tt.operation))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("newOperationTools() = %v、エラー = %v、期待値 = %qを含むエラー", tools, err, tt.want)
			}
		})
	}
}

func TestNewOperationTools_循環する出力のスキーマは付けない(t *testing.T) {
	t.Parallel()

	tools, err := newOperationTools(loadSpecWithOperation(t, `    get:
      operationId: getItem
`+testOperationExtension+`      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Item"
`))
	if err != nil {
		t.Fatalf("newOperationTools()のエラー = %v", err)
	}
	if len(tools) != 1 || tools[0].tool.OutputSchema != nil {
		t.Errorf("ツール = %+v、出力のスキーマの無いツールを1つ期待", tools)
	}
}
