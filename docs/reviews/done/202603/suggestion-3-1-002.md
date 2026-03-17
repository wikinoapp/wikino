# コードレビュー: suggestion-3-1

## レビュー情報

| 項目                       | 内容                             |
| -------------------------- | -------------------------------- |
| レビュー日                 | 2026-03-17                       |
| 対象ブランチ               | suggestion-3-1                   |
| ベースブランチ             | suggestion-2-3                   |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md |
| 変更ファイル数             | 14 ファイル                      |
| 変更行数（実装）           | +342 / -0 行（Go実装コード）     |
| 変更行数（テスト）         | +447 / -0 行                     |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/validator/suggestion.go`
- [x] `go/internal/usecase/create_suggestion.go`
- [x] `go/internal/repository/draft_page.go`
- [x] `go/internal/repository/page_revision.go`
- [x] `go/db/queries/draft_pages.sql`
- [x] `go/db/queries/page_revisions.sql`

### テストファイル

- [x] `go/internal/validator/suggestion_test.go`
- [x] `go/internal/usecase/create_suggestion_test.go`

### 自動生成ファイル

- [x] `go/internal/query/draft_pages.sql.go`
- [x] `go/internal/query/page_revisions.sql.go`

### 設定・その他

- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`
- [x] `docs/plans/1_doing/suggestion.md`
- [x] `docs/reviews/done/202603/suggestion-3-1-001.md`

## ファイルごとのレビュー結果

### `go/internal/validator/suggestion.go`: Result構造体に`Err`フィールドがない

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - Result構造体の設計
- [@go/docs/validation-guide.md#ベストプラクティス5](/workspace/go/docs/validation-guide.md) - Result構造体でバリデーション結果を返す

**問題点・改善提案**:

- **[@go/docs/validation-guide.md]**: `SuggestionCreateValidatorResult`に`Err error`フィールドがない。他のすべてのバリデーター（`SignInCreateValidatorResult`, `PasswordUpdateValidatorResult`, `AccountCreateValidatorResult`等）は`Err error`フィールドを持っており、Handler側でシステムエラーとバリデーションエラーを区別できるようになっている。

  現在の実装では、下書きページの取得でDBエラーが発生した場合に`formErrors.AddGlobal`でフォームエラーとして処理しているが、これではHandlerがシステムエラーを検知してログに記録する（`slog.ErrorContext`）ことができない。

  ```go
  // 現在のコード（validator/suggestion.go:68-71）
  draftPage, err := v.draftPageRepo.FindByID(ctx, draftPageID, input.SpaceID)
  if err != nil {
      formErrors.AddGlobal(i18n.T(ctx, "validation_system_error"))
      return &SuggestionCreateValidatorResult{FormErrors: formErrors}
  }
  ```

  **修正案**:

  ```go
  // Result構造体にErrフィールドを追加
  type SuggestionCreateValidatorResult struct {
      FormErrors *session.FormErrors
      DraftPages []*model.DraftPage
      Err        error
  }

  // エラー発生時はErrフィールドで返す
  draftPage, err := v.draftPageRepo.FindByID(ctx, draftPageID, input.SpaceID)
  if err != nil {
      return &SuggestionCreateValidatorResult{Err: err}
  }
  ```

  **対応方針**:
  - [x] 修正案の通り`Err`フィールドを追加し、システムエラーは`Err`で返す
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/usecase/create_suggestion.go`: コメントのステップ番号が重複している

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約

**問題点・改善提案**:

- **[@go/docs/coding-guide.md]**: `Execute`メソッド内のコメントのステップ番号が重複している。ステップ1〜5の後に再びステップ4が出現する。

  ```go
  // 1. トピック内の次の編集提案番号を取得（行79）
  // 2. 編集提案の本文HTMLをレンダリング（行85）
  // 3. Wikiリンク解決（行88）
  // 4. スタンドアロン画像のラッピング（行97）
  // 5. 編集提案を作成（行100）
  // 4. 各下書きページからSuggestionPage...を作成（行115）← 重複
  ```

  **修正案**:

  ```go
  // 6. 各下書きページからSuggestionPageとSuggestionPageRevisionを作成
  ```

  **対応方針**:
  - [x] 修正案の通りステップ番号を6に修正する
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

タスク3-1（編集提案作成のValidator・UseCase）の実装として、作業計画書の要件を満たしている。

**良かった点**:

- アーキテクチャガイドに準拠した3層構造の実装（UseCase → Repository → Query）
- WithTxパターンによるトランザクション管理が適切
- `resolveLinkedPages`関数は`resolveAndCreateLinkedPages`の読み取り専用版として適切に分離されている
- セキュリティ面: SQLクエリは`space_id`でスコープされており、バリデーターで`SpaceMemberID`と`TopicID`のチェックも行われている
- テストは正常系（1ページ、複数ページ）と異常系（リビジョンなし、他メンバーの下書き）を適切にカバー
- i18n対応が完全（ja/en両方）

**修正が必要な点**:

1. **Validator Result構造体の`Err`フィールド欠如**: 既存の全バリデーターとの一貫性を保つために必要。Handlerがシステムエラーとバリデーションエラーを区別できるようになる
2. **コメントのステップ番号重複**: 軽微だが修正が必要
