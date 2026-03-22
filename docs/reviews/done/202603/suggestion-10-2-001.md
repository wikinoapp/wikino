# コードレビュー: suggestion-10-2

## レビュー情報

| 項目                       | 内容                                         |
| -------------------------- | -------------------------------------------- |
| レビュー日                 | 2026-03-22                                   |
| 対象ブランチ               | suggestion-10-2                              |
| ベースブランチ             | develop                                      |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md（10-2）     |
| 変更ファイル数             | 14 ファイル                                  |
| 変更行数（実装）           | +102 / -20 行（自動生成除く実装ファイル約8） |
| 変更行数（テスト）         | +94 / -3 行                                  |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/db/queries/joined_draft_pages.sql`
- [x] `go/internal/handler/draft_page_index/index.go`
- [x] `go/internal/query/joined_draft_pages.sql.go`（自動生成）
- [x] `go/internal/repository/draft_page.go`
- [x] `go/internal/templates/pages/draft_page/index.templ`
- [x] `go/internal/templates/pages/draft_page/index_templ.go`（自動生成）
- [x] `go/internal/usecase/get_draft_pages.go`
- [x] `go/internal/viewmodel/draft_page_for_index.go`

### テストファイル

- [x] `go/internal/handler/draft_page_index/index_test.go`
- [x] `go/internal/usecase/get_draft_pages_test.go`

### 設定・その他

- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`
- [x] `docs/plans/1_doing/suggestion.md`

## ファイルごとのレビュー結果

問題のあるファイルはありませんでした。全ファイルがガイドラインに準拠しています。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

フェーズ10-2「下書き一覧画面の『編集提案する...』ボタン」の実装が、作業計画書の仕様通りに完了している。

**良かった点**:

- **アーキテクチャの遵守**: 3層アーキテクチャの依存関係ルールに完全に準拠。Handler → UseCase → Repository の依存の方向が正しく、Handler からの直接 Repository アクセスは一切ない
- **フィーチャーフラグの適切な配置**: `GetDraftPagesUsecase` 内でフィーチャーフラグを確認し、テンプレートでは `SuggestionEnabled` の真偽値のみで制御。テンプレートがリポジトリに依存しない設計
- **テストの網羅性**: フィーチャーフラグ無効時に「編集提案する」ボタンが非表示になることのテスト、フラグ有効時にボタンと正しいURLが表示されることのテスト、UseCase単体テストが揃っている
- **国際化の徹底**: ボタンラベルが `templates.T(ctx, "draft_page_index_create_suggestion_button")` で国際化されており、ja/en両方の翻訳が追加済み
- **既存コードとの一貫性**: `DraftPageGroupForIndex` の拡張（`TopicNumber` フィールド追加）、SQLクエリへの `t.number` カラム追加、Repository の変換処理追加がすべて既存パターンに従っている
- **変更量が適切**: 実装コードは作業計画書の想定に沿った小さな差分に収まっている
