# コードレビュー: suggestion-8-1

## レビュー情報

| 項目                       | 内容                                   |
| -------------------------- | -------------------------------------- |
| レビュー日                 | 2026-03-19                             |
| 対象ブランチ               | suggestion-8-1                         |
| ベースブランチ             | suggestion-7-2                         |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md       |
| 変更ファイル数             | 15 ファイル（自動生成 1 ファイル含む） |
| 変更行数（実装）           | +308 / -2 行                           |
| 変更行数（テスト）         | +584 / -0 行                           |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/suggestion/show.go`
- [x] `go/internal/handler/suggestion_close/handler.go`
- [x] `go/internal/handler/suggestion_close/create.go`
- [x] `go/internal/usecase/close_suggestion.go`
- [x] `go/internal/policy/suggestion.go`
- [x] `go/internal/templates/pages/suggestion/show.templ`
- [x] `go/internal/templates/pages/suggestion/show_templ.go`（自動生成）
- [x] `go/internal/templates/path.go`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`

### テストファイル

- [x] `go/internal/handler/suggestion_close/create_test.go`
- [x] `go/internal/handler/suggestion_close/main_test.go`
- [x] `go/internal/policy/suggestion_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/suggestion.md`

## ファイルごとのレビュー結果

### `go/internal/handler/suggestion_close/create_test.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テスト戦略
- 作業計画書 タスク 8-1 - クローズ権限（作成者またはスペースオーナー/トピック管理者）

**問題点・改善提案**:

- **[作業計画書 8-1]**: トピック管理者によるクローズの成功テストケースがない

  作業計画書ではクローズ権限を「作成者本人、スペースオーナー、またはトピック管理者」と定義している。ポリシーの単体テスト（`suggestion_test.go`）ではトピック管理者のケースがカバーされているが、ハンドラーの統合テストにはトピック管理者が正常にクローズできるテストケースがない。

  **修正案**: `TestCreate_トピック管理者がクローズできる` テストケースを追加する。

  **対応方針**:
  - [x] テストケースを追加する
  - [ ] ポリシーテストでカバーされているため不要（理由を回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

- **[作業計画書 8-1 / べき等性の仕様]**: 反映済み（Applied）の編集提案に対するクローズリクエストのテストケースがない

  ハンドラーの 80-84 行目に「オープンステータスでなければクローズ不可」のロジックがあるが、この分岐（Applied ステータスの場合）をカバーするテストがない。べき等性のテストはクローズ済み（Closed）のケースのみ。

  **修正案**: `TestCreate_反映済みの編集提案はクローズできずエラーが返る` テストケースを追加する。

  **対応方針**:
  - [x] テストケースを追加する
  - [ ] 既存のテストで十分（理由を回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

### `go/internal/policy/suggestion.go`: CanCloseSuggestion の実装パターンについて

**ステータス**: 要確認

**現状**:

`CanCloseSuggestion` はパッケージレベルの関数として実装されている。一方、`CanApplySuggestion` は `TopicPolicy` インターフェースのメソッドとして実装されており、同じ Suggestion リソースに対する権限判定で異なるパターンが使われている。

```go
// apply は TopicPolicy インターフェースのメソッド
topicPolicy := policy.NewTopicPolicy(output.SpaceMember, output.TopicMember)
canApply = topicPolicy.CanApplySuggestion(output.Suggestion)

// close はパッケージレベルの関数
canClose = policy.CanCloseSuggestion(output.SpaceMember, output.TopicMember, output.Suggestion)
```

**提案**:

現状のままとする理由は理解できる: `CanCloseSuggestion` は「作成者本人」という判定を含むため、`SpaceMemberID` が必要だが、既存の `TopicPolicy` 実装体（`topicOwnerPolicy`, `topicAdminPolicy` 等）はこの情報を保持していない。`TopicPolicy` に `CanCloseSuggestion` を追加するには、各実装体に `spaceMemberID` フィールドを追加する必要があり、既存の設計への影響が大きい。

ただし、同じリソース（Suggestion）に対する権限チェックが 2 つの異なるパターンで実装されていることは、将来の開発者にとって混乱の原因になる可能性がある。

**メリット**（現状維持の場合）:

- 既存の `TopicPolicy` 実装への変更が不要
- `CanCloseSuggestion` に必要な全データが引数として明示的に渡される
- シンプルで理解しやすい

**トレードオフ**:

- 同じリソースに対する権限判定で 2 つのパターンが混在する
- 将来 Suggestion 関連の権限が追加される場合、どちらのパターンに従うべきか判断が必要になる

**対応方針**:

- [ ] 現状のまま（パッケージレベル関数を維持）
- [x] TopicPolicy に CanCloseSuggestion を追加する（既存実装の変更が必要）
- [ ] その他（下の回答欄に記入）

**回答**:

```
（ここに回答を記入）
```

## 総合評価

**評価**: Comment

**総評**:

フェーズ 8-1（編集提案のクローズ）の実装が作業計画書に沿って正しく行われている。

**良い点**:

- 既存の `suggestion_apply` ハンドラーと一貫性のあるパターンで実装されている（ファイル構成、認証・認可フロー、べき等性の処理）
- セキュリティ面の配慮が適切: CSRF トークン、認証チェック、スペースメンバーシップ確認、権限チェックの順序が正しい
- UseCase 内でも `SpaceID` スコープ付きでクエリが行われており、セキュリティガイドラインに準拠
- ポリシーテスト（`suggestion_test.go`）が網羅的で、テーブル駆動テストの使い方が適切
- i18n 対応が完全（ja.toml, en.toml の両方にメッセージ追加済み）
- べき等性の設計が作業計画書のステータス変更仕様に準拠
- テンプレートの CSRF トークンが含まれている

**改善が望ましい点**:

- ハンドラーテストにトピック管理者のクローズ成功ケースがない（ポリシーテストではカバー済み）
- Applied ステータスのエラーケースのテストがない
- `CanCloseSuggestion` と `CanApplySuggestion` で異なる実装パターン（いずれも軽微な指摘であり、修正は任意）
