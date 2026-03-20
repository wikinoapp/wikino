# コードレビュー: suggestion-9-1

## レビュー情報

| 項目                       | 内容                                   |
| -------------------------- | -------------------------------------- |
| レビュー日                 | 2026-03-19                             |
| 対象ブランチ               | suggestion-9-1                         |
| ベースブランチ             | suggestion-8a-1                        |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md       |
| 変更ファイル数             | 22 ファイル                            |
| 変更行数（実装）           | +665 / -35 行（自動生成の templ 含む） |
| 変更行数（テスト）         | +640 / -0 行                           |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templ テンプレートガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/suggestion_change/index.go`
- [x] `go/internal/handler/suggestion_page_edit/handler.go`
- [x] `go/internal/handler/suggestion_page_edit/new.go`
- [x] `go/internal/handler/suggestion_page_edit/create.go`
- [x] `go/internal/usecase/start_suggestion_page_edit.go`
- [x] `go/internal/viewmodel/suggestion.go`
- [x] `go/internal/templates/page_name.go`
- [x] `go/internal/templates/path.go`
- [x] `go/internal/templates/pages/suggestion_change/index.templ`
- [x] `go/internal/templates/pages/suggestion_change/index_templ.go`
- [x] `go/internal/templates/pages/suggestion_page_edit/new.templ`
- [x] `go/internal/templates/pages/suggestion_page_edit/new_templ.go`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`

### テストファイル

- [x] `go/internal/handler/suggestion_page_edit/create_test.go`
- [x] `go/internal/handler/suggestion_page_edit/main_test.go`
- [x] `go/internal/usecase/start_suggestion_page_edit_test.go`

### 設定・その他

- [x] `go/docs/testing-guide.md`
- [x] `docs/plans/1_doing/suggestion.md`
- [x] `docs/reviews/done/202603/suggestion-9-1-001.md`
- [x] `docs/reviews/done/202603/suggestion-9-1-002.md`

## ファイルごとのレビュー結果

### `go/internal/handler/suggestion_page_edit/create.go`

**ステータス**: 確認済み（問題なし）

**チェックしたガイドライン**:

- [@go/docs/security-guide.md#CSRF対策](/workspace/go/docs/security-guide.md) - CSRF 対策
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - ハンドラーガイドライン

**問題点・改善提案**:

- **[@go/docs/security-guide.md#CSRF対策]**: Create ハンドラー（POST エンドポイント）で CSRF トークンの検証を行っていない。他の POST ハンドラー（例: `suggestion_apply/create.go`, `suggestion_close/create.go`）ではミドルウェアで CSRF 保護が行われているため、実際にはミドルウェアレベルで保護されている可能性がある。ただし、テストコードでは `middleware.SetCSRFTokenToContext(ctx, "test-csrf-token")` を呼んでおり、ミドルウェアが CSRF 検証を行っていることを前提としている。ミドルウェアで保護されている場合は問題ないが、確認が必要。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [ ] ミドルウェアで CSRF 保護が行われていることを確認済み（問題なし）
  - [ ] ハンドラー内で明示的に CSRF トークンを検証するコードを追加する
  - [x] その他（下の回答欄に記入）

  **回答**:

  ```
  ミドルウェアで CSRF 保護が行われていることの確認をお願いします
  ```

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 9-1（編集提案ページ編集開始の UseCase・ハンドラー）の実装として、作業計画書の仕様に沿った適切な実装がなされている。

**良かった点**:

- UseCase でのフロー制御（既存下書きの有無、同一提案ページへのリンク済み、通常編集との競合、Force による上書き）が作業計画書の設計通りに実装されている
- テストが 4 つの主要フロー（下書きなし新規作成、リンク済みリダイレクト、通常編集との競合、Force 上書き）を網羅している
- ハンドラーテストでも認証・認可（未ログイン、非スペースメンバー）のケースが含まれている
- 既存の認可パターン（`getSuggestionDetailUsecase` を使った認可チェック）との一貫性が保たれている
- 確認画面の templ テンプレートが構造体ベースのデータパターンに従っている
- CSRF トークンがフォームに含まれており、セキュリティガイドラインに準拠
- i18n が日英両方で追加されており、description も適切に記載されている
- リバースプロキシの既存パターン（`^/s/[^/]+/suggestions/\d+`）で新しい URL パスもカバーされている

**確認済み事項**:

- CSRF 保護はミドルウェア（`internal/middleware/csrf.go`）で POST/PATCH/PUT/DELETE リクエストに対して自動的に検証されている。ハンドラー内での明示的な検証は不要（確認済み）
