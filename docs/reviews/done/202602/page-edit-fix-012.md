# コードレビュー: page-edit-fix

## レビュー情報

| 項目                       | 内容                                         |
| -------------------------- | -------------------------------------------- |
| レビュー日                 | 2026-02-18                                   |
| 対象ブランチ               | page-edit-fix                                |
| ベースブランチ             | page-edit                                    |
| 作業計画書（指定があれば） | docs/plans/1_doing/page-edit-go-migration.md |
| 変更ファイル数             | 48 ファイル                                  |
| 変更行数（実装）           | +1679 / -64 行                               |
| 変更行数（テスト）         | +592 / -0 行                                 |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/db/queries/topics.sql`
- [x] `go/internal/handler/page/edit.go`
- [x] `go/internal/handler/page/handler.go`
- [x] `go/internal/handler/welcome/show.go`
- [x] `go/internal/i18n/locales/en.toml`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/middleware/reverse_proxy.go`
- [x] `go/internal/query/topics.sql.go`
- [x] `go/internal/repository/topic.go`
- [x] `go/internal/templates/components/flash.templ`
- [x] `go/internal/templates/components/flash_templ.go`
- [x] `go/internal/templates/components/top_nav.templ`
- [x] `go/internal/templates/components/top_nav_templ.go`
- [x] `go/internal/templates/helper.go`
- [x] `go/internal/templates/icons_custom.go`
- [x] `go/internal/templates/icons_phosphor.go`
- [x] `go/internal/templates/layouts/default.templ`
- [x] `go/internal/templates/layouts/default_templ.go`
- [x] `go/internal/templates/pages/account/new.templ`
- [x] `go/internal/templates/pages/account/new_templ.go`
- [x] `go/internal/templates/pages/email_confirmation/edit.templ`
- [x] `go/internal/templates/pages/email_confirmation/edit_templ.go`
- [x] `go/internal/templates/pages/page/edit.templ`
- [x] `go/internal/templates/pages/page/edit_templ.go`
- [x] `go/internal/templates/pages/password/reset.templ`
- [x] `go/internal/templates/pages/password/reset_sent.templ`
- [x] `go/internal/templates/pages/password/reset_sent_templ.go`
- [x] `go/internal/templates/pages/password/reset_templ.go`
- [x] `go/internal/templates/pages/sign_in/new.templ`
- [x] `go/internal/templates/pages/sign_in/new_templ.go`
- [x] `go/internal/templates/pages/sign_in_two_factor/new.templ`
- [x] `go/internal/templates/pages/sign_in_two_factor/new_templ.go`
- [x] `go/internal/templates/pages/sign_in_two_factor/recovery_new.templ`
- [x] `go/internal/templates/pages/sign_in_two_factor/recovery_new_templ.go`
- [x] `go/internal/templates/pages/sign_up/new.templ`
- [x] `go/internal/templates/pages/sign_up/new_templ.go`
- [x] `go/internal/templates/pages/welcome/show.templ`
- [x] `go/internal/templates/pages/welcome/show_templ.go`
- [x] `go/internal/templates/path.go`
- [x] `go/internal/viewmodel/space.go`
- [x] `go/internal/viewmodel/topic.go`

### テストファイル

- [x] `go/internal/handler/page/edit_test.go`
- [x] `go/internal/handler/page/main_test.go`
- [x] `go/internal/viewmodel/space_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/page-edit-go-migration.md`
- [x] `docs/plans/3_done/202602/icon-file-separation.md`
- [x] `docs/reviews/done/202602/page-edit-fix-011.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。すべてのファイルがガイドラインに準拠しています。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

ページ編集画面（GET）のGo版実装として、作業計画書の「ページを開く」フローに正確に対応した実装です。以下の点で品質が高いと評価します：

**セキュリティ**:

- 認証チェック（ログイン確認） → スペースメンバーチェック → トピックポリシーチェック（`CanUpdatePage`）の3段階の認可フローが適切に実装されている
- CSRFトークンがフォームに含まれている
- SQLクエリは`space_id`によるスコープが適用されている（`FindBySpaceAndID`、`FindBySpaceAndNumber`など）

**アーキテクチャ**:

- Handler → Repository の依存関係が正しく、Query への直接依存がない
- ハンドラーのファイル名（`handler.go`, `edit.go`）、メソッド名（`Edit`）がガイドラインに準拠
- Handler構造体のフィールド数が8個（ガイドライン上限）で適切
- `policy`パッケージによる認可ロジックの分離が適切

**設計との整合性**:

- 作業計画書の「DraftPageが存在すればその内容を表示し、存在しなければPageの現在の内容を表示する」フローが正確に実装されている
- タイトルが空の場合のオートフォーカス制御が実装されている
- パンくずリスト（TopNav）にスペース → トピック（アイコン付き）の階層が正しく表示される

**テスト**:

- 正常系（ページ表示、DraftPage優先表示、オートフォーカス）と異常系（未ログイン、スペース不在、メンバー外、ページ不在、無効な番号）が網羅されている
- 多言語テスト（英語ロケール）も含まれている
- TestMainパターン、SetupTx、t.Parallel()が正しく使用されている

**i18n**:

- すべてのユーザー向け文字列が`templates.T(ctx, ...)`で国際化されている
- 翻訳エントリにすべて`description`フィールドが含まれている

**リファクタリング（付随変更）**:

- `IconName`型の導入、アイコンデータの別ファイル分離、レイアウトの`Plain`→`Default`リネームなどのリファクタリングが適切に行われ、既存コードへの影響も正しく反映されている
- `Path`型の導入によりURL生成が型安全になっている

本PRはページ編集画面の表示（GET）のみで、更新処理（PATCH）は後続のタスクです。スコープが適切に絞られており、レビューしやすいサイズです。
