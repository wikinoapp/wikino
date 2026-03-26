# コードレビュー: suggestion-12-4

## レビュー情報

| 項目                       | 内容                                                  |
| -------------------------- | ----------------------------------------------------- |
| レビュー日                 | 2026-03-26                                            |
| 対象ブランチ               | suggestion-12-4                                       |
| ベースブランチ             | suggestion-12-3                                       |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md（タスク 12-4）       |
| 変更ファイル数             | 26 ファイル（レビュー・計画書除くと 22 ファイル）     |
| 変更行数（実装）           | +570 / -2 行（自動生成・翻訳・ドキュメント含む +658） |
| 変更行数（テスト）         | +538 / -44 行                                         |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - 3 層アーキテクチャ
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラー
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーション
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティ
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templ テンプレート
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テスト

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/db/queries/suggestions.sql`
- [x] `go/internal/handler/suggestion/edit.go`
- [x] `go/internal/handler/suggestion/handler.go`
- [x] `go/internal/handler/suggestion/update.go`
- [x] `go/internal/query/suggestions.sql.go`
- [x] `go/internal/repository/suggestion.go`
- [x] `go/internal/templates/page_name.go`
- [x] `go/internal/templates/pages/suggestion/edit.templ`
- [x] `go/internal/templates/pages/suggestion/edit_templ.go`
- [x] `go/internal/usecase/update_suggestion.go`
- [x] `go/internal/validator/suggestion.go`

### テストファイル

- [x] `go/internal/handler/suggestion/edit_test.go`
- [x] `go/internal/handler/suggestion/index_test.go`
- [x] `go/internal/handler/suggestion/new_test.go`
- [x] `go/internal/handler/suggestion/show_test.go`
- [x] `go/internal/handler/suggestion/update_test.go`
- [x] `go/internal/usecase/update_suggestion_test.go`
- [x] `go/internal/validator/suggestion_test.go`

### 設定・その他

- [x] `go/internal/i18n/locales/en.toml`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/docs/handler-guide.md`
- [x] `docs/plans/1_doing/suggestion.md`
- [x] `docs/reviews/suggestion-12-4-001.md`
- [x] `docs/reviews/suggestion-12-4-002.md`
- [x] `docs/reviews/suggestion-12-4-003.md`

## ファイルごとのレビュー結果

### `go/internal/handler/suggestion/edit.go` / `update.go`: ステータスチェックの重複

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md#認可チェック（Policy）](/workspace/go/docs/architecture-guide.md) - 認可チェックはHandlerで実行

**問題点・改善提案**:

- **[@go/docs/architecture-guide.md#認可チェック（Policy）]**: `edit.go:74` と `update.go:82` でステータスチェック（`output.Suggestion.Status != model.SuggestionStatusOpen`）を明示的に行っているが、`CanUpdateSuggestion` の全ポリシー実装（admin, owner, member, guest）が既に `suggestion.Status == model.SuggestionStatusOpen` をチェックしている。

  ```go
  // edit.go:74, update.go:82 - ステータスチェックが重複している
  if output.SpaceMember == nil || output.Suggestion.Status != model.SuggestionStatusOpen {
      handler.NotFound(w, r)
      return
  }
  topicPolicy := policy.NewTopicPolicy(output.SpaceMember, output.TopicMember)
  if !topicPolicy.CanUpdateSuggestion(output.Suggestion) {
      handler.NotFound(w, r)
      return
  }
  ```

  `SpaceMember == nil` チェックは `NewTopicPolicy` の前に必要だが、ステータスチェックは Policy に任せることで責務が明確になる。ただし、defense-in-depth として意図的に残している可能性もある。

  **修正案**:

  ```go
  if output.SpaceMember == nil {
      handler.NotFound(w, r)
      return
  }
  topicPolicy := policy.NewTopicPolicy(output.SpaceMember, output.TopicMember)
  if !topicPolicy.CanUpdateSuggestion(output.Suggestion) {
      handler.NotFound(w, r)
      return
  }
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 修正案の通りステータスチェックを削除し、Policy に委ねる
  - [ ] defense-in-depth として現状維持（ステータスチェックを残す）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

### `go/internal/validator/suggestion.go`: `SuggestionCreateValidator` にも本文の長さ制限を追加する

**ステータス**: 要確認

**現状**:

`SuggestionUpdateValidator` は `suggestionBodyMaxLength`（10000 文字）で本文の長さを検証しているが、`SuggestionCreateValidator` には本文の長さ制限がない。`create.go` も `body := r.FormValue("body")` で本文を受け取っているため、作成時に 10000 文字を超える本文を設定できてしまう。

```go
// SuggestionCreateValidatorInput には Body フィールドがない
type SuggestionCreateValidatorInput struct {
    Title         string
    DraftPageIDs  []model.DraftPageID
    SpaceMemberID model.SpaceMemberID
    TopicID       model.TopicID
    SpaceID       model.SpaceID
}
```

**提案**:

`SuggestionCreateValidatorInput` に `Body` フィールドを追加し、`SuggestionUpdateValidator` と同じ長さ制限を適用する。

**メリット**:

- 作成と更新で同じバリデーションルールが適用され、一貫性が保たれる
- 本文の長さ制限がバリデーション層で一元管理される

**トレードオフ**:

- 本 PR のスコープ外であり、別タスクとして対応する方が適切かもしれない
- 作成時は本文が空でも問題ないため、実際に問題が起きるケースは稀

**対応方針**:

- [x] 本 PR で対応する
- [ ] 別タスクとして対応する
- [ ] 対応不要（理由を回答欄に記入）

**回答**:

```
（ここに回答を記入）
```

## 総合評価

**評価**: Approve

**総評**:

タスク 12-4 の要件（Validator・UseCase・ハンドラー・テンプレート・ルーティング・翻訳）がすべて正しく実装されている。

**良かった点**:

- 3 層アーキテクチャに正確に従っている（Handler → UseCase → Repository → Query）
- セキュリティ面が適切: CSRF トークン、SQL クエリの `space_id` スコープ
- UseCase が「トランザクション前のデータ取得 → トランザクション内は永続化のみ」パターンを正しく実装している
- 全ユーザー向けメッセージが国際化されている（ja/en 両方）
- テストが網羅的: Handler テスト（認証・認可・バリデーション・正常系）、UseCase テスト（更新・空本文・Markdown 変換）、Validator テスト（境界値含む）
- テストヘルパー `newIndexRequest` → `newSuggestionRequest` へのリファクタリングが、既存テストとの一貫性を保ちつつ新テストのニーズに対応している
- templ テンプレートが構造体ベースのデータ受け渡しパターンに従っている
- ハンドラーガイドへのテスト関数配置ルールの追記が適切

**指摘事項**:

- ステータスチェックの重複（1 件）: Policy が既にチェックしているステータスをハンドラーでも明示的にチェックしている。defense-in-depth として意図的なら問題なし
- `SuggestionCreateValidator` の本文長さ制限の欠如（1 件、設計改善提案）: 本 PR で導入された `suggestionBodyMaxLength` が作成時には適用されていない
