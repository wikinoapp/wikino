# コードレビュー: page-edit-8-3

## レビュー情報

| 項目                       | 内容                                                   |
| -------------------------- | ------------------------------------------------------ |
| レビュー日                 | 2026-02-27                                             |
| 対象ブランチ               | page-edit-8-3                                          |
| ベースブランチ             | page-edit                                              |
| 作業計画書（指定があれば） | docs/plans/1_doing/page-edit-go-migration.md           |
| 変更ファイル数             | 10 ファイル                                            |
| 変更行数（実装）           | +500 / -1 行（TS 3 ファイル新規, templ 1行, 設定 2行） |
| 変更行数（テスト）         | +0 / -0 行                                             |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md#セキュリティガイドライン](/workspace/go/CLAUDE.md) - XSS対策
- [@go/CLAUDE.md#templテンプレート](/workspace/go/CLAUDE.md) - templテンプレートガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/web/markdown-editor/direct-upload.ts`
- [x] `go/web/markdown-editor/file-upload-handler.ts`
- [x] `go/web/markdown-editor/upload-placeholder.ts`
- [x] `go/web/markdown-editor/markdown-editor.ts`
- [x] `go/internal/templates/pages/page/edit.templ`

### テストファイル

（なし — タスク 8-3 はコア機能の追加のみ。テストはタスク 8-4 で E2E テストとして追加予定）

### 設定・その他

- [x] `go/package.json`
- [x] `go/pnpm-lock.yaml`
- [x] `go/internal/templates/pages/page/edit_templ.go`（自動生成ファイル）
- [x] `docs/plans/1_doing/page-edit-go-migration.md`（タスク完了チェック更新）
- [x] `docs/reviews/done/202602/page-edit-8-3-001.md`（前回レビュー）

## ファイルごとのレビュー結果

問題のあるファイルのみ記載します。問題がないファイルは「変更ファイル一覧」のチェックボックスにチェックを入れています。

（全ファイルに問題は見つかりませんでした）

## 設計改善の提案

### `go/web/markdown-editor/file-upload-handler.ts`: 画像サイズの二重取得

**ステータス**: 要確認

**現状**:

画像ファイルのアップロード時に `getImageDimensions` が2回呼ばれている:

1. `validateFile` 内（225行目）: サイズ上限チェック
2. `handleFileUpload` 内（114行目）: img タグの width 属性用

```typescript
// handleFileUpload内（114行目）
const dimensions = await this.getImageDimensions(file);

// validateFile内（225行目）
const dimensions = await this.getImageDimensions(file);
```

**提案**:

`validateFile` でバリデーション済みの dimensions を返し、`handleFileUpload` で再利用する。

```typescript
// validateFileの戻り値にdimensionsを含める
private async validateFile(file: File): Promise<{ width?: number; height?: number }> {
  // ... バリデーション ...
  if (file.type.startsWith("image/")) {
    const dimensions = await this.getImageDimensions(file);
    // サイズチェック
    return dimensions;
  }
  return {};
}

// handleFileUpload内で再利用
const dimensions = await this.validateFile(file);
```

**メリット**:

- 画像のObjectURL作成・読み込みが1回で済む
- 大きな画像ファイルでのパフォーマンス改善

**トレードオフ**:

- `validateFile` の戻り値が複雑になる（バリデーション結果 + dimensions）
- Rails版との一貫性が下がる（Rails版も同様に2回呼んでいる）
- 実質的な影響は小さい（ブラウザキャッシュが効く可能性あり）

**対応方針**:

- [x] 提案通り変更する
- [ ] 現状のまま（Rails版との一貫性を優先）
- [ ] その他（下の回答欄に記入）

**回答**:

```
（ここに回答を記入）
```

## 総合評価

**評価**: Approve

**総評**:

タスク 8-3（ファイルアップロードのコア機能の追加）の実装として、作業計画書の仕様通りに正しく実装されている。

**良かった点**:

- **Rails版との一貫性**: ファイルサイズ制限、許可MIMEタイプ、プレースホルダー形式、Markdown出力形式がすべてRails版と一致している
- **XSS対策**: `escapeHtmlAttr` と `escapeMarkdownLinkText` によるエスケープ処理が適切に実装されている（前回レビュー指摘を反映済み）
- **CSRF対策**: Presignリクエストで `X-CSRF-Token` ヘッダーを正しく送信している
- **設計の分離**: `direct-upload.ts`（S3アップロード）、`file-upload-handler.ts`（オーケストレーション）、`upload-placeholder.ts`（プレースホルダー管理）の責務が明確に分かれている
- **エラーハンドリング**: アップロード失敗時のプレースホルダー削除とトースト通知が適切

**作業計画書との整合性**:

- `file-upload-handler.ts` ✅
- `direct-upload.ts` ✅
- `upload-placeholder.ts` ✅
- Presignエンドポイントは引き続きRails版にプロキシ ✅
- `spaceIdentifier` のテンプレート→エディタへの受け渡し ✅（タスク8-4/8-5で必要な準備）
- `spark-md5` 依存の追加 ✅
