package model_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/wikinoapp/wikino/go/internal/model"
)

// featureFlagNameTypeは、AllFeatureFlagNamesが列挙すべき定数の型。
const featureFlagNameType = "FeatureFlagName"

// TestAllFeatureFlagNamesCoversEveryConstantは、パッケージが宣言する
// FeatureFlagName定数をAllFeatureFlagNamesがすべて列挙していることを確認する。
//
// Goは定数グループのメンバーを列挙できないため、一覧は手作業で維持している。
// 定数として追加されたのに一覧から漏れたフラグは、ほかの何にも捕まらない。
// シードは一覧にあるフラグだけを付与し、シード側のテストも同じ一覧を期待値に
// 使うため、漏れがあっても、開発環境でそのフラグが効かない理由を誰かが不思議に
// 思うまで気づけない。一覧と突き合わせられる機械的な事実はソース上の定数だけで
// あり、シードのテーブル一覧をinformation_schemaと突き合わせているのと同じ
// 考え方にあたる。
func TestAllFeatureFlagNamesCoversEveryConstant(t *testing.T) {
	t.Parallel()

	declared := featureFlagConstantValues(t)

	listed := make(map[string]bool, len(model.AllFeatureFlagNames))
	for _, name := range model.AllFeatureFlagNames {
		if listed[string(name)] {
			t.Errorf("AllFeatureFlagNamesに%sが重複して入っている", name)
		}
		listed[string(name)] = true
	}

	for value, constName := range declared {
		if !listed[value] {
			t.Errorf("定数%s (%q) がAllFeatureFlagNamesに追加されていない", constName, value)
		}
	}
	for value := range listed {
		if _, exists := declared[value]; !exists {
			t.Errorf("AllFeatureFlagNamesの%qに対応する%s定数が無い", value, featureFlagNameType)
		}
	}
}

// featureFlagConstantValuesは、パッケージが宣言するFeatureFlagName定数を
// すべて返す。キーは定数の文字列値、値は定数の識別子。
//
// コンパイル後の形ではなくパッケージ自身のソースを解析するのは、定数そのものが
// 検査対象であるため。実行時には、AllFeatureFlagNamesに追加されなかった定数と
// 存在しない定数を区別できない。テストファイルを除外しているのは、テスト内で
// 宣言したフィクスチャによって本体の一覧が不完全に見えないようにするため。
func featureFlagConstantValues(t *testing.T) map[string]string {
	t.Helper()

	// テストバイナリはパッケージのディレクトリを作業ディレクトリとして
	// 動くため、ソースは同じ場所にある。parser.ParseDirではなくファイルを列挙して
	// 1つずつ解析するのは、同関数がビルドタグを無視するとして非推奨になっている
	// ため。ここではタグに関わらずディレクトリ内のすべての .goファイルが対象で
	// よい。タグの後ろで宣言された定数も、一覧に載る必要があることは変わらない。
	const dir = "."

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("%sの読み取りに失敗: %v", dir, err)
	}

	fset := token.NewFileSet()
	values := make(map[string]string)
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		file, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, 0)
		if err != nil {
			t.Fatalf("%sの解析に失敗: %v", name, err)
		}
		collectFeatureFlagConstants(t, file, values)
	}

	if len(values) == 0 {
		t.Fatalf("%s型の定数が1つも見つからなかった (解析の失敗を見逃さないための検査)", featureFlagNameType)
	}

	return values
}

// collectFeatureFlagConstantsは、fileが宣言するFeatureFlagName定数を
// valuesへ追加する。
//
// constブロックの中では、型を持たないspecは直前のspecの型を引き継ぐため、
// 最後に見た型を持ち回す。値をまったく持たないspecはiota形式であり、文字列を
// 値に取るフラグがこれを使うことはない。そのようなspecは推測せずに読み飛ばす。
func collectFeatureFlagConstants(t *testing.T, file *ast.File, values map[string]string) {
	t.Helper()

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.CONST {
			continue
		}

		var lastType string
		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			if ident, ok := valueSpec.Type.(*ast.Ident); ok {
				lastType = ident.Name
			}
			if lastType != featureFlagNameType || len(valueSpec.Values) == 0 {
				continue
			}

			for i, name := range valueSpec.Names {
				if i >= len(valueSpec.Values) {
					break
				}

				lit, ok := valueSpec.Values[i].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					t.Errorf("定数%sの値が文字列リテラルではない", name.Name)

					continue
				}

				value, err := strconv.Unquote(lit.Value)
				if err != nil {
					t.Errorf("定数%sの値%sの解釈に失敗: %v", name.Name, lit.Value, err)

					continue
				}

				if previous, exists := values[value]; exists {
					t.Errorf("フィーチャーフラグ名%qが定数%sと%sで重複している", value, previous, name.Name)

					continue
				}
				values[value] = name.Name
			}
		}
	}
}
