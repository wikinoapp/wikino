package apigen_test

import (
	"context"
	"encoding/json"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/apigen"
)

// 生成コードに埋め込んだOpenAPI記述を、リクエスト検証などで読み込めることを確かめる
func TestGetSpec(t *testing.T) {
	t.Parallel()

	spec, err := apigen.GetSpec()
	if err != nil {
		t.Fatalf("GetSpec()のエラー = %v", err)
	}

	if !strings.HasPrefix(spec.OpenAPI, "3.1.") {
		t.Errorf("OpenAPI = %q、期待値 = 3.1.x", spec.OpenAPI)
	}

	if err := spec.Validate(context.Background()); err != nil {
		t.Errorf("Validate()のエラー = %v", err)
	}

	t.Run("トークンに付与できるスコープがOAuthのスキームに定義されている", func(t *testing.T) {
		t.Parallel()

		scheme, ok := spec.Components.SecuritySchemes["oauth2"]
		if !ok || scheme.Value == nil {
			t.Fatal("securitySchemes.oauth2が定義されていない")
		}
		flow := scheme.Value.Flows.AuthorizationCode
		if flow == nil {
			t.Fatal("securitySchemes.oauth2.flows.authorizationCodeが定義されていない")
		}

		var got []string
		for scope := range flow.Scopes {
			got = append(got, scope)
		}
		slices.Sort(got)
		want := []string{"attachment:read", "draft_page:delete", "draft_page:read", "draft_page:write", "page:read", "page:write", "topic:read"}
		if !slices.Equal(got, want) {
			t.Errorf("scopes = %v、期待値 = %v", got, want)
		}
	})

	t.Run("個人用アクセストークンのスキームがBearer認証として定義されている", func(t *testing.T) {
		t.Parallel()

		scheme, ok := spec.Components.SecuritySchemes["personalAccessToken"]
		if !ok || scheme.Value == nil {
			t.Fatal("securitySchemes.personalAccessTokenが定義されていない")
		}
		if scheme.Value.Type != "http" || scheme.Value.Scheme != "bearer" {
			t.Errorf("type = %q、scheme = %q、期待値 = http bearer", scheme.Value.Type, scheme.Value.Scheme)
		}
	})

	// リモートMCPサーバーなどGoの利用側は、読み込んだ記述の拡張項目から操作の分類を読む。
	// 分類の値の矛盾はvacuumの独自ルールが検査するため、ここでは読み込んだ記述に分類が残ることを確かめる。
	t.Run("すべてのoperationの操作の分類を読み込んだ記述から読める", func(t *testing.T) {
		t.Parallel()

		wantKeys := []string{"confirmation_required", "idempotent", "irreversible", "read_only"}
		count := 0
		for path, item := range spec.Paths.Map() {
			for method, op := range item.Operations() {
				count++
				raw, ok := op.Extensions["x-wikino-operation"]
				if !ok {
					t.Errorf("%s %sに x-wikino-operationが無い", method, path)
					continue
				}
				encoded, err := json.Marshal(raw)
				if err != nil {
					t.Fatalf("%s %sの x-wikino-operationのエンコードのエラー = %v", method, path, err)
				}
				var classification map[string]bool
				if err := json.Unmarshal(encoded, &classification); err != nil {
					t.Errorf("%s %sの x-wikino-operationを真偽値のマップとして読めない: %v", method, path, err)
					continue
				}
				gotKeys := slices.Sorted(maps.Keys(classification))
				if !slices.Equal(gotKeys, wantKeys) {
					t.Errorf("%s %sの x-wikino-operationの項目 = %v、期待値 = %v", method, path, gotKeys, wantKeys)
				}
			}
		}
		if count == 0 {
			t.Error("operationが1件も読み込まれていない")
		}
	})

	t.Run("OpenAPI記述を返すoperationはトークンを要求しない", func(t *testing.T) {
		t.Parallel()

		op := spec.Paths.Find("/openapi.yaml")
		if op == nil || op.Get == nil {
			t.Fatal("GET /openapi.yamlが定義されていない")
		}
		if op.Get.Security == nil || len(*op.Get.Security) != 0 {
			t.Errorf("security = %v、期待値 = []", op.Get.Security)
		}
	})
}
