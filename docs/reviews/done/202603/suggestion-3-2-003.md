# コードレビュー: suggestion-3-2

## レビュー情報

| 項目                       | 内容                                          |
| -------------------------- | --------------------------------------------- |
| レビュー日                 | 2026-03-17                                    |
| 対象ブランチ               | suggestion-3-2                                |
| ベースブランチ             | suggestion-3-1                                |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md（タスク3-2） |
| 変更ファイル数             | 28 ファイル                                   |
| 変更行数（実装）           | +850 / -0 行                                  |
| 変更行数（テスト）         | +1212 / -15 行                                |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/suggestion/handler.go`
- [x] `go/internal/handler/suggestion/new.go`
- [x] `go/internal/handler/suggestion/create.go`
- [x] `go/internal/usecase/get_suggestion_new.go`
- [x] `go/internal/viewmodel/suggestion.go`
- [x] `go/internal/repository/draft_page.go`
- [x] `go/db/queries/draft_pages.sql`
- [x] `go/internal/query/draft_pages.sql.go`
- [x] `go/internal/templates/pages/suggestion/new.templ`
- [x] `go/internal/templates/pages/suggestion/new_templ.go`
- [x] `go/internal/templates/path.go`
- [x] `go/internal/templates/page_name.go`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`

### テストファイル

- [x] `go/internal/handler/suggestion/new_test.go`
- [x] `go/internal/handler/suggestion/create_test.go`
- [ ] `go/internal/handler/suggestion/index_test.go`
- [x] `go/internal/usecase/get_suggestion_new_test.go`
- [x] `go/internal/viewmodel/suggestion_test.go`

### テストユーティリティ

- [x] `go/internal/testutil/suggestion_builder.go`
- [x] `go/internal/testutil/suggestion_page_builder.go`
- [x] `go/internal/testutil/page_revision_builder.go`
- [x] `go/internal/testutil/space_member_builder.go`
- [x] `go/internal/testutil/topic_builder.go`

### 設定・その他

- [x] `docs/plans/1_doing/suggestion.md`
- [x] `docs/reviews/done/202603/suggestion-3-2-001.md`
- [x] `docs/reviews/done/202603/suggestion-3-2-002.md`

## ファイルごとのレビュー結果

### `go/internal/handler/suggestion/index_test.go` / `go/internal/handler/suggestion/new_test.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テスト戦略

**問題点・改善提案**:

- **テストヘルパーの重複**: `new_test.go` に定義されている `newGetRequest` 関数は、`index_test.go` に既に存在する `newIndexRequest` 関数と機能的に完全に同一です（どちらも chi の URL パラメータ付き GET リクエストを作成する）。同一パッケージ内での重複になっています。

  ```go
  // index_test.go
  func newIndexRequest(t *testing.T, path string, params map[string]string) *http.Request {
      // ... chi のルートコンテキスト付き GET リクエストを作成
  }

  // new_test.go（重複）
  func newGetRequest(t *testing.T, path string, params map[string]string) *http.Request {
      // ... 同一の実装
  }
  ```

  **修正案**:

  `newGetRequest` を削除し、`new_test.go` のテストケースで `newIndexRequest` を使用する。もしくは、名前をより汎用的な `newGetRequest` に統一して `index_test.go` 側を更新する。

  **対応方針**:
  - [x] `newGetRequest` を削除し、`newIndexRequest` を使用する
  - [ ] `newIndexRequest` を `newGetRequest` にリネームして統一する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

### `go/internal/templates/path.go`: `SuggestionCreatePath` と `SuggestionListPath` の重複

**ステータス**: 要確認

**現状**:

`SuggestionCreatePath` と `SuggestionListPath` は全く同じパスを生成する関数です。

```go
func SuggestionListPath(spaceIdentifier string, topicNumber int32) Path {
    return Path(fmt.Sprintf("/s/%s/topics/%d/suggestions", spaceIdentifier, topicNumber))
}

func SuggestionCreatePath(spaceIdentifier string, topicNumber int32) Path {
    return Path(fmt.Sprintf("/s/%s/topics/%d/suggestions", spaceIdentifier, topicNumber))
}
```

**提案**:

RESTful 設計では GET（一覧）と POST（作成）が同じ URL パターンを使用するため意図は理解できるが、テンプレートの `<form action>` で `SuggestionCreatePath` の代わりに `SuggestionListPath` を使用しても同一の結果になる。意味的な明確さを取るか、DRY 原則を取るかの判断。

**メリット**:

- 統一すれば関数が 1 つ減り、保守コストが下がる
- 分離する場合はテンプレートで「このパスは何のため?」が明確になる

**トレードオフ**:

- 分離を維持する方が、テンプレートのコードリーディング時に「GET 用」「POST 用」の意図が即座に分かる
- 既存の他のハンドラーで同様のパターンがあれば一貫性を維持すべき

**対応方針**:

- [x] `SuggestionCreatePath` を削除し、テンプレートで `SuggestionListPath` を使用する
- [ ] 現状のまま（意味的な明確さを優先）
- [ ] その他（下の回答欄に記入）

**回答**:

```
（ここに回答を記入）
```

## 総合評価

**評価**: Comment

**総評**:

タスク 3-2（編集提案作成のハンドラーとテンプレート）の実装として、設計は堅実で、ガイドラインにも概ね準拠しています。

**良い点**:

- 3 層アーキテクチャの依存関係ルールに正しく従っている（Handler → UseCase → Repository、Handler から Repository への直接依存なし）
- セキュリティ対策が適切（CSRF トークン、認証チェック、space_id によるクエリスコープ）
- i18n が徹底されている（ユーザー向けメッセージはすべて翻訳キーを使用）
- テストカバレッジが充実している（正常系・異常系・境界値を網羅、UseCaseテストでは非公開トピックのアクセス制御もテスト）
- テンプレートが構造体ベースのデータ渡しパターンに従っている（`context.Context` を明示的に渡していない）
- SQL クエリで `space_id` 条件を含めてスペーススコープを確保している
- バリデーションエラー時のフォーム再表示で入力値とチェック状態が適切に保持されている

**指摘事項**:

- テストヘルパー `newGetRequest` と `newIndexRequest` の重複（軽微）
