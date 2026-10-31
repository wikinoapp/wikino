package mcpserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// spaceOperationPathは、ツールにするoperationのパスの接頭辞。トークンはスペースに属し、ツールの
// 呼び出しではスペースをトークンから決めるため、パスの `space_identifier` はツールの入力に含めない
const spaceOperationPath = "/spaces/{space_identifier}"

// spaceIdentifierParamは、ツールの入力に含めないスペースのパスの変数
const spaceIdentifierParam = "space_identifier"

// operationExtensionは、operationの操作の分類を持つ拡張項目
const operationExtension = "x-wikino-operation"

// oauth2SecuritySchemeは、ツールに要るスコープを読むセキュリティの方式。MCPサーバーはOAuthの
// アクセストークンだけを受け付ける
const oauth2SecurityScheme = "oauth2"

// jsonBodyMediaTypesは、ツールの入力から組み立てられるリクエスト本文のメディアタイプ
var jsonBodyMediaTypes = []string{"application/json", "application/merge-patch+json"}

// errUnrepresentableSchemaは、スキーマを参照を展開した1つのJSON Schemaで表せないことを表す (循環する参照など)
var errUnrepresentableSchema = errors.New("参照を展開したJSON Schemaで表せないスキーマ")

// toolParamは、ツールの入力のプロパティのうち、リクエストのパス・クエリ・ヘッダーに置くもの
type toolParam struct {
	// propertyはツールの入力のプロパティの名前
	property string
	// inはパラメーターの置き場所 (`path`・`query`・`header`)
	in string
	// nameはOpenAPI記述でのパラメーターの名前 (ヘッダーでは `If-Match` など)
	name string
}

// operationToolは、OpenAPI記述の1つのoperationから作ったツールと、呼び出しで公開APIへの要求を
// 組み立てるための情報
type operationTool struct {
	tool *mcp.Tool

	// methodとpathは、operationのHTTPのメソッドとOpenAPI記述のパス (`servers` の `/api/v1` を除く)
	method string
	path   string

	// scopesは、ツールを呼ぶのに要るトークンのスコープ (operationの `security` のOAuthのスコープ)
	scopes []model.Scope

	params []toolParam

	// bodyPropertiesは、ツールの入力のプロパティのうちリクエスト本文に置くもの。本文の無いoperationでは空
	bodyProperties []string
	// bodyMediaTypeは、リクエスト本文のメディアタイプ。本文の無いoperationでは空
	bodyMediaType string
}

// isAvailableToは、トークンのスコープscopes (含意を展開したもの) でツールを呼べるかを返す
func (t *operationTool) isAvailableTo(scopes []model.Scope) bool {
	for _, scope := range t.scopes {
		if !model.HasScope(scopes, scope) {
			return false
		}
	}
	return true
}

// newOperationToolsは、OpenAPI記述specのスペース配下のoperationを、それぞれ1つのツールにする。
// ツールに変換できないoperationがあれば、その理由をエラーで返す
func newOperationTools(spec *openapi3.T) ([]*operationTool, error) {
	inliner := &schemaInliner{spec: spec}
	paths := spec.Paths.Map()
	pathNames := make([]string, 0, len(paths))
	for path := range paths {
		if path == spaceOperationPath || strings.HasPrefix(path, spaceOperationPath+"/") {
			pathNames = append(pathNames, path)
		}
	}
	slices.Sort(pathNames)

	var tools []*operationTool
	for _, path := range pathNames {
		item := paths[path]
		operations := item.Operations()
		methods := make([]string, 0, len(operations))
		for method := range operations {
			methods = append(methods, method)
		}
		slices.Sort(methods)
		for _, method := range methods {
			tool, err := newOperationTool(inliner, spec, path, method, item, operations[method])
			if err != nil {
				return nil, fmt.Errorf("%s %sをツールにできない: %w", method, path, err)
			}
			tools = append(tools, tool)
		}
	}
	return tools, nil
}

// newOperationToolは、パスpathのメソッドmethodのoperationからツールを作る
func newOperationTool(inliner *schemaInliner, spec *openapi3.T, path, method string, item *openapi3.PathItem, op *openapi3.Operation) (*operationTool, error) {
	if op.OperationID == "" {
		return nil, errors.New("operationIdが無い")
	}
	annotations, err := toolAnnotations(op)
	if err != nil {
		return nil, err
	}

	t := &operationTool{method: method, path: path, scopes: operationScopes(spec, op)}
	input := newInputSchema()

	parameters := slices.Concat(item.Parameters, op.Parameters)
	for _, ref := range parameters {
		param := ref.Value
		if param.In == openapi3.ParameterInPath && param.Name == spaceIdentifierParam {
			continue
		}
		var property string
		switch param.In {
		case openapi3.ParameterInPath, openapi3.ParameterInQuery:
			property = param.Name
		case openapi3.ParameterInHeader:
			property = strings.ReplaceAll(strings.ToLower(param.Name), "-", "_")
		default:
			return nil, fmt.Errorf("%sのパラメーター%sはツールの入力にできない", param.In, param.Name)
		}
		schema, err := inliner.inline(param.Schema)
		if err != nil {
			return nil, fmt.Errorf("パラメーター%sのスキーマ: %w", param.Name, err)
		}
		if _, ok := schema["description"]; !ok && param.Description != "" {
			schema["description"] = param.Description
		}
		if err := input.add(property, schema, param.Required); err != nil {
			return nil, err
		}
		t.params = append(t.params, toolParam{property: property, in: param.In, name: param.Name})
	}

	if op.RequestBody != nil {
		if err := t.addBody(inliner, input, op.RequestBody.Value); err != nil {
			return nil, err
		}
	}

	t.tool = &mcp.Tool{
		Name:        op.OperationID,
		Title:       op.Summary,
		Description: op.Description,
		InputSchema: input.schema(),
		Annotations: annotations,
	}
	if output, ok := outputSchema(inliner, op); ok {
		t.tool.OutputSchema = output
	}
	return t, nil
}

// addBodyは、リクエスト本文bodyのオブジェクトのプロパティをツールの入力に加える
func (t *operationTool) addBody(inliner *schemaInliner, input *inputSchema, body *openapi3.RequestBody) error {
	if len(body.Content) != 1 {
		return fmt.Errorf("リクエスト本文のメディアタイプが%d個ある", len(body.Content))
	}
	for mediaType, content := range body.Content {
		if !slices.Contains(jsonBodyMediaTypes, mediaType) {
			return fmt.Errorf("リクエスト本文のメディアタイプ%sはツールの入力にできない", mediaType)
		}
		schema, err := inliner.inline(content.Schema)
		if err != nil {
			return fmt.Errorf("リクエスト本文のスキーマ: %w", err)
		}
		if schema["type"] != "object" {
			return errors.New("リクエスト本文のスキーマがオブジェクトでない")
		}
		properties, _ := schema["properties"].(map[string]any)
		required, _ := schema["required"].([]any)
		names := make([]string, 0, len(properties))
		for name := range properties {
			names = append(names, name)
		}
		slices.Sort(names)
		for _, name := range names {
			property, ok := properties[name].(map[string]any)
			if !ok {
				return fmt.Errorf("リクエスト本文のプロパティ%sのスキーマがオブジェクトでない", name)
			}
			if err := input.add(name, property, body.Required && slices.Contains(required, any(name))); err != nil {
				return err
			}
			t.bodyProperties = append(t.bodyProperties, name)
		}
		t.bodyMediaType = mediaType
	}
	return nil
}

// toolAnnotationsは、operationの操作の分類 (`x-wikino-operation`) からツールの注釈を作る
func toolAnnotations(op *openapi3.Operation) (*mcp.ToolAnnotations, error) {
	raw, ok := op.Extensions[operationExtension].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%sが無い", operationExtension)
	}
	flags := map[string]bool{}
	for _, name := range []string{"read_only", "irreversible", "idempotent", "confirmation_required"} {
		value, ok := raw[name].(bool)
		if !ok {
			return nil, fmt.Errorf("%sの%sが真偽値でない", operationExtension, name)
		}
		flags[name] = value
	}
	destructive := flags["irreversible"] || flags["confirmation_required"]
	// Wikinoの中だけで閉じ、外部のシステムに作用しないため、どのツールも閉じた世界とする
	openWorld := false
	return &mcp.ToolAnnotations{
		ReadOnlyHint:    flags["read_only"],
		IdempotentHint:  flags["idempotent"],
		DestructiveHint: &destructive,
		OpenWorldHint:   &openWorld,
	}, nil
}

// operationScopesは、operationの `security` (無ければOpenAPI記述全体の `security`) のOAuthのスコープを返す
func operationScopes(spec *openapi3.T, op *openapi3.Operation) []model.Scope {
	security := spec.Security
	if op.Security != nil {
		security = *op.Security
	}
	for _, requirement := range security {
		if scopes, ok := requirement[oauth2SecurityScheme]; ok {
			return model.StringsToScopes(scopes)
		}
	}
	return nil
}

// outputSchemaは、operationの成功の応答 (2xx) のJSONのスキーマを、参照を展開して返す。
// 成功の応答に本文が無い場合と、オブジェクトでないか参照を展開して表せないスキーマの場合は、付けない (false)
func outputSchema(inliner *schemaInliner, op *openapi3.Operation) (map[string]any, bool) {
	for _, status := range []int{http.StatusOK, http.StatusCreated} {
		response := op.Responses.Status(status)
		if response == nil || response.Value == nil {
			continue
		}
		content := response.Value.Content.Get("application/json")
		if content == nil || content.Schema == nil {
			return nil, false
		}
		schema, err := inliner.inline(content.Schema)
		if err != nil || schema["type"] != "object" {
			return nil, false
		}
		return schema, true
	}
	return nil, false
}

// inputSchemaは、ツールの入力のJSON Schema (オブジェクト) を組み立てる
type inputSchema struct {
	properties map[string]any
	required   []string
}

func newInputSchema() *inputSchema {
	return &inputSchema{properties: map[string]any{}}
}

// addは、プロパティnameを加える。パス・クエリ・ヘッダー・本文のどこかで同じ名前が既にあればエラーにする
func (s *inputSchema) add(name string, schema map[string]any, required bool) error {
	if _, ok := s.properties[name]; ok {
		return fmt.Errorf("ツールの入力のプロパティ%sが重複している", name)
	}
	s.properties[name] = schema
	if required {
		s.required = append(s.required, name)
	}
	return nil
}

// schemaは、ツールの入力のJSON Schemaを返す。定義していないプロパティは受け付けない
func (s *inputSchema) schema() map[string]any {
	schema := map[string]any{
		"type":                 "object",
		"properties":           s.properties,
		"additionalProperties": false,
	}
	if len(s.required) > 0 {
		schema["required"] = s.required
	}
	return schema
}

// schemaInlinerは、OpenAPI記述のスキーマを、`components` への参照 (`$ref`) を展開した1つのJSON Schemaにする。
// ツールの入力と出力のスキーマは単独で解釈されるため、OpenAPI記述の中への参照を残せない
type schemaInliner struct {
	spec *openapi3.T
}

// componentSchemaPrefixは、展開する参照の接頭辞
const componentSchemaPrefix = "#/components/schemas/"

// inlineは、スキーマrefを参照を展開したJSONの値にする
func (i *schemaInliner) inline(ref *openapi3.SchemaRef) (map[string]any, error) {
	if ref == nil || ref.Value == nil {
		return nil, errUnrepresentableSchema
	}
	node, err := toJSONValue(ref.Value)
	if err != nil {
		return nil, err
	}
	resolved, err := i.resolve(node, nil)
	if err != nil {
		return nil, err
	}
	schema, ok := resolved.(map[string]any)
	if !ok {
		return nil, errUnrepresentableSchema
	}
	return schema, nil
}

// resolveは、JSONの値nodeの中の参照を展開する。stackは展開中のスキーマの名前で、循環を検出するために使う
func (i *schemaInliner) resolve(node any, stack []string) (any, error) {
	switch v := node.(type) {
	case map[string]any:
		if ref, ok := v["$ref"].(string); ok {
			return i.resolveRef(ref, v, stack)
		}
		resolved := make(map[string]any, len(v))
		for key, value := range v {
			r, err := i.resolve(value, stack)
			if err != nil {
				return nil, err
			}
			resolved[key] = r
		}
		return resolved, nil
	case []any:
		resolved := make([]any, len(v))
		for idx, value := range v {
			r, err := i.resolve(value, stack)
			if err != nil {
				return nil, err
			}
			resolved[idx] = r
		}
		return resolved, nil
	default:
		return v, nil
	}
}

// resolveRefは、参照refを指すスキーマに置き換える。参照と並べて書いたキーワード (`description` など) は、
// 参照先より優先して残す
func (i *schemaInliner) resolveRef(ref string, siblings map[string]any, stack []string) (any, error) {
	name, ok := strings.CutPrefix(ref, componentSchemaPrefix)
	if !ok || slices.Contains(stack, name) {
		return nil, errUnrepresentableSchema
	}
	target := i.spec.Components.Schemas[name]
	if target == nil || target.Value == nil {
		return nil, errUnrepresentableSchema
	}
	node, err := toJSONValue(target.Value)
	if err != nil {
		return nil, err
	}
	resolved, err := i.resolve(node, append(slices.Clone(stack), name))
	if err != nil {
		return nil, err
	}
	schema, ok := resolved.(map[string]any)
	if !ok {
		return nil, errUnrepresentableSchema
	}
	for key, value := range siblings {
		if key == "$ref" {
			continue
		}
		r, err := i.resolve(value, stack)
		if err != nil {
			return nil, err
		}
		schema[key] = r
	}
	return schema, nil
}

// toJSONValueは、スキーマをJSONに書き出して読み直した値にする。入れ子のスキーマへの参照は `$ref` のまま残る
func toJSONValue(schema *openapi3.Schema) (any, error) {
	data, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("スキーマの書き出しに失敗: %w", err)
	}
	var node any
	if err := json.Unmarshal(data, &node); err != nil {
		return nil, fmt.Errorf("スキーマの読み直しに失敗: %w", err)
	}
	return node, nil
}
