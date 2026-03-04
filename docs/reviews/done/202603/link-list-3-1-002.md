# コードレビュー: link-list-3-1

## レビュー情報

| 項目                       | 内容                                       |
| -------------------------- | ------------------------------------------ |
| レビュー日                 | 2026-02-28                                 |
| 対象ブランチ               | link-list-3-1                              |
| ベースブランチ             | page-edit                                  |
| 作業計画書（指定があれば） | docs/plans/1_doing/link-list-alignment.md  |
| 変更ファイル数             | 19 ファイル（自動生成 2、レビュー 1 含む） |
| 変更行数（実装）           | +421 / -27 行                              |
| 変更行数（テスト）         | +354 行                                    |

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

- [x] `go/db/queries/pages.sql`
- [x] `go/internal/query/pages.sql.go`（自動生成）
- [x] `go/internal/repository/page.go`
- [x] `go/internal/viewmodel/link_list.go`
- [x] `go/internal/handler/page/edit.go`
- [x] `go/internal/handler/page_link_list/show.go`
- [x] `go/internal/templates/components/link_list.templ`
- [x] `go/internal/templates/components/backlink_list.templ`
- [x] `go/internal/templates/components/backlink_list_templ.go`（自動生成）
- [x] `go/internal/templates/components/link_list_templ.go`（自動生成）
- [x] `go/internal/templates/icons_phosphor.go`

### テストファイル

- [x] `go/internal/viewmodel/link_list_test.go`
- [x] `go/internal/viewmodel/backlink_list_test.go`
- [x] `go/internal/repository/page_test.go`
- [x] `go/internal/handler/page_link_list/show_test.go`

### 設定・その他

- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`
- [x] `docs/plans/1_doing/link-list-alignment.md`
- [x] `docs/reviews/link-list-3-1-001.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。すべてのファイルがガイドラインに準拠しています。

### レビュー確認事項（問題なし）

**セキュリティ**:

- SQL クエリ（`FindBacklinkedPagesForTargets`, `CountBacklinkedPagesForTargets`）はすべて `space_id = $2` を WHERE 条件に含めている ✅
- `FindLinkedPagesPaginated` も `space_id` でスコープされている ✅
- テンプレートで `templ.SafeURL()` を使用して URL をサニタイズしている ✅
- ページネーションパラメータ `page` は `p > 0` でバリデーションされている ✅

**アーキテクチャ**:

- Handler → Repository → ViewModel の依存方向が正しく守られている ✅
- Handler は Query パッケージに直接依存していない ✅
- 読み取り専用の処理のため UseCase は作成せず Repository で完結させている（ガイドライン準拠）✅
- ドメイン ID 型（`model.PageID`, `model.SpaceID`）が正しく使用されている ✅

**国際化**:

- ユーザー向けテキストはすべて `templates.T(ctx, ...)` で翻訳されている ✅
- 翻訳キー `page_edit_links_load_more` は命名規則に準拠している ✅
- `ja.toml` と `en.toml` の両方に `description` 付きで翻訳が追加されている ✅

**テスト**:

- ViewModel（`backlink_list_test.go` 新規、`link_list_test.go` 更新）のテストが追加されている ✅
- Repository（`page_test.go`）に `FindBacklinksForPages` のテスト（4 サブテスト）が追加されている ✅
- Handler（`show_test.go`）にページネーションパラメータのテストが追加されている ✅
- `t.Parallel()` でテストが並行実行されている ✅
- `testutil.SetupTx(t)` パターンに準拠している ✅

**コーディング規約**:

- ログ出力はすべて `slog.ErrorContext(ctx, ...)` を使用している ✅
- コメントは日本語で記述されている ✅
- ファイル名は標準 9 種類に準拠している（`show.go`, `edit.go`）✅
- 共有定数 `LinkLimit`, `BacklinkLimit` が `viewmodel/link_list.go` に定義されている ✅
- N+1 クエリは LATERAL JOIN を使用した一括取得で解消されている ✅

**Datastar 統合**:

- `data-on:click` でコロン区切りの正しい構文を使用している ✅
- SSE レスポンスは `datastar.NewSSE` + `PatchElementTempl` の正しいパターンで実装されている ✅

## 設計改善の提案

設計改善の提案はありません。

## 設計との整合性チェック

作業計画書（タスク 3-1）の要件との整合性を確認しました:

| 要件                                     | 実装状況  |
| ---------------------------------------- | --------- |
| ページネーション付きリンク先取得         | ✅ 実装済 |
| バックリンクデータ取得                   | ✅ 実装済 |
| リンク一覧テンプレートにバックリンク表示 | ✅ 実装済 |
| バックリンク一覧コンポーネント新規作成   | ✅ 実装済 |
| 「もっと見る」ボタン                     | ✅ 実装済 |
| 翻訳ファイルへのメッセージ追加           | ✅ 実装済 |
| リンク一覧: 15 件/ページ                 | ✅ 実装済 |
| バックリンク一覧: 14 件/ページ           | ✅ 実装済 |
| ソート順: `modified_at DESC, id DESC`    | ✅ 実装済 |
| クエリに `space_id` を条件として含める   | ✅ 実装済 |
| edit.go と show.go の両方を更新          | ✅ 実装済 |

前回レビュー（001）で指摘された 3 点（テスト追加、N+1 解消、共有定数定義）もすべて対応済みです。

## 総合評価

**評価**: Approve

**総評**:

前回レビュー（001）の指摘事項への対応を含め、すべての変更がガイドラインに準拠しています。

**良かった点**:

- **セキュリティ**: SQL クエリにはすべて `space_id` が含まれており、スペース境界が正しく維持されている
- **N+1 解消**: LATERAL JOIN を使用した一括バックリンク取得（`FindBacklinksForPages`）により、クエリ数が大幅に削減されている
- **テストの網羅性**: ViewModel、Repository、Handler の各レイヤーにテストが追加されており、正常系・異常系ともにカバーされている
- **アーキテクチャ準拠**: 3 層アーキテクチャの依存方向が正しく守られている
- **既存コードとの一貫性**: `LinkListItem` と `BacklinkListItem` で `Page` ビューモデルをラップする統一パターンを採用している
