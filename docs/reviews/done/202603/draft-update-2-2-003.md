# コードレビュー: draft-update-2-2

## レビュー情報

| 項目                       | 内容                                       |
| -------------------------- | ------------------------------------------ |
| レビュー日                 | 2026-03-05                                 |
| 対象ブランチ               | draft-update-2-2                           |
| ベースブランチ             | draft-update-2-1                           |
| 作業計画書（指定があれば） | docs/plans/1_doing/draft-update.md         |
| 変更ファイル数             | 15 ファイル（うちドキュメント 4 ファイル） |
| 変更行数（実装）           | +314 / -68 行                              |
| 変更行数（テスト）         | +255 / -2 行                               |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/handler/draft_page/create.go`
- [x] `go/internal/handler/draft_page/handler.go`
- [ ] `go/internal/handler/page/edit.go`
- [ ] `go/internal/handler/page/update.go`
- [x] `go/internal/templates/pages/page/edit.templ`
- [x] `go/internal/templates/pages/page/edit_templ.go`（自動生成）
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`
- [x] `go/cmd/server/main.go`

### テストファイル

- [x] `go/internal/handler/draft_page/create_test.go`
- [x] `go/internal/handler/draft_page/update_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/draft-update.md`
- [x] `docs/plans/1_doing/edit-suggestion.md`
- [x] `docs/reviews/draft-update-2-2-001.md`
- [x] `docs/reviews/draft-update-2-2-002.md`

## ファイルごとのレビュー結果

### `go/internal/handler/page/edit.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - テンプレートデータ構造体とViewModelの関係

**問題点・改善提案**:

- **[@go/docs/templ-guide.md#テンプレートデータ構造体とViewModelの関係]**: `ManualSaveURL` を `string` として `EditPageData` に渡しているが、URL の構築をハンドラー側で行っている。このパターン自体は問題ないが、`edit.go` と `update.go` の `renderEditWithErrors` の両方で同じ `templates.PageDraftPagePath(spaceIdentifier.String(), int32(pg.Number))` を呼んでおり、URL構築ロジックが一貫している点は良い。ただし、`string(templates.PageDraftPagePath(...))` という型変換が冗長に見える点を確認したい。`PageDraftPagePath` の戻り値の型は `templ.SafeURL` か `string` か？

  **修正案**:

  `PageDraftPagePath` が `templ.SafeURL` を返す場合、テンプレートデータに `templ.SafeURL` 型で持つか、もしくは現状の `string` 変換のままでも機能的に問題はない。軽微な確認事項。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 現状のまま（`string` 変換で問題ない）
  - [ ] `EditPageData.ManualSaveURL` の型を `templ.SafeURL` に変更する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/handler/page/update.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約

**問題点・改善提案**:

- **コードの重複**: `renderEditWithErrors` メソッドに追加されたリンク一覧・バックリンク一覧の取得ロジック（204〜290行目）が、`edit.go` の `Edit` メソッドのロジック（約120〜202行目）とほぼ完全に重複している。約90行の同一ロジックが2箇所に存在する。

  これは前回のレビュー（draft-update-2-2-002）で対応された「バリデーションエラー時の編集画面再表示にリンク一覧・バックリンク一覧を追加」の結果だと理解しているが、重複量が大きい。

  **修正案**:

  リンク一覧・バックリンク一覧の取得ロジックをプライベートメソッドに抽出する：

  ```go
  type editLinkResult struct {
      linkList     viewmodel.LinkList
      backlinkList viewmodel.BacklinkList
  }

  func (h *Handler) fetchEditLinks(ctx context.Context, pg *model.Page, spaceMember *model.SpaceMember, space *model.Space, spaceIdentifier model.SpaceIdentifier) (*editLinkResult, error) {
      // 共通のリンク一覧・バックリンク一覧取得ロジック
  }
  ```

  ただし、YAGNI原則と「過度な抽象化を避ける」方針に従い、現時点ではこの重複を許容する判断もあり得る。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] プライベートメソッドに抽出して重複を解消する
  - [ ] 現状のまま（重複を許容する。理由を回答欄に記入）
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

タスク 2-2（編集画面に「下書き保存」ボタンを追加）の実装として、作業計画書の要件を満たしている。

**良かった点**:

- `create.go` のハンドラー実装が既存の `update.go` や他のハンドラーと一貫したパターンに従っている
- 認証・認可チェック（ユーザー認証、スペースメンバー確認、TopicPolicy）が適切に行われている
- テストが正常系・異常系（未ログイン、不正なページ番号、スペース不存在、下書き不存在）を網羅している
- i18n対応が日本語・英語の両方で適切に行われている
- DatastarのSSE経由でPOSTリクエストを送る実装が適切
- CSRF対策としてヘッダーでCSRFトークンを渡す実装が適切

**確認事項**:

- `update.go` の `renderEditWithErrors` と `edit.go` の `Edit` でリンク一覧取得ロジックが重複しているが、これは軽微であり対応は任意
- `ManualSaveURL` の型（`string` vs `templ.SafeURL`）は軽微な確認事項
