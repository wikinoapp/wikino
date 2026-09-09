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

// featureFlagNameType is the type whose constants AllFeatureFlagNames has to
// enumerate.
//
// [Ja] featureFlagNameType は、AllFeatureFlagNames が列挙すべき定数の型。
const featureFlagNameType = "FeatureFlagName"

// TestAllFeatureFlagNamesCoversEveryConstant checks that AllFeatureFlagNames
// lists every FeatureFlagName constant the package declares.
//
// Go cannot enumerate the members of a constant group, so the list is
// maintained by hand. Nothing else would catch a flag that is added as a
// constant but left out of the list: the seed grants exactly the flags on the
// list, and the seed's own test uses the same list as its expectation, so a
// missing entry would go unnoticed until someone wondered why the flag never
// took effect in development. Reading the constants back out of the source is
// the only mechanical fact available to compare the list against, in the same
// spirit as comparing the seed's table lists against information_schema.
//
// [Ja] TestAllFeatureFlagNamesCoversEveryConstant は、パッケージが宣言する
// FeatureFlagName 定数を AllFeatureFlagNames がすべて列挙していることを確認する。
//
// Go は定数グループのメンバーを列挙できないため、一覧は手作業で維持している。
// 定数として追加されたのに一覧から漏れたフラグは、ほかの何にも捕まらない。
// シードは一覧にあるフラグだけを付与し、シード側のテストも同じ一覧を期待値に
// 使うため、漏れがあっても、開発環境でそのフラグが効かない理由を誰かが不思議に
// 思うまで気づけない。一覧と突き合わせられる機械的な事実はソース上の定数だけで
// あり、シードのテーブル一覧を information_schema と突き合わせているのと同じ
// 考え方にあたる。
func TestAllFeatureFlagNamesCoversEveryConstant(t *testing.T) {
	t.Parallel()

	declared := featureFlagConstantValues(t)

	listed := make(map[string]bool, len(model.AllFeatureFlagNames))
	for _, name := range model.AllFeatureFlagNames {
		if listed[string(name)] {
			t.Errorf("AllFeatureFlagNames に %s が重複して入っている", name)
		}
		listed[string(name)] = true
	}

	for value, constName := range declared {
		if !listed[value] {
			t.Errorf("定数 %s (%q) が AllFeatureFlagNames に追加されていない", constName, value)
		}
	}
	for value := range listed {
		if _, exists := declared[value]; !exists {
			t.Errorf("AllFeatureFlagNames の %q に対応する %s 定数が無い", value, featureFlagNameType)
		}
	}
}

// featureFlagConstantValues returns every FeatureFlagName constant declared in
// the package, as a map from the constant's string value to its identifier.
//
// The package's own source is parsed rather than its compiled form because the
// constants are the thing under test: at run time a constant that was never
// added to AllFeatureFlagNames is indistinguishable from one that does not
// exist. Test files are skipped so that a fixture declared in a test cannot
// make the production list look incomplete.
//
// [Ja] featureFlagConstantValues は、パッケージが宣言する FeatureFlagName 定数を
// すべて返す。キーは定数の文字列値、値は定数の識別子。
//
// コンパイル後の形ではなくパッケージ自身のソースを解析するのは、定数そのものが
// 検査対象であるため。実行時には、AllFeatureFlagNames に追加されなかった定数と
// 存在しない定数を区別できない。テストファイルを除外しているのは、テスト内で
// 宣言したフィクスチャによって本体の一覧が不完全に見えないようにするため。
func featureFlagConstantValues(t *testing.T) map[string]string {
	t.Helper()

	// The test binary runs with the package directory as its working
	// directory, so the sources sit alongside it. The files are listed and
	// parsed one by one rather than with parser.ParseDir, which is deprecated
	// for ignoring build tags; here every .go file in the directory is wanted
	// regardless of the tags it carries, since a constant declared behind one
	// still has to appear in the list.
	//
	// [Ja] テストバイナリはパッケージのディレクトリを作業ディレクトリとして
	// 動くため、ソースは同じ場所にある。parser.ParseDir ではなくファイルを列挙して
	// 1 つずつ解析するのは、同関数がビルドタグを無視するとして非推奨になっている
	// ため。ここではタグに関わらずディレクトリ内のすべての .go ファイルが対象で
	// よい。タグの後ろで宣言された定数も、一覧に載る必要があることは変わらない。
	const dir = "."

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("%s の読み取りに失敗: %v", dir, err)
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
			t.Fatalf("%s の解析に失敗: %v", name, err)
		}
		collectFeatureFlagConstants(t, file, values)
	}

	if len(values) == 0 {
		t.Fatalf("%s 型の定数が 1 つも見つからなかった (解析の失敗を見逃さないための検査)", featureFlagNameType)
	}

	return values
}

// collectFeatureFlagConstants adds the FeatureFlagName constants declared in
// file to values.
//
// Within a const block a spec without its own type inherits the previous
// spec's, so the last type seen is carried forward. A spec that declares no
// value at all is the iota form, which a string-valued flag never uses; such a
// spec is skipped rather than guessed at.
//
// [Ja] collectFeatureFlagConstants は、file が宣言する FeatureFlagName 定数を
// values へ追加する。
//
// const ブロックの中では、型を持たない spec は直前の spec の型を引き継ぐため、
// 最後に見た型を持ち回す。値をまったく持たない spec は iota 形式であり、文字列を
// 値に取るフラグがこれを使うことはない。そのような spec は推測せずに読み飛ばす。
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
					t.Errorf("定数 %s の値が文字列リテラルではない", name.Name)

					continue
				}

				value, err := strconv.Unquote(lit.Value)
				if err != nil {
					t.Errorf("定数 %s の値 %s の解釈に失敗: %v", name.Name, lit.Value, err)

					continue
				}

				if previous, exists := values[value]; exists {
					t.Errorf("フィーチャーフラグ名 %q が定数 %s と %s で重複している", value, previous, name.Name)

					continue
				}
				values[value] = name.Name
			}
		}
	}
}
