# コードレビュー: usecase-3-1

## レビュー情報

| 項目                       | 内容                                                 |
| -------------------------- | ---------------------------------------------------- |
| レビュー日                 | 2026-03-27                                           |
| 対象ブランチ               | usecase-3-1                                          |
| ベースブランチ             | usecase-2-1                                          |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md |
| 変更ファイル数             | 15 ファイル（レビューファイル・計画書除く）          |
| 変更行数（実装）           | +340 / -175 行                                       |
| 変更行数（テスト）         | +248 / -183 行                                       |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - ハンドラーガイド
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/suggestion/create.go`
- [x] `go/internal/handler/suggestion/handler.go`
- [x] `go/internal/handler/suggestion_comment/create.go`
- [x] `go/internal/policy/topic.go`
- [x] `go/internal/policy/topic_admin.go`
- [x] `go/internal/policy/topic_guest.go`
- [x] `go/internal/policy/topic_member.go`
- [x] `go/internal/policy/topic_owner.go`
- [ ] `go/internal/usecase/create_suggestion.go`
- [x] `go/internal/validator/suggestion.go`

### テストファイル

- [x] `go/internal/handler/suggestion/index_test.go`
- [x] `go/internal/handler/suggestion_comment/create_test.go`
- [ ] `go/internal/usecase/create_suggestion_test.go`
- [x] `go/internal/validator/suggestion_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/usecase-orchestration-refactor.md`

## ファイルごとのレビュー結果

### `go/internal/usecase/create_suggestion.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - UseCase のルール、依存関係
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約

**問題点・改善提案**:

- **未使用フィールド**: `createSuggestionInput` 構造体（225 行目）の `SpaceIdentifier` フィールドが `createSuggestion` メソッド内で参照されていない

  ```go
  // 225行目: createSuggestionInput 構造体
  type createSuggestionInput struct {
      SpaceID         model.SpaceID
      SpaceIdentifier model.SpaceIdentifier // ← このフィールドが未使用
      TopicID         model.TopicID
      // ...
  }
  ```

  **修正案**:

  `SpaceIdentifier` フィールドを `createSuggestionInput` から削除し、`Execute` メソッド内の構築箇所からも除去する。

  **対応方針**:
  - [x] 未使用フィールドを削除する
  - [ ] 将来必要になる可能性があるため残す（理由を回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/usecase/create_suggestion_test.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド

**問題点・改善提案**:

- **テストカバレッジの減少**: "リンク付き提案" テストケース（旧: 170 行付近、新: 177 行付近）で、`linkedPageID` の作成と `SuggestionPage.LinkedPageIDs` のアサーションが削除されている

  変更前:

  ```go
  linkedPageID := testutil.NewPageBuilderDB(t, db).
      WithSpaceID(spaceID).
      WithTopicID(topicID).
      WithNumber(2).
      WithTitle("Linked Page").
      Build()
  // ...
  if len(sp.LinkedPageIDs) != 1 || sp.LinkedPageIDs[0] != linkedPageID {
      t.Errorf("SuggestionPage.LinkedPageIDs = %v, want [%v]", sp.LinkedPageIDs, linkedPageID)
  }
  ```

  変更後: `linkedPageID` の作成とアサーションが削除されている。

  DraftPage を DB ビルダー経由で作成するようになったことで `LinkedPageIDs` の設定が困難になった可能性があるが、このフィールドの検証が失われている点を確認したい。

  **修正案**:

  DraftPage ビルダーに `WithLinkedPageIDs` メソッドがあれば追加してアサーションを復元する。なければ、この検証は DraftPage 作成のユースケースやリポジトリテストで担保されているか確認する。

  **対応方針**:
  - [x] LinkedPageIDs のアサーションを復元する
  - [ ] 別のテストで担保されているため不要（理由を回答欄に記入）
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

作業計画書タスク 3-1 の要件に正確に沿った実装。パイロット移行（タスク 2-1, suggestion_comment）で確立したパターンを一貫して適用している。

**良かった点**:

- UseCase の `Execute` メソッドが作業計画書の 5 ステップ（データ取得 → 認可 → バリデーション → ビジネスロジック → 永続化）に忠実に構造化されている
- Handler が薄い Adapter に変わり、HTTP の入出力変換に徹している
- `validationErrorToFormErrors` ブリッジ関数が一時的なものとして明確にコメントされている
- Validator の返り値が `(data, error)` パターンに変更され、Go の標準的なエラーハンドリングが使える
- Policy に `CanCreateSuggestion` が正しく追加されている（Guest は公開トピックのみ許可）
- suggestion_comment の Forbidden 応答が 403 → 404 に統一され、セキュリティが向上
- UseCase テストに異常系テスト（存在しないスペース、非メンバー、バリデーションエラー）が追加されている
- 既存の Handler テスト（`create_test.go`）が変更なしで動作する設計（外部振る舞いが不変のリファクタリング）
