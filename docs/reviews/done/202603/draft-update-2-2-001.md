# コードレビュー: draft-update-2-2

## レビュー情報

| 項目                       | 内容                                  |
| -------------------------- | ------------------------------------- |
| レビュー日                 | 2026-03-05                            |
| 対象ブランチ               | draft-update-2-2                      |
| ベースブランチ             | draft-update-2-1                      |
| 作業計画書（指定があれば） | docs/plans/1_doing/draft-update.md    |
| 変更ファイル数             | 13 ファイル                           |
| 変更行数（実装）           | +399 / -69 行（自動生成ファイル含む） |
| 変更行数（テスト）         | +170 / -2 行                          |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/handler/draft_page/create.go`
- [x] `go/internal/handler/draft_page/handler.go`
- [x] `go/internal/handler/page/edit.go`
- [x] `go/internal/handler/page/update.go`
- [x] `go/cmd/server/main.go`
- [x] `go/internal/templates/pages/page/edit.templ`
- [x] `go/internal/templates/pages/page/edit_templ.go`（自動生成）

### テストファイル

- [x] `go/internal/handler/draft_page/create_test.go`
- [x] `go/internal/handler/draft_page/update_test.go`

### 設定・その他

- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`
- [x] `docs/plans/1_doing/draft-update.md`
- [x] `docs/plans/1_doing/edit-suggestion.md`

## ファイルごとのレビュー結果

### `go/internal/handler/page/update.go`

**ステータス**: 対応済み

**チェックしたガイドライン**:

- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - テンプレートデータ構造体と ViewModel の関係

**問題点・改善提案**:

- **[@go/docs/templ-guide.md#テンプレートデータ構造体とViewModelの関係]**: `renderEditWithErrors` で `ManualSaveURL` を設定しているが、`LinkList` と `BacklinkList` が設定されていない。エラー再表示時にリンク一覧とバックリンク一覧が空になる。これは本PRで導入された問題ではなく既存の挙動だが、`ManualSaveURL` の追加に伴い確認したため記録する。

  ```go
  // update.go:209-216 - LinkList, BacklinkList が未設定
  content := pagepages.Edit(pagepages.EditPageData{
      CSRFToken:     csrfToken,
      FormErrors:    formErrors,
      Page:          pageVM,
      Space:         spaceVM,
      Topic:         topicVM,
      ManualSaveURL: string(templates.PageDraftPagePath(spaceIdentifier.String(), int32(pg.Number))),
  })
  ```

  **修正案**: 本PRのスコープ外のため、今回は対応不要。

  **対応方針**:
  - [x] 本PRで対応する（LinkList, BacklinkListも設定する）
  - [ ] 別タスクとして対応する
  - [ ] 現状のまま（エラー再表示時はリンク一覧が不要と判断）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計との整合性チェック

作業計画書 タスク 2-2 の要件と実装を照合:

- [x] `internal/handler/draft_page/create.go` に POST ハンドラーを追加（DraftPageRevision の作成）
- [x] `internal/handler/draft_page/handler.go` に Usecase の依存を追加
- [x] `internal/templates/pages/page/edit.templ` に「下書き保存」ボタンを追加
- [x] `internal/i18n/locales/ja.toml`, `en.toml` に翻訳キーを追加
- [x] ルーティング追加（`cmd/server/main.go`）
- [x] 現時点ではその場に留まる動作（`w.WriteHeader(http.StatusNoContent)` で実装）

設計との乖離はありません。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 2-2「編集画面に下書き保存ボタンを追加」の実装として適切です。

良い点:

- `create.go` のハンドラー実装が既存の `update.go` や `show.go` のパターンと一貫している（認証チェック、URLパラメータ取得、スペース/メンバー/ページ取得、TopicPolicy による認可チェック）
- Datastar の `@post` を使った AJAX リクエストで CSRF トークンをヘッダーに含めており、セキュリティガイドラインに準拠
- テストが異常系（未ログイン、不正なページ番号、スペース未存在、下書き未存在）を適切にカバー
- 翻訳キーの命名が既存パターン（`page_edit_*`）と一貫
- 作業計画書の仕様通り、現時点では `204 No Content` で応答し、リダイレクト先の変更はタスク 3-3 に委ねている
