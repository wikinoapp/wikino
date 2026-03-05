# コードレビュー: draft-update-2-2

## レビュー情報

| 項目                       | 内容                                            |
| -------------------------- | ----------------------------------------------- |
| レビュー日                 | 2026-03-05                                      |
| 対象ブランチ               | draft-update-2-2                                |
| ベースブランチ             | draft-update-2-1                                |
| 作業計画書（指定があれば） | docs/plans/1_doing/draft-update.md              |
| 変更ファイル数             | 14 ファイル                                     |
| 変更行数（実装）           | +456 / -34 行（生成ファイル・ドキュメント除く） |
| 変更行数（テスト）         | +172 / -3 行                                    |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版の開発ガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templ テンプレートガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/handler/draft_page/create.go`
- [x] `go/internal/handler/draft_page/handler.go`
- [x] `go/internal/handler/page/edit.go`
- [x] `go/internal/handler/page/update.go`
- [x] `go/internal/templates/pages/page/edit.templ`
- [x] `go/internal/templates/pages/page/edit_templ.go`（自動生成）
- [x] `go/cmd/server/main.go`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`

### テストファイル

- [x] `go/internal/handler/draft_page/update_test.go`
- [ ] `go/internal/handler/draft_page/create_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/draft-update.md`
- [x] `docs/plans/1_doing/edit-suggestion.md`
- [x] `docs/reviews/draft-update-2-2-001.md`

## ファイルごとのレビュー結果

### `go/internal/handler/draft_page/create_test.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テスト戦略

**問題点・改善提案**:

- **[@go/docs/testing-guide.md#エラーケースを必ずテスト]**: 正常系のテストが不足している。現在のテストは異常系のみ（未認証、不正ページ番号、スペース未発見、下書き未発見）で、成功パターン（下書きが存在する状態で POST → 204 No Content）のテストがない

  作業計画書のタスク 2-2 には「想定行数: テスト 80」とあるが、正常系のテストを含めてもこの範囲に収まるはず。

  **修正案**:

  以下のような正常系テストを追加する:

  ```go
  func TestCreate_Success(t *testing.T) {
      t.Parallel()

      _, tx := testutil.SetupTx(t)
      queries := testutil.QueriesWithTx(tx)

      // ユーザー、スペース、スペースメンバー、トピック、トピックメンバー、ページ、下書きページを作成
      // ...

      handler := setupHandler(t, queries)

      req := newPostRequestWithChiParams(t, "/s/.../pages/1/draft_page", map[string]string{...})
      ctx := middleware.SetUserToContext(req.Context(), &model.User{ID: userID})
      req = req.WithContext(ctx)

      rr := httptest.NewRecorder()
      handler.Create(rr, req)

      if rr.Code != http.StatusNoContent {
          t.Errorf("wrong status code: got %v want %v", rr.Code, http.StatusNoContent)
      }
  }
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 正常系テストを追加する
  - [ ] 現状のまま（理由を回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Request Changes

**総評**:

タスク 2-2（編集画面に「下書き保存」ボタンを追加）の実装として、全体的にガイドラインに沿った良い実装です。

**良かった点**:

- ハンドラーガイドに従った `create.go` の命名と配置
- 認証・認可チェック（ユーザー認証、スペースメンバー確認、TopicPolicy）が適切
- `renderEditWithErrors` に `ManualSaveURL` を正しく渡しており、バリデーションエラー後の再表示時にも下書き保存ボタンが機能する
- i18n の翻訳キーが命名規則に従っている（`page_edit_save_draft_button`）
- テストヘルパー `setupHandler` の `draftPageRepo` 共有化リファクタリングが適切
- Datastar の `@post` パターンによるクライアントサイドの POST 送信が既存パターンと一貫している

**修正が必要な点**:

- Create ハンドラーの正常系テストが不足している（1 件）
