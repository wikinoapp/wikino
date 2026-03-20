# コードレビュー: suggestion-7-2

## レビュー情報

| 項目                       | 内容                                   |
| -------------------------- | -------------------------------------- |
| レビュー日                 | 2026-03-19                             |
| 対象ブランチ               | suggestion-7-2                         |
| ベースブランチ             | suggestion-7-1c                        |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md       |
| 変更ファイル数             | 18 ファイル                            |
| 変更行数（実装）           | +233 / -3 行（自動生成ファイルを除く） |
| 変更行数（テスト）         | +486 / 0 行                            |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/suggestion/show.go`
- [x] `go/internal/handler/suggestion_apply/create.go`
- [x] `go/internal/handler/suggestion_apply/handler.go`
- [ ] `go/internal/policy/topic.go`
- [ ] `go/internal/policy/topic_admin.go`
- [x] `go/internal/policy/topic_guest.go`
- [x] `go/internal/policy/topic_member.go`
- [ ] `go/internal/policy/topic_owner.go`
- [x] `go/internal/templates/pages/suggestion/show.templ`
- [x] `go/internal/templates/pages/suggestion/show_templ.go`
- [x] `go/internal/templates/path.go`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`

### テストファイル

- [x] `go/internal/handler/suggestion_apply/create_test.go`
- [x] `go/internal/handler/suggestion_apply/main_test.go`
- [ ] `go/internal/policy/topic_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/suggestion.md`

## ファイルごとのレビュー結果

### `go/internal/policy/topic_owner.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド

**問題点・改善提案**:

- **[@go/docs/security-guide.md#認可]**: `CanApplySuggestion` が `suggestion.SpaceID` をチェックしていない（防御の多層化の欠如）

  同じ構造体の `CanUpdatePage` は `p.spaceID == page.SpaceID` をチェックしているが、`CanApplySuggestion` は引数を無視（`_`）して `spaceMemberActive` のみチェックしている。Handlerで既にスペースの整合性は検証されているが、防御の多層化として他のメソッドと同様にスペースIDを検証すべき。

  ```go
  // 現在のコード
  func (p *topicOwnerPolicy) CanApplySuggestion(_ *model.Suggestion) bool {
      return p.spaceMemberActive
  }
  ```

  **修正案**:

  ```go
  func (p *topicOwnerPolicy) CanApplySuggestion(suggestion *model.Suggestion) bool {
      return p.spaceMemberActive && p.spaceID == suggestion.SpaceID
  }
  ```

  **対応方針**:
  - [x] 修正案の通りスペースIDチェックを追加する
  - [ ] 現状のまま（理由を回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/policy/topic_admin.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

**問題点・改善提案**:

- **[@go/docs/security-guide.md#認可]**: `topicOwnerPolicy` のスペースIDチェックを追加する場合、`topicAdminPolicy.CanApplySuggestion` のトピックIDチェックとの一貫性を確認する必要がある

  `topicAdminPolicy.CanApplySuggestion` は `p.topicID == suggestion.TopicID` をチェックしており、これは他のメソッド（`CanUpdatePage` の `p.topicID == page.TopicID`）と一貫性がある。問題なし。

  ただし、`topicOwnerPolicy` 修正時にテストの `newSuggestion` ヘルパーも `SpaceID` を設定するよう更新が必要（下記 `topic_test.go` 参照）。

  **対応方針**:
  - [x] `topicOwnerPolicy` のスペースIDチェック追加に合わせてテストヘルパーも更新する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/policy/topic_test.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド

**問題点・改善提案**:

- **[@go/docs/testing-guide.md]**: テストヘルパー `newSuggestion` に `SpaceID` が設定されていない

  `topicOwnerPolicy.CanApplySuggestion` にスペースIDチェックを追加する場合、`newSuggestion` ヘルパーにも `SpaceID` を設定する必要がある。現状ではゼロ値のままであり、スペースIDチェックのテストが正しく動作しない。

  ```go
  // 現在のコード
  func newSuggestion(topicID model.TopicID) *model.Suggestion {
      return &model.Suggestion{
          ID:      "suggestion-1",
          TopicID: topicID,
          Status:  model.SuggestionStatusOpen,
      }
  }
  ```

  **修正案**:

  ```go
  func newSuggestion(spaceID model.SpaceID, topicID model.TopicID) *model.Suggestion {
      return &model.Suggestion{
          ID:      "suggestion-1",
          SpaceID: spaceID,
          TopicID: topicID,
          Status:  model.SuggestionStatusOpen,
      }
  }
  ```

  呼び出し箇所もすべて `newSuggestion("space-1", "topic-1")` に更新する。

  **対応方針**:
  - [x] 修正案の通り `newSuggestion` にスペースIDを追加する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/policy/topic.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド

**問題点・改善提案**:

- **[@go/docs/architecture-guide.md#認可チェック]**: `CanApplySuggestion` のシグネチャで `*model.Suggestion` を受け取る設計は適切

  Policyは `model` のみに依存し、HandlerがUseCaseの出力を使ってPolicyチェックを実行する設計パターンに準拠している。ただし、`topicOwnerPolicy` でその引数を `_` で無視している点が問題（上記 `topic_owner.go` で指摘済み）。

  **対応方針**:
  - [x] `topic_owner.go` の修正に合わせて対応する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Request Changes

**総評**:

タスク 7-2（編集提案反映のハンドラーとPolicy）の実装として、作業計画書に記載された要件を網羅的に実装している。

- ハンドラーガイドに準拠したディレクトリ構造（`suggestion_apply/handler.go`, `create.go`）
- 認可チェックをHandlerで実行するアーキテクチャパターンに準拠
- べき等性の実装（反映済みの場合は何もせず成功リダイレクト）
- テストカバレッジ（認証・認可・正常系・べき等性の6ケース）
- i18n対応（ja/en両方に翻訳追加）
- CSRFトークンを含むフォーム送信

修正が必要な点は `topicOwnerPolicy.CanApplySuggestion` のスペースIDチェック欠如のみ。他のメソッド（`CanUpdatePage` 等）と一貫性を保つため、防御の多層化としてスペースIDの検証を追加すべき。テストヘルパーの更新も併せて必要。
