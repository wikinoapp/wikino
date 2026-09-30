package model_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
)

func TestSortAPITokenScopes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		scopes []model.Scope
		want   []model.Scope
	}{
		{
			name:   "文字列の順に保存されたスコープをAPITokenScopesの順に並べる",
			scopes: []model.Scope{model.ScopePageRead, model.ScopePageWrite, model.ScopeTopicRead},
			want:   []model.Scope{model.ScopeTopicRead, model.ScopePageRead, model.ScopePageWrite},
		},
		{
			name:   "重複を除く",
			scopes: []model.Scope{model.ScopePageWrite, model.ScopePageWrite, model.ScopeTopicRead},
			want:   []model.Scope{model.ScopeTopicRead, model.ScopePageWrite},
		},
		{
			name:   "APITokenScopesに無いスコープは元の順のまま末尾に置く",
			scopes: []model.Scope{"unknown:read", model.ScopePageRead, model.ScopeSpaceAdmin},
			want:   []model.Scope{model.ScopePageRead, "unknown:read", model.ScopeSpaceAdmin},
		},
		{
			name:   "空",
			scopes: []model.Scope{},
			want:   []model.Scope{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := model.SortAPITokenScopes(tt.scopes)
			if !slices.Equal(got, tt.want) {
				t.Errorf("SortAPITokenScopes() = %v、期待値 = %v", got, tt.want)
			}
		})
	}
}

// scopeConstantsは、scope.goでScope型として宣言した定数の値を宣言順に返す。
// 定数を足したときに定義の表への追加漏れを検出できるよう、手で列挙せずソースから集める
func scopeConstants(t *testing.T) []model.Scope {
	t.Helper()

	file, err := parser.ParseFile(token.NewFileSet(), "scope.go", nil, 0)
	if err != nil {
		t.Fatalf("scope.goの解析に失敗: %v", err)
	}

	var scopes []model.Scope
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			typ, ok := vs.Type.(*ast.Ident)
			if !ok || typ.Name != "Scope" {
				continue
			}
			for _, v := range vs.Values {
				lit, ok := v.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					t.Fatalf("Scope型の定数の値が文字列リテラルではない: %T", v)
				}
				value, err := strconv.Unquote(lit.Value)
				if err != nil {
					t.Fatalf("文字列リテラルの解釈に失敗: %v", err)
				}
				scopes = append(scopes, model.Scope(value))
			}
		}
	}
	return scopes
}

func TestScopeDefinitions(t *testing.T) {
	t.Parallel()

	t.Run("space:adminを除くスコープの定数がすべて表に1回ずつ載っている", func(t *testing.T) {
		t.Parallel()

		constants := scopeConstants(t)
		if len(constants) == 0 {
			t.Fatal("Scope型の定数が見つからない")
		}

		counts := make(map[model.Scope]int, len(model.ScopeDefinitions))
		for _, d := range model.ScopeDefinitions {
			counts[d.Scope]++
		}

		for _, c := range constants {
			if c == model.ScopeSpaceAdmin {
				if counts[c] != 0 {
					t.Errorf("%sが表に載っている", c)
				}
				continue
			}
			if counts[c] != 1 {
				t.Errorf("%sが表に載っている回数 = %d、期待値 = 1", c, counts[c])
			}
		}
		for s := range counts {
			if !slices.Contains(constants, s) {
				t.Errorf("%sは表にあるがscope.goの定数に無い", s)
			}
		}
	})

	t.Run("付けられる最も下の単位が定義済みの値である", func(t *testing.T) {
		t.Parallel()

		for _, d := range model.ScopeDefinitions {
			if d.LowestLevel != model.ScopeLevelSpace && d.LowestLevel != model.ScopeLevelTopic {
				t.Errorf("%sの付けられる最も下の単位 = %d、期待値はScopeLevelSpaceかScopeLevelTopic", d.Scope, d.LowestLevel)
			}
		}
	})

	t.Run("含意の先が表にあり同じリソースである", func(t *testing.T) {
		t.Parallel()

		for _, d := range model.ScopeDefinitions {
			resource, _, _ := strings.Cut(d.Scope.String(), ":")
			for _, implied := range d.Implies {
				if !slices.ContainsFunc(model.ScopeDefinitions, func(other model.ScopeDefinition) bool { return other.Scope == implied }) {
					t.Errorf("%sの含意の先%sが表に無い", d.Scope, implied)
				}
				impliedResource, _, _ := strings.Cut(implied.String(), ":")
				if impliedResource != resource {
					t.Errorf("%sの含意の先%sが別のリソースである", d.Scope, implied)
				}
			}
		}
	})

	t.Run("トークンに付与できるスコープはトピックまで付けられる", func(t *testing.T) {
		t.Parallel()

		for _, d := range model.ScopeDefinitions {
			if d.APIToken && d.LowestLevel != model.ScopeLevelTopic {
				t.Errorf("%sの付けられる最も下の単位 = %d、期待値 = %d", d.Scope, d.LowestLevel, model.ScopeLevelTopic)
			}
		}
	})
}
