# コードレビュー: go-topic-1-2

## レビュー情報

| 項目                       | 内容                                                |
| -------------------------- | --------------------------------------------------- |
| レビュー日                 | 2026-03-08                                          |
| 対象ブランチ               | go-topic-1-2                                        |
| ベースブランチ             | go-topic                                            |
| 作業計画書（指定があれば） | docs/plans/1_doing/topic-show-go-migration.md       |
| 変更ファイル数             | 4 ファイル（実装 1 + テスト 1 + 設定 1 + 計画書 1） |
| 変更行数（実装）           | +13 / -5 行                                         |
| 変更行数（テスト）         | +92 / -0 行                                         |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド（ViewModel 設計）
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/viewmodel/page.go`

### テストファイル

- [x] `go/internal/viewmodel/page_test.go`

### 設定・その他

- [x] `.claude/commands/review.md`
- [x] `docs/plans/1_doing/topic-show-go-migration.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。すべてのファイルがガイドラインに準拠しています。

## 設計との整合性チェック

作業計画書タスク 1-2 の要件:

- [x] `NewCardLinkPage` コンストラクタでアイキャッチ画像 URL（`CardImageURL`）を設定する処理を追加
- [x] `CardImageURL` フィールドとテンプレート側の表示ロジックは既に存在するため、コンストラクタの修正のみ
- [x] 既存テストの更新（新規テストとして追加）

すべての要件が実装されています。

### 確認した点

- `/attachments/%s` の URL パターンは Rails 版のルーティング（`config/routes.rb`）と一致している
- `FeaturedImageAttachmentID` はドメイン ID 型 `*AttachmentID` が正しく使用されている
- テストはテーブル駆動テストパターンに従い、アイキャッチ画像あり/なし/タイトル nil の 3 パターンを網羅している
- `t.Parallel()` が適切に使用されている

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 1-2 の要件通りに、`NewCardLinkPage` コンストラクタへのアイキャッチ画像 URL 設定処理が正しく実装されています。変更は最小限に留められており、既存の `CardImageURL` フィールドに対して `FeaturedImageAttachmentID` から URL を生成するロジックのみが追加されています。テストも網羅的で、ガイドラインに準拠した実装です。
