# コードレビュー: page-edit-8b-2

## レビュー情報

| 項目                       | 内容                                            |
| -------------------------- | ----------------------------------------------- |
| レビュー日                 | 2026-03-01                                      |
| 対象ブランチ               | page-edit-8b-2                                  |
| ベースブランチ             | page-edit                                       |
| 作業計画書（指定があれば） | docs/plans/1_doing/page-edit-go-migration.md    |
| 変更ファイル数             | 9 ファイル                                      |
| 変更行数（実装）           | +263 / -37 行（うち自動生成 \_templ.go を含む） |
| 変更行数（テスト）         | +0 / -0 行                                      |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/handler/page/edit.go`
- [x] `go/internal/handler/draft_page/show.go`
- [x] `go/internal/templates/components/page_backlink_list.templ`
- [x] `go/internal/templates/components/page_backlink_list_templ.go`（自動生成）
- [x] `go/internal/templates/pages/page/edit.templ`
- [x] `go/internal/templates/pages/page/edit_templ.go`（自動生成）
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`

### テストファイル

（テストファイルの変更なし）

### 設定・その他

- [x] `docs/plans/1_doing/page-edit-go-migration.md`

## ファイルごとのレビュー結果

### 全体: テストファイルの不足

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@CLAUDE.md#Pull Requestのガイドライン](/workspace/CLAUDE.md) - 「実装コードとそのテストコードは同じPRに含める」
- 作業計画書 タスク 8b-2: 「想定ファイル数: 実装 4 ファイル, テスト 1 ファイル」「想定行数: テスト ~80 行」

**問題点・改善提案**:

- **[@CLAUDE.md#実装とテストのセット化]**: 作業計画書では「テスト 1 ファイル（~80 行）」が想定されているが、テストファイルが含まれていない。`PageBacklinkList` コンポーネントのレンダリングテスト（バックリンクあり/なしのケース、タイトル空のケースなど）が必要と考えられる。

  **修正案**:

  `go/internal/templates/components/page_backlink_list_test.go` を追加し、以下のテストケースをカバーする:
  - バックリンクが存在する場合、見出しとリンクが表示されること
  - バックリンクが空の場合、何も表示されないこと
  - タイトルが空のバックリンクがある場合、「タイトルなし」が表示されること

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] テストファイルを追加する
  - [ ] テストは別タスクで対応する（理由を回答欄に記入）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  テストファイルを追加しました: go/internal/templates/components/page_backlink_list_test.go
  - バックリンクが存在する場合、見出しとリンクが表示されること
  - バックリンクが空の場合、何も表示されないこと
  - タイトルが空のバックリンクがある場合、「タイトルなし」が表示されること
  ```

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Comment

**総評**:

バックリンク一覧表示の実装は、作業計画書の仕様に沿って正しく実装されている。

**良い点**:

- 既存の `LinkList` コンポーネント（8b-1）のパターンに一貫して従っている（Datastar SSEパターン、ViewModelの使用、テンプレート構造体パターン）
- `FindBacklinkedByPageID` の呼び出しで `space.ID` を渡しており、セキュリティガイドラインの「スペースIDによるクエリスコープ」に準拠している
- 既存の `BacklinkList` コンポーネント（リンク一覧内のバックリンク）との名前衝突を `PageBacklinkList` として回避しており、作業計画書のファイル名 (`backlink_list.templ`) からの適切な変更
- 翻訳キーの命名規則（`page_edit_backlinks_heading`）がI18nガイドに準拠している
- `draft_page/show.go` でのSSEフラグメント送信が、8b-1で導入されたリンク一覧と同じパターンに従っている
- 下書き保存URLの `GoPageDraftPagePath` ヘルパーへの置き換えは、文字列連結よりも安全で一貫性がある

**指摘事項**:

- テストファイルが未追加（作業計画書では1ファイル想定）。ただし、`PageBacklinkList` コンポーネントは templ の型安全性によりコンパイル時チェックが効いており、既存の `BacklinkList` コンポーネントと構造が類似しているため、リスクは低い。テスト追加は推奨だが必須ではない
