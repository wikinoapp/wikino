# コードレビュー: suggestion-9-1

## レビュー情報

| 項目                       | 内容                                     |
| -------------------------- | ---------------------------------------- |
| レビュー日                 | 2026-03-19                               |
| 対象ブランチ               | suggestion-9-1                           |
| ベースブランチ             | suggestion-8a-1                          |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md         |
| 変更ファイル数             | 19 ファイル（自動生成 2 ファイルを含む） |
| 変更行数（実装）           | +646 / -42 行（自動生成を除く）          |
| 変更行数（テスト）         | +638 行                                  |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templ テンプレートガイド
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
- [x] `go/internal/templates/pages/suggestion_change/index.templ`
- [x] `go/internal/templates/pages/suggestion_page_edit/new.templ`
- [x] `go/internal/templates/path.go`
- [x] `go/internal/templates/page_name.go`

### テストファイル

- [ ] `go/internal/handler/suggestion_page_edit/create_test.go`
- [x] `go/internal/handler/suggestion_page_edit/main_test.go`
- [x] `go/internal/usecase/start_suggestion_page_edit_test.go`

### 設定・翻訳・その他

- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`
- [x] `go/internal/templates/pages/suggestion_change/index_templ.go`（自動生成）
- [x] `go/internal/templates/pages/suggestion_page_edit/new_templ.go`（自動生成）
- [x] `docs/plans/1_doing/suggestion.md`

## ファイルごとのレビュー結果

### `go/internal/handler/suggestion_page_edit/create_test.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド

**問題点・改善提案**:

- **[@go/docs/testing-guide.md#テストのベストプラクティス]**: `TestCreate_下書きなしの場合にページ編集画面にリダイレクトされる`（178 行目）に `t.Parallel()` が欠けている

  他のテスト関数（`TestCreate_未ログインでサインインにリダイレクトされる`、`TestCreate_スペースメンバーでないユーザーは403が返る`、`TestCreate_通常編集の下書きがある場合に確認画面にリダイレクトされる`）はすべて `t.Parallel()` を呼んでいる。このテストは `GetTestDB()` で直接 DB アクセスするパターンだが、ユニークな識別子を使用しているため並行実行は安全であり、`t.Parallel()` を追加すべき。

  ```go
  // 現在のコード（178行目）
  func TestCreate_下書きなしの場合にページ編集画面にリダイレクトされる(t *testing.T) {
  	db := testutil.GetTestDB()
  ```

  **修正案**:

  ```go
  func TestCreate_下書きなしの場合にページ編集画面にリダイレクトされる(t *testing.T) {
  	t.Parallel()

  	db := testutil.GetTestDB()
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 修正案の通り `t.Parallel()` を追加する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Comment

**総評**:

作業計画書タスク 9-1（編集提案ページ編集開始の UseCase・ハンドラー）の実装として、要件を正確に満たしている。

**良かった点**:

- 3 層アーキテクチャの依存関係ルールに正しく従っている（Handler → UseCase → Repository）
- セキュリティ面：認証チェック、スペースメンバー権限チェック、CSRF トークン、オープンステータスの検証がすべて適切
- space_id によるクエリスコープが UseCase 内の全リポジトリ呼び出しで徹底されている
- 確認画面フロー（通常編集の下書きとの競合）が作業計画書の「DraftPage と編集提案の連携」設計に沿って実装されている
- UseCase テストが正常系 4 パターン（新規作成、リンク済み、Conflict、Force 上書き）を網羅しており品質が高い
- ハンドラーテストも認証・認可・正常系・競合の各シナリオをカバーしている
- 国際化が日本語・英語の両方で適切に追加されている

**軽微な指摘**:

- テスト 1 箇所で `t.Parallel()` が欠けている（修正は任意）
