# コードレビュー: page-edit-8-3

## レビュー情報

| 項目                       | 内容                                              |
| -------------------------- | ------------------------------------------------- |
| レビュー日                 | 2026-02-27                                        |
| 対象ブランチ               | page-edit-8-3                                     |
| ベースブランチ             | page-edit                                         |
| 作業計画書（指定があれば） | docs/plans/1_doing/page-edit-go-migration.md      |
| 変更ファイル数             | 9 ファイル                                        |
| 変更行数（実装）           | +486 / -20 行（TS実装3ファイル + templ + editor） |
| 変更行数（テスト）         | +0 / -0 行                                        |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/web/markdown-editor/file-upload-handler.ts`
- [x] `go/web/markdown-editor/upload-placeholder.ts`
- [x] `go/web/markdown-editor/direct-upload.ts`
- [x] `go/web/markdown-editor/markdown-editor.ts`
- [x] `go/internal/templates/pages/page/edit.templ`
- [x] `go/internal/templates/pages/page/edit_templ.go`（自動生成）

### 設定・その他

- [x] `go/package.json`
- [x] `go/pnpm-lock.yaml`
- [x] `docs/plans/1_doing/page-edit-go-migration.md`

## ファイルごとのレビュー結果

### `go/web/markdown-editor/upload-placeholder.ts`

**ステータス**: 対応済み

**チェックしたガイドライン**:

- [@go/docs/security-guide.md#XSS対策](/workspace/go/docs/security-guide.md) - XSS対策

**問題点・改善提案**:

- **[@go/docs/security-guide.md#XSS対策]**: `replacePlaceholderWithUrl`関数でHTML文字列を動的に構築しているが、`altText`（ファイル名由来）がエスケープされていない

  ```typescript
  // 問題のあるコード（upload-placeholder.ts:57-59）
  newText = `<img width="${width}" alt="${altText || fileName}" src="${url}">`;
  ```

  `altText`はユーザーがアップロードしたファイルのファイル名から来ており、`"`, `<`, `>` などの文字を含む可能性がある。この文字列はCodeMirrorのドキュメントに挿入されるため、Markdownとして解釈される文脈では直接的なXSSリスクは低いが、`alt`属性の`"`でHTML属性が壊れる可能性がある。

  例: ファイル名 `test" onload="alert(1)` の場合:

  ```html
  <img width="800" alt="test" onload="alert(1)" src="/attachments/xxx" />
  ```

  **修正案**:

  HTMLの属性値に挿入する文字列をエスケープするヘルパー関数を追加する:

  ```typescript
  function escapeHtmlAttr(str: string): string {
    return str.replace(/&/g, "&amp;").replace(/"/g, "&quot;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
  }

  // 使用箇所
  const safeAlt = escapeHtmlAttr(altText || fileName);
  newText = `<img width="${width}" alt="${safeAlt}" src="${url}">`;
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 修正案の通りエスケープ処理を追加する
  - [ ] Rails版と同じ挙動にする（Rails版にエスケープがあるか確認した上で判断）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

- **[@go/docs/security-guide.md#XSS対策]**: `url`もエスケープされていない

  ```typescript
  // 問題のあるコード（upload-placeholder.ts:57-62）
  newText = `<img width="${width}" alt="${altText || fileName}" src="${url}">`;
  // ...
  newText = `[${altText || fileName}](${url})`;
  ```

  `url`は`/attachments/${presignData.attachmentId}`の形式でサーバーレスポンスから構築されるため、通常は安全だが、防御的プログラミングの観点からエスケープが望ましい。`altText`のエスケープと合わせて対応すると良い。Markdown形式のリンク`[text](url)`でも`text`部分に`]`が含まれると壊れる可能性がある。

  **修正案**:

  ```typescript
  // Markdown用のエスケープも追加
  function escapeMarkdownLinkText(str: string): string {
    return str.replace(/\[/g, "\\[").replace(/\]/g, "\\]");
  }

  const safeText = escapeMarkdownLinkText(altText || fileName);
  newText = `[${safeText}](${url})`;
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 修正案の通りMarkdownエスケープも追加する
  - [ ] HTMLエスケープのみ対応し、Markdownは対応しない
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

タスク8-3の実装は作業計画書の要件を適切にカバーしている。3つの新規TSファイル（`file-upload-handler.ts`、`direct-upload.ts`、`upload-placeholder.ts`）がそれぞれ明確な責務を持ち、作業計画書で定義されたファイル構成と一致している。

**良かった点**:

- ファイルバリデーション（サイズ制限、MIMEタイプ、画像サイズ）が作業計画書の仕様と完全に一致
- チャンク分割によるMD5チェックサム計算で大きなファイルにも対応
- テスト環境とプロダクション環境の分岐が適切に実装されている
- プレースホルダーの位置追跡が、エディタの内容変更時にフォールバック検索（`indexOf`）を行う堅牢な実装
- `DirectUpload`クラスの`cancel`メソッドによるアップロード中断のサポート
- Rails版のファイルバリデーションルール（サイズ制限、MIMEタイプ、画像サイズ上限）との整合性が取れている

**指摘事項**:

- `upload-placeholder.ts`でHTML/Markdown出力を構築する際のエスケープ処理が不足している。セキュリティ上の実質的なリスクは低い（CodeMirrorのテキストドキュメントに挿入されるため、ブラウザによる直接的なHTML解釈は起きない）が、エディタの内容がHTMLとしてレンダリングされる後段の処理（プレビュー等）を考慮すると、入力段階でのエスケープが望ましい

**設計との整合性**: 作業計画書のタスク8-3で定義された3ファイル（`file-upload-handler.ts`、`direct-upload.ts`、`upload-placeholder.ts`）が計画通りに実装されている。テンプレートへの`spaceIdentifier`属性の追加や`markdown-editor.ts`への統合も適切
