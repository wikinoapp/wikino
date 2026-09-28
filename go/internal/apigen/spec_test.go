package apigen_test

import (
	"context"
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
		t.Fatalf("GetSpec() error = %v", err)
	}

	if !strings.HasPrefix(spec.OpenAPI, "3.1.") {
		t.Errorf("OpenAPI = %q, want 3.1.x", spec.OpenAPI)
	}

	if err := spec.Validate(context.Background()); err != nil {
		t.Errorf("Validate() error = %v", err)
	}

	t.Run("トークンに付与できるスコープがOAuthのスキームに定義されている", func(t *testing.T) {
		t.Parallel()

		scheme, ok := spec.Components.SecuritySchemes["oauth2"]
		if !ok || scheme.Value == nil {
			t.Fatal("securitySchemes.oauth2 is not defined")
		}
		flow := scheme.Value.Flows.AuthorizationCode
		if flow == nil {
			t.Fatal("securitySchemes.oauth2.flows.authorizationCode is not defined")
		}

		var got []string
		for scope := range flow.Scopes {
			got = append(got, scope)
		}
		slices.Sort(got)
		want := []string{"page:read", "page:write", "topic:read"}
		if !slices.Equal(got, want) {
			t.Errorf("scopes = %v, want %v", got, want)
		}
	})

	t.Run("個人アクセストークンのスキームがBearer認証として定義されている", func(t *testing.T) {
		t.Parallel()

		scheme, ok := spec.Components.SecuritySchemes["personalAccessToken"]
		if !ok || scheme.Value == nil {
			t.Fatal("securitySchemes.personalAccessToken is not defined")
		}
		if scheme.Value.Type != "http" || scheme.Value.Scheme != "bearer" {
			t.Errorf("type = %q, scheme = %q, want http bearer", scheme.Value.Type, scheme.Value.Scheme)
		}
	})

	t.Run("OpenAPI記述を返すoperationはトークンを要求しない", func(t *testing.T) {
		t.Parallel()

		op := spec.Paths.Find("/openapi.yaml")
		if op == nil || op.Get == nil {
			t.Fatal("GET /openapi.yaml is not defined")
		}
		if op.Get.Security == nil || len(*op.Get.Security) != 0 {
			t.Errorf("security = %v, want []", op.Get.Security)
		}
	})
}
