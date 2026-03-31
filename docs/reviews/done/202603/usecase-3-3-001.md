# コードレビュー: usecase-3-3

## レビュー情報

| 項目                       | 内容                                                 |
| -------------------------- | ---------------------------------------------------- |
| レビュー日                 | 2026-03-27                                           |
| 対象ブランチ               | usecase-3-3                                          |
| ベースブランチ             | usecase-3-2                                          |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md |
| 変更ファイル数             | 10 ファイル                                          |
| 変更行数（実装）           | +215 / -85 行                                        |
| 変更行数（テスト）         | +167 / -51 行                                        |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - 3層アーキテクチャ、UseCase
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラー
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーション
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約、ログ出力
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティ
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テスト戦略

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/usecase/update_suggestion_comment.go`
- [x] `go/internal/handler/suggestion_comment_edit/update.go`
- [x] `go/internal/handler/suggestion_comment_edit/handler.go`
- [x] `go/internal/handler/errors.go`
- [x] `go/internal/validator/suggestion_comment.go`
- [x] `go/cmd/server/main.go`

### テストファイル

- [x] `go/internal/usecase/update_suggestion_comment_test.go`
- [x] `go/internal/validator/suggestion_comment_test.go`
- [x] `go/internal/handler/suggestion_comment_edit/edit_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/usecase-orchestration-refactor.md`

## ファイルごとのレビュー結果

### `go/internal/handler/suggestion/update.go`（変更対象外・関連ファイル）

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - コードの一貫性

**問題点・改善提案**:

- **一貫性**: 今回 `handler/errors.go` に `ValidationErrorToFormErrors` をエクスポート関数として新設し、`suggestion_comment_edit/update.go` から使用している。一方、先行実装の `suggestion/create.go` には同一ロジックのパッケージローカル関数 `validationErrorToFormErrors` が残っている。同じ変換処理が2箇所に存在しており、今後の保守性に影響する。

  **修正案**:

  `suggestion/create.go` と `suggestion/update.go` の `validationErrorToFormErrors` を `handler.ValidationErrorToFormErrors` に置き換える（本PRの範囲外でも可）。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 本PRで `suggestion` パッケージも統一する
  - [ ] 別PRで対応する（タスク3-4以降のどこかで）
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

タスク3-3（update_suggestion_comment UseCaseの移行）が作業計画書通りに正しく実装されている。主な変更点：

- **UseCase**: fetchData → authorize → validate → ビジネスロジック → 永続化のパターンが先行タスク（3-1, 3-2）と完全に一致しており、一貫性が保たれている
- **Handler**: policy/validator への直接依存が削除され、UseCase呼び出し → エラーハンドリングの薄いAdapterに変更されている
- **Validator**: Result型を廃止し `(error)` 返しに変更、`model.ValidationError` を使用
- **テスト**: 正常系2件、異常系3件（存在しないスペース、非メンバー、バリデーションエラー）と十分なカバレッジ
- **セキュリティ**: space_id スコープによるクエリ、認可チェックが適切に実装されている

指摘は1件のみ（`validationErrorToFormErrors` の重複）であり、本PRの品質自体には影響しない軽微な一貫性の問題。マージ可能と判断する。
