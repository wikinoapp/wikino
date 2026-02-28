# コードレビュー: link-list-3-1

## レビュー情報

| 項目                       | 内容                                                 |
| -------------------------- | ---------------------------------------------------- |
| レビュー日                 | 2026-02-28                                           |
| 対象ブランチ               | link-list-3-1                                        |
| ベースブランチ             | page-edit                                            |
| 作業計画書（指定があれば） | docs/plans/1_doing/link-list-alignment.md            |
| 変更ファイル数             | 11 ファイル（自動生成 2 含む）                       |
| 変更行数（実装）           | +134 / -5 行（手動編集分、templ + Go + toml + icon） |
| 変更行数（テスト）         | +約 200 行                                           |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templ テンプレートガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/handler/page/edit.go`
- [x] `go/internal/handler/page_link_list/show.go`
- [x] `go/internal/templates/components/backlink_list.templ`
- [x] `go/internal/templates/components/link_list.templ`
- [x] `go/internal/viewmodel/link_list.go`
- [x] `go/internal/templates/icons_phosphor.go`

### テストファイル

- [x] `go/internal/viewmodel/link_list_test.go`（更新: `PageNumber` フィールドのテスト追加）
- [x] `go/internal/viewmodel/backlink_list_test.go`（新規: `BacklinkList` の生成・ページネーションテスト）
- [x] `go/internal/repository/page_test.go`（更新: `FindBacklinksForPages` のテスト追加）
- [x] `go/internal/handler/page_link_list/show_test.go`（更新: ページネーションパラメータのテスト追加）

### 設定・その他

- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`
- [x] `go/internal/templates/components/backlink_list_templ.go`（自動生成）
- [x] `go/internal/templates/components/link_list_templ.go`（自動生成）
- [x] `docs/plans/1_doing/link-list-alignment.md`

## ファイルごとのレビュー結果

### テストファイルの欠如

**ステータス**: 対応済み

**チェックしたガイドライン**:

- [@CLAUDE.md#Pull Request のガイドライン](/workspace/CLAUDE.md) - 実装とテストのセット化

**対応内容**:

以下のテストを追加した:

1. **`go/internal/viewmodel/backlink_list_test.go`**（新規）: `TestNewBacklinkList`（3 ケース）、`TestNewBacklinkList_WithPagination`
2. **`go/internal/viewmodel/link_list_test.go`**（更新）: `TestNewLinkList_WithPagination` に `PageNumber` フィールドのテスト追加
3. **`go/internal/repository/page_test.go`**（更新）: `TestPageRepository_FindBacklinksForPages`（4 サブテスト: 一括取得、limit 制限、バックリンクなし、空リスト）
4. **`go/internal/handler/page_link_list/show_test.go`**（更新）: `TestShow_正常系_ページネーションパラメータが反映される`

全テスト PASS 確認済み（viewmodel: 32, repository: 187, handler/page_link_list: 9）。

## 設計改善の提案

### `go/internal/handler/page/edit.go` / `go/internal/handler/page_link_list/show.go`: バックリンク取得の N+1 クエリ

**ステータス**: 対応済み

**対応内容**:

- `FindBacklinksForPages` メソッドを `PageRepository` に追加（LATERAL JOIN を使用した一括バックリンク取得）
- SQL クエリ `FindBacklinkedPagesForTargets`（LATERAL JOIN）と `CountBacklinkedPagesForTargets` を追加
- 両ハンドラー（`edit.go`, `show.go`）のループ処理を一括取得に置き換え
- クエリ数: 32 → 4 に削減
- テスト: `TestPageRepository_FindBacklinksForPages`（4 サブテスト）で検証済み

### `go/internal/handler/page/edit.go` / `go/internal/handler/page_link_list/show.go`: リンク上限定数の重複

**ステータス**: 対応済み

**対応内容**:

- `viewmodel.LinkLimit`（15）と `viewmodel.BacklinkLimit`（14）を `viewmodel/link_list.go` に共有定数として定義
- 両ハンドラーのローカル定数を共有定数の参照に置き換え

## 総合評価

**評価**: Approve

**総評**:

すべての指摘事項に対応済み:

1. **テストの追加**: viewmodel（backlink_list_test.go 新規、link_list_test.go 更新）、repository（page_test.go 更新）、handler（show_test.go 更新）の 4 ファイルにテストを追加。全テスト PASS 確認済み
2. **N+1 クエリの解消**: LATERAL JOIN を使用した一括バックリンク取得（`FindBacklinksForPages`）を実装。クエリ数 32 → 4 に削減
3. **共有定数の定義**: `viewmodel.LinkLimit` / `viewmodel.BacklinkLimit` として定義し、重複を排除

実装コードの品質は高く、作業計画書の設計に忠実:

- **セキュリティ**: クエリにはすべて `space.ID` を含めており、スペース境界を正しく維持
- **アーキテクチャ**: Handler → Repository → ViewModel の依存方向が正しく守られている
- **国際化**: ユーザー向けメッセージはすべて `templates.T(ctx, ...)` で翻訳されている
- **テスト**: 実装に対応するテストが網羅的に追加されている
