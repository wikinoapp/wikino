# コードレビュー: error-pages-2-1

## レビュー情報

| 項目                       | 内容                                       |
| -------------------------- | ------------------------------------------ |
| レビュー日                 | 2026-03-16                                 |
| 対象ブランチ               | error-pages-2-1                            |
| ベースブランチ             | error-pages-1-1                            |
| 作業計画書（指定があれば） | docs/plans/1_doing/go-error-pages.md       |
| 変更ファイル数             | 9 ファイル（レビュードキュメント含む）     |
| 変更行数（実装）           | +119 行（Go, templ, toml, main.go の合計） |
| 変更行数（テスト）         | +73 行                                     |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templ テンプレートガイド
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/handler/errors.go`
- [x] `go/internal/templates/pages/errors/not_found.templ`
- [x] `go/internal/templates/pages/errors/not_found_templ.go`（自動生成）
- [x] `go/cmd/server/main.go`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`

### テストファイル

- [ ] `go/internal/handler/errors_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/go-error-pages.md`
- [x] `docs/reviews/done/202603/error-pages-2-1-001.md`

## ファイルごとのレビュー結果

### `go/internal/handler/errors_test.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約

**問題点・改善提案**:

- **[コード品質]**: テスト構造体に `unwantContents` フィールドが定義されているが、どのテストケースでも設定されておらず、検証ロジックも存在しない。未使用のフィールドは削除すべき。

  ```go
  // 問題のあるコード（L16-22）
  tests := []struct {
      name           string
      locale         string
      wantStatus     int
      wantContents   []string
      unwantContents []string  // ← 未使用
  }{
  ```

  **修正案**:

  ```go
  tests := []struct {
      name         string
      locale       string
      wantStatus   int
      wantContents []string
  }{
  ```

  **対応方針**:
  - [x] 修正案の通り `unwantContents` フィールドを削除する
  - [ ] 将来的に使用する予定があるため残す（回答欄に理由を記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Comment

**総評**:

作業計画書タスク 2-1 の要件がすべて正しく実装されている。

**良い点**:

- メンテナンスページ（`maintenance.templ`）と同じスタンドアロン方式で統一されたパターン
- `Content-Type` ヘッダーを `WriteHeader` の前に設定する正しい順序
- テンプレートのレンダリングエラーを `_` で無視する判断（レスポンス書き込み後のため）とその理由のコメント
- `noindex, nofollow` メタタグによるSEO考慮
- 日英両方のテストケースによる i18n 検証
- 作業計画書で指定されていなかった `error_not_found_title` 翻訳キーの追加（`<title>` タグに必要）

**指摘事項**:

- テスト構造体の未使用フィールド `unwantContents` の削除（軽微）
