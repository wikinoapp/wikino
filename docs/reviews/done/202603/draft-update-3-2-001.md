# コードレビュー: draft-update-3-2

## レビュー情報

| 項目                       | 内容                                   |
| -------------------------- | -------------------------------------- |
| レビュー日                 | 2026-03-06                             |
| 対象ブランチ               | draft-update-3-2                       |
| ベースブランチ             | draft-update-3-1                       |
| 作業計画書（指定があれば） | docs/plans/1_doing/draft-update.md     |
| 変更ファイル数             | 14 ファイル                            |
| 変更行数（実装）           | +199 / -6 行（自動生成 +200 行を除く） |
| 変更行数（テスト）         | +347 / -0 行                           |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/draft_page_index/handler.go`
- [x] `go/internal/handler/draft_page_index/index.go`
- [x] `go/internal/templates/page_name.go`
- [x] `go/internal/templates/path.go`
- [x] `go/internal/viewmodel/draft_page_for_index.go`
- [x] `go/internal/templates/pages/draft_page/index.templ`
- [x] `go/internal/templates/pages/draft_page/index_templ.go`（自動生成）

### テストファイル

- [x] `go/internal/handler/draft_page_index/index_test.go`
- [x] `go/internal/handler/draft_page_index/main_test.go`
- [x] `go/internal/viewmodel/draft_page_for_index_test.go`

### 設定・その他

- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`
- [x] `docs/plans/1_doing/draft-update.md`

## ファイルごとのレビュー結果

問題のあるファイルはありませんでした。全ファイルがガイドラインに従っています。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 3-2（下書き一覧画面の ViewModel・テンプレート・ハンドラー）が作業計画書の仕様通りに実装されています。

**良い点**:

- **アーキテクチャ準拠**: 3 層アーキテクチャに従い、Handler → Repository → Model の依存方向が正しい。Query への直接依存もない
- **ハンドラーガイドライン準拠**: `draft_page_index/` ディレクトリに `handler.go` + `index.go` の標準構成で実装されている
- **ViewModel の設計**: `DraftPageGroupForIndex` によるスペース・トピック単位のグルーピングロジックが ViewModel に適切に配置されている。`DisplayTitle` メソッドで「無題」のフォールバック処理を ViewModel 側に持たせている点も良い
- **既存コードとの一貫性**: `draftPageTitle` や `topicVisibilityIconName` など既存のヘルパー関数を再利用している
- **テンプレートガイドライン準拠**: 構造体ベースの引数パターン（`IndexData`）を使用し、`ctx` を明示的に渡していない
- **i18n 対応**: 翻訳キーの命名規則（`draft_page_index_*`）と `description` の記述が適切。ja/en 両方が追加されている
- **セキュリティ**: 認証チェック（`middleware.UserFromContext` + nil チェック）が適切に実装されている。`ListByUserForIndex` がユーザー ID でスコープされている
- **テストの充実**: 未ログイン、空一覧、データあり一覧の 3 パターンをカバー。ViewModel のグルーピングとタイトルフォールバックのテストも網羅的
- **PR サイズ**: 実装コード約 199 行（300 行以下の目安内）
