# コードレビュー: page-edit-8-3 (3回目)

## レビュー情報

| 項目                       | 内容                                                 |
| -------------------------- | ---------------------------------------------------- |
| レビュー日                 | 2026-02-27                                           |
| 対象ブランチ               | page-edit-8-3                                        |
| ベースブランチ             | page-edit                                            |
| 作業計画書（指定があれば） | docs/plans/1_doing/page-edit-go-migration.md         |
| 変更ファイル数             | 11 ファイル（うちドキュメント・自動生成 4 ファイル） |
| 変更行数（実装）           | +493 / -0 行                                         |
| 変更行数（テスト）         | +0 / -0 行                                           |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版の開発ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/web/markdown-editor/file-upload-handler.ts`
- [x] `go/web/markdown-editor/direct-upload.ts`
- [x] `go/web/markdown-editor/upload-placeholder.ts`
- [x] `go/web/markdown-editor/markdown-editor.ts`
- [x] `go/internal/templates/pages/page/edit.templ`

### テストファイル

（テストなし — タスク 8-3 はコア機能のファイル追加のみで、テストは 8-4 で E2E テストとして実装予定）

### 設定・その他

- [x] `go/package.json`
- [x] `go/pnpm-lock.yaml`
- [x] `go/internal/templates/pages/page/edit_templ.go`（自動生成）
- [x] `docs/plans/1_doing/page-edit-go-migration.md`
- [x] `docs/reviews/done/202602/page-edit-8-3-001.md`
- [x] `docs/reviews/done/202602/page-edit-8-3-002.md`

## ファイルごとのレビュー結果

### `go/web/markdown-editor/file-upload-handler.ts`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - XSS対策
- [@go/CLAUDE.md#既存コードとの一貫性](/workspace/go/CLAUDE.md) - Rails版との一貫性
- 作業計画書 タスク 8-3 の仕様

**問題点・改善提案**:

- **[Rails版との差異] `spaceIdentifier` が `EditorConfig` に追加されたが `createEditor` 内で未使用**

  `markdown-editor.ts` の `EditorConfig` に `spaceIdentifier` が追加され、data属性から読み込まれてconfigに渡されていますが、`createEditor` 関数内では使用されていません。8-4で `FileUploadHandler` のインスタンス化時に使用する想定であるため、現時点では問題ありません。ただし、`spaceIdentifier` が config に含まれている意図が現在のコードからは読み取りにくいです。

  タスク 8-4 の統合で使用されることを確認するだけで十分ですが、念のため確認します。

  **対応方針**:
  - [x] 8-4 で使用予定のため現状のままで良い
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

タスク 8-3 の要件（`file-upload-handler.ts`、`direct-upload.ts`、`upload-placeholder.ts` の3ファイル追加）は正しく実装されています。

**良かった点**:

1. **Rails版との高い一貫性**: ファイルバリデーション（サイズ制限、許可MIMEタイプ、画像寸法制限）、プレースホルダーの挿入・置換・削除ロジック、プリサインフロー、DirectUploadのXHRラッパーがRails版と同等の仕様で再実装されています
2. **前回レビューの対応が反映済み**: `escapeHtmlAttr` / `escapeMarkdownLinkText` によるXSS対策（1回目レビュー）、`validateFile` で画像サイズの取得を統合して二重取得を解消（2回目レビュー）が正しく反映されています
3. **Go版に適した適切な変更**: Rails版が `@rails/request.js` で行っていたHTTPリクエストを `fetch` + 明示的 `X-CSRF-Token` ヘッダーに置換し、MD5チェックサム計算を `@rails/activestorage` の `FileChecksum` から `spark-md5` に置換するなど、Go版フロントエンドに適した依存関係が選択されています
4. **定数のモジュールレベル抽出**: `FILE_SIZE_LIMITS`, `ALLOWED_FILE_TYPES`, `MAX_IMAGE_DIMENSION` がモジュールレベルに抽出され、Rails版（メソッド内で定義）よりも可読性が向上しています
5. **テンプレートへの `data-markdown-editor-space-identifier` の追加**: 8-4でのFileUploadHandler統合に向けた準備が適切に行われています

**注意点**:

- テストはタスク 8-4 で E2E テストとして追加予定のため、現時点ではテストなしで問題ありません
- `markdown-editor.ts` の `EditorConfig.spaceIdentifier` は現時点では未使用ですが、8-4 での統合で使用予定です
