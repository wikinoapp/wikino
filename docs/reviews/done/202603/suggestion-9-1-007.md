# コードレビュー: suggestion-9-1 (タスク 9-1)

## レビュー情報

| 項目                       | 内容                                          |
| -------------------------- | --------------------------------------------- |
| レビュー日                 | 2026-03-19                                    |
| 対象ブランチ               | suggestion-9-1                                |
| ベースブランチ             | suggestion-8a-1                               |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md              |
| 変更ファイル数             | 20 ファイル（レビュー・計画書を除く）         |
| 変更行数（実装）           | +686 / -35 行（自動生成ファイル・テスト除く） |
| 変更行数（テスト）         | +640 行                                       |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [ ] `go/internal/handler/suggestion_change/index.go`
- [x] `go/internal/handler/suggestion_page_edit/handler.go`
- [ ] `go/internal/handler/suggestion_page_edit/show.go`
- [x] `go/internal/handler/suggestion_page_edit/create.go`
- [x] `go/internal/usecase/start_suggestion_page_edit.go`
- [x] `go/internal/viewmodel/suggestion.go`
- [x] `go/internal/templates/pages/suggestion_page_edit/show.templ`
- [x] `go/internal/templates/pages/suggestion_change/index.templ`
- [x] `go/internal/templates/pages/suggestion_change/index_templ.go`（自動生成）
- [x] `go/internal/templates/pages/suggestion_page_edit/show_templ.go`（自動生成）
- [x] `go/internal/templates/path.go`
- [x] `go/internal/templates/page_name.go`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`

### テストファイル

- [x] `go/internal/handler/suggestion_page_edit/create_test.go`
- [x] `go/internal/handler/suggestion_page_edit/main_test.go`
- [x] `go/internal/usecase/start_suggestion_page_edit_test.go`

### 設定・ドキュメント

- [x] `go/docs/security-guide.md`
- [x] `go/docs/testing-guide.md`

## ファイルごとのレビュー結果

### `go/internal/handler/suggestion_change/index.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティ

**問題点・改善提案**:

- **[@go/docs/security-guide.md#CSRF対策]**: 変更差分画面（GET）にCSRFトークンを渡してフォームに埋め込んでいるが、未ログインユーザーの場合のCSRFトークンの挙動を確認すべき。`suggestion_change/index.go` は未ログインユーザーも閲覧可能（`user` が nil の場合がある）だが、`middleware.GetCSRFTokenFromContext(ctx)` は未ログインでも正常に動作するか？ `CanEditSuggestionPages` が false になり「編集する」ボタンは非表示になるため実害はないが、不要なCSRFトークン取得処理が走る。

  **修正案**:

  ```go
  // CSRFトークンの取得をCanEditSuggestionPagesの判定後に移動し、必要な場合のみ取得
  canEditSuggestionPages := output.SpaceMember != nil && output.Suggestion.Status == model.SuggestionStatusOpen
  var csrfToken string
  if canEditSuggestionPages {
      csrfToken = middleware.GetCSRFTokenFromContext(ctx)
  }
  ```

  **対応方針**:
  - [x] 修正案の通り条件付きで取得する
  - [ ] 現状のままで問題なし（実害がないため）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/handler/suggestion_page_edit/show.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド

**問題点・改善提案**:

- **[@go/docs/architecture-guide.md#レイヤー間の依存関係]**: Show ハンドラーでは `detailOutput.SuggestionPages` をループしてタイトルを取得している（77-89行目）。このロジック自体は軽量だが、`getSuggestionDetailUsecase` は編集提案詳細画面全体のデータ（コメント、ユーザーマップ等）を取得するため、確認画面に不要なデータも取得している。現時点では実害は小さいが、将来的にデータ量が増えた場合にパフォーマンスの問題になる可能性がある。

  確認画面に必要なデータは「スペース、トピック、スペースメンバー情報、対象SuggestionPageのタイトル」のみ。専用の読み取りUseCaseがあればデータ取得量を最小化できるが、既存のUseCaseの再利用はコードの簡潔さという点でメリットもある。

  **修正案**: 現時点ではこのままでも許容。データ量が問題になった場合に専用のUseCaseを切り出す。

  **対応方針**:
  - [x] 現状のまま（許容範囲内）
  - [ ] 専用の読み取りUseCaseを作成する
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

タスク 9-1（編集提案ページ編集開始のUseCase・ハンドラー）の実装として、作業計画書の設計に準拠した適切な実装になっている。

**良かった点**:

- 3層アーキテクチャに従った適切なレイヤー分割（Handler → UseCase → Repository）
- 通常編集の下書きとの衝突検出・確認画面・Force上書きの3パターンが作業計画書通りに実装されている
- UseCaseのステータスパターン（Redirect/Conflict）による分岐が明確で読みやすい
- テストが正常系・異常系の主要パターンを網羅（未ログイン、権限なし、下書きなし、衝突、Force上書き）
- 翻訳ファイルに日英両方のメッセージが適切に追加されている
- `url.PathEscape` を使用した `SuggestionPageEditShowPath` はXSS対策として適切
- ドキュメント（security-guide.md, testing-guide.md）の改善も含まれている

**軽微な確認事項**:

- `suggestion_change/index.go` の未ログイン時のCSRFトークン取得は実害なしだが、不要な処理ではある（上記参照）
- `show.go` の `getSuggestionDetailUsecase` の再利用は、現時点では許容範囲だがデータ取得量の点で将来的に検討の余地あり
