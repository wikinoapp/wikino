# コードレビュー: suggestion-2-2

## レビュー情報

| 項目                       | 内容                                   |
| -------------------------- | -------------------------------------- |
| レビュー日                 | 2026-03-16                             |
| 対象ブランチ               | suggestion-2-2                         |
| ベースブランチ             | develop                                |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md       |
| 変更ファイル数             | 14 ファイル（自動生成 1 ファイル含む） |
| 変更行数（実装）           | +496 / -19 行                          |
| 変更行数（テスト）         | +364 / -11 行                          |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/suggestion/handler.go`
- [ ] `go/internal/handler/suggestion/index.go`
- [x] `go/internal/i18n/locales/en.toml`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/templates/page_name.go`
- [ ] `go/internal/templates/pages/suggestion/index.templ`
- [x] `go/internal/templates/path.go`
- [x] `go/internal/usecase/get_suggestion_list.go`

### テストファイル

- [x] `go/internal/handler/suggestion/index_test.go`
- [x] `go/internal/handler/suggestion/main_test.go`
- [x] `go/internal/usecase/get_suggestion_list_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/suggestion.md`
- [x] `go/internal/templates/pages/suggestion/index_templ.go`（自動生成）

## ファイルごとのレビュー結果

### `go/internal/handler/suggestion/index.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - ハンドラーガイドライン
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

**問題点・改善提案**:

- **[@go/docs/handler-guide.md#ファイル命名規則]**: `index.go` がimportで `usecase` パッケージをインポートしているが、`usecase.GetSuggestionListInput` の `Statuses` フィールドにハンドラー内で決定した `[]model.SuggestionStatus` を渡している。これ自体はアーキテクチャ違反ではないが、ステータスのフィルタリングロジック（どのステータスがオープン/クローズに属するか）がハンドラーとUseCaseの両方に分散している点が気になる。UseCaseの `Execute` 内でもオープン件数・クローズ件数のために同じステータスグループを定義している（L112-114, L121-123）。

  ```go
  // index.go: ハンドラー側でステータスを決定
  if showClosed {
      statuses = []model.SuggestionStatus{model.SuggestionStatusApplied, model.SuggestionStatusClosed}
  } else {
      statuses = []model.SuggestionStatus{model.SuggestionStatusDraft, model.SuggestionStatusOpen}
  }
  ```

  ```go
  // get_suggestion_list.go: UseCase内でも同じグルーピングを定義
  openCount, err := uc.suggestionRepo.CountByTopicAndStatuses(ctx, topic.ID, space.ID, []model.SuggestionStatus{
      model.SuggestionStatusDraft,
      model.SuggestionStatusOpen,
  })
  ```

  **修正案**:

  UseCase の入力を `ShowClosed bool` に変更し、ステータスのグルーピングロジックをUseCase内に集約する案。ただし、現在の実装はトピック詳細ハンドラー（`topic/show.go`）のパターンに倣っており、ハンドラーが必要なデータを指定してUseCaseに渡す設計と一貫している。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] UseCase の入力を `ShowClosed bool` に変更してロジックを集約する
  - [ ] 現状のまま（ハンドラーが必要データを指定するパターンとして一貫している）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/templates/pages/suggestion/index.templ`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド

**問題点・改善提案**:

- **パス生成の重複**: `IndexData.suggestionsPath()` メソッドが `fmt.Sprintf` でパスを生成しているが、`go/internal/templates/path.go` に同じベースパスを生成する `SuggestionListPath()` ヘルパーが今回の差分で追加されている。ベースパスの生成ロジックが2箇所に存在する。

  ```go
  // index.templ: テンプレート内でパス生成
  func (d IndexData) suggestionsPath(tab string) string {
      if tab == "" {
          return fmt.Sprintf("/s/%s/topics/%d/suggestions", d.Space.Identifier, d.Topic.Number)
      }
      return fmt.Sprintf("/s/%s/topics/%d/suggestions?tab=%s", d.Space.Identifier, d.Topic.Number, tab)
  }
  ```

  ```go
  // path.go: パスヘルパーで同じベースパスを生成
  func SuggestionListPath(spaceIdentifier string, topicNumber int32) Path {
      return Path(fmt.Sprintf("/s/%s/topics/%d/suggestions", spaceIdentifier, topicNumber))
  }
  ```

  **修正案**:

  `suggestionsPath()` のベースパス部分を `SuggestionListPath()` を使うように変更する:

  ```go
  func (d IndexData) suggestionsPath(tab string) string {
      base := string(templates.SuggestionListPath(d.Space.Identifier.String(), d.Topic.Number))
      if tab == "" {
          return base
      }
      return base + "?tab=" + tab
  }
  ```

  あるいは、`SuggestionListPath` はタスク2-3（トピック詳細画面のタブ）で使われる予定なので、現在のテンプレートではクエリパラメータ付きの独自メソッドを持つのは合理的とも言える。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] `SuggestionListPath` を使うように `suggestionsPath` を変更する
  - [ ] 現状のまま（クエリパラメータ付きの独自メソッドとして合理的）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク2-2（編集提案一覧のハンドラーとテンプレート）の要件を満たしている。

**良かった点**:

- **既存パターンとの一貫性**: `topic/show.go` の実装パターン（URLパラメータ取得 → UseCase呼び出し → ViewModel変換 → レイアウト構築 → レンダリング）に忠実に従っている
- **UseCase の設計**: `GetTopicDetailUsecase` と同じ構造で、スペース/トピック取得・権限チェック・データ集約を一貫したパターンで実装している
- **テストの充実**: 404ケース（存在しないスペース/トピック/不正番号）、権限チェック（公開トピック未ログイン閲覧・非公開トピック拒否・オーナーアクセス）、タブフィルタリングなど網羅的
- **国際化対応**: ja.toml/en.toml の両方に適切な翻訳が追加されており、`description` も記載されている
- **セキュリティ**: 非公開トピックの権限チェック、`space_id` によるクエリスコープが適切に実装されている
- **テンプレート設計**: `IndexData` 構造体ベースのパターン、`ctx` の暗黙的使用など templ-guide に準拠

**指摘事項のサマリー**:

- 2件とも「要確認」レベルであり、既存パターンとの一貫性は保たれているため、現状維持でも問題ない
- パス生成の重複は軽微だが、今後の保守性を考えると集約が望ましい
