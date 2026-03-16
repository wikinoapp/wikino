# コードレビュー: go-topic-3-2

## レビュー情報

| 項目                       | 内容                                                                      |
| -------------------------- | ------------------------------------------------------------------------- |
| レビュー日                 | 2026-03-08                                                                |
| 対象ブランチ               | go-topic-3-2                                                              |
| ベースブランチ             | go-topic                                                                  |
| 作業計画書（指定があれば） | docs/plans/1_doing/topic-show-go-migration.md                             |
| 変更ファイル数             | 7 ファイル                                                                |
| 変更行数（実装）           | +281 行（handler.go: 44, show.go: 221, main.go: 14, reverse_proxy.go: 1） |
| 変更行数（テスト）         | +421 行（show_test.go: 409, main_test.go: 12）                            |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/handler/topic/handler.go`
- [ ] `go/internal/handler/topic/show.go`
- [x] `go/cmd/server/main.go`
- [x] `go/internal/middleware/reverse_proxy.go`

### テストファイル

- [x] `go/internal/handler/topic/main_test.go`
- [x] `go/internal/handler/topic/show_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/topic-show-go-migration.md`

## ファイルごとのレビュー結果

### `go/internal/handler/topic/show.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

**問題点・改善提案**:

- **未使用パラメータ**: `canCreateTopicPage` 関数の `topic *model.Topic` パラメータが使用されていない（213 行目）

  ```go
  // 現在のコード
  func canCreateTopicPage(spaceMember *model.SpaceMember, topicMember *model.TopicMember, topic *model.Topic) bool {
      if spaceMember == nil {
          return false
      }
      if spaceMember.Role == model.SpaceMemberRoleOwner {
          return true
      }
      return topicMember != nil
  }
  ```

  Rails 版の `can_create_page?` は `topic_record` を `in_same_topic?` チェックに使用しているが、Go 版では `topicMember` がすでにトピック固有で取得されているため、`topic` パラメータは不要。YAGNI 原則に従い、未使用パラメータは削除すべき。

  **修正案**:

  ```go
  func canCreateTopicPage(spaceMember *model.SpaceMember, topicMember *model.TopicMember) bool {
      if spaceMember == nil {
          return false
      }
      if spaceMember.Role == model.SpaceMemberRoleOwner {
          return true
      }
      return topicMember != nil
  }
  ```

  呼び出し元（133 行目）も修正:

  ```go
  canCreatePage := canCreateTopicPage(spaceMember, topicMember)
  ```

  **対応方針**:
  - [x] 修正案の通り `topic` パラメータを削除する
  - [ ] 将来の拡張に備えて現状のまま残す（理由を回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Comment

**総評**:

全体的に高品質な実装です。ガイドラインに準拠した構成で、既存のハンドラーパターンとの一貫性も保たれています。

**良かった点**:

- Handler 構造体が handler-guide.md の命名規則と構造に準拠（8 フィールド以内）
- 権限チェックが作業計画書の仕様通りに実装されている（公開/非公開トピック、スペースオーナー、トピックメンバー）
- `canUpdateTopic` / `canCreateTopicPage` を関数として切り出しており、テスタビリティが高い
- テストが正常系・異常系を網羅しており、権限パターンのテストが充実（8 テストケース）
- リバースプロキシの正規表現パターンが既存パターンと一致
- main.go のルーティング登録が既存パターンに従っている
- ログ出力が `slog.ErrorContext` を使用しておりコーディング規約準拠
- `space_id` をクエリスコープに含めており、セキュリティガイドライン準拠（`FindPinnedByTopic`, `FindRegularByTopicPaginated` の引数に `spaceID` を渡している）

**指摘事項**:

- `canCreateTopicPage` 関数に未使用の `topic` パラメータがある（軽微、golangci-lint はパスしている）
