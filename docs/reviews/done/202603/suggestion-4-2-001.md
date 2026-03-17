# コードレビュー: suggestion-4-2

## レビュー情報

| 項目                       | 内容                                  |
| -------------------------- | ------------------------------------- |
| レビュー日                 | 2026-03-17                            |
| 対象ブランチ               | suggestion-4-2                        |
| ベースブランチ             | suggestion-4-1                        |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md      |
| 変更ファイル数             | 14 ファイル                           |
| 変更行数（実装）           | +638 / -23 行（自動生成ファイル除く） |
| 変更行数（テスト）         | +262 / -0 行                          |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/suggestion/handler.go`
- [x] `go/internal/handler/suggestion/show.go`
- [x] `go/internal/usecase/get_suggestion_detail.go`
- [x] `go/internal/viewmodel/suggestion.go`
- [x] `go/internal/templates/pages/suggestion/show.templ`
- [x] `go/internal/templates/pages/suggestion/index.templ`
- [x] `go/internal/templates/path.go`
- [x] `go/internal/templates/page_name.go`

### テストファイル

- [x] `go/internal/handler/suggestion/show_test.go`
- [x] `go/internal/handler/suggestion/index_test.go`

### 設定・その他

- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`
- [x] `go/internal/templates/pages/suggestion/show_templ.go`（自動生成）
- [x] `go/internal/templates/pages/suggestion/index_templ.go`（自動生成）
- [ ] `docs/plans/1_doing/suggestion.md`

## ファイルごとのレビュー結果

### `docs/plans/1_doing/suggestion.md`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@CLAUDE.md](/workspace/CLAUDE.md) - 作業計画書のガイドライン

**問題点・改善提案**:

- **作業計画書の整合性**: フェーズ4a（`4a-1`, `4a-2`）で編集提案番号をスペーススコープに変更しURLを `/s/{space}/suggestions/{number}` に変更する計画が追加されている。現在のフェーズ4-2の実装では `/s/{space}/topics/{topic}/suggestions/{number}` のままであり、フェーズ4aで変更予定と明記されている。この段階的な移行計画は妥当だが、フェーズ4-2のタスク説明にURLがフェーズ4aで変更予定であることが記載されている点を確認。

  **対応方針**:
  - [x] 現状のまま（段階的移行として妥当）
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

フェーズ4-2（編集提案詳細のハンドラーとテンプレート）の実装として、作業計画書の仕様に忠実に実装されている。

**良かった点**:

- **アーキテクチャの遵守**: Handler → UseCase → Repository の依存関係が正しく守られている。Handlerからの直接的なRepository依存はない
- **既存パターンとの一貫性**: `show.go` のハンドラー実装パターン（URLパラメータ取得 → UseCase実行 → ViewModel変換 → レイアウトレンダリング）が既存の `index.go` と一貫している
- **templ テンプレートガイドの遵守**: `ShowData` 構造体をテンプレートに渡す形式が正しく、ViewModelを構成要素として使用している。`context.Context` を明示的に渡していない
- **i18n の徹底**: すべてのユーザー向けメッセージが翻訳キーを使用しており、日本語・英語の両方のtomlファイルに追加されている。命名規則（`suggestion_show_*`）も統一されている
- **セキュリティ**: `templ.Raw()` の使用はサーバー側で生成された `BodyHTML`（Markdownパイプライン経由）に限定されており、他のテンプレートと同じパターンで安全
- **テストの網羅性**: 404ケース（存在しないスペース、不正な提案番号、存在しない提案番号）、公開トピックの未ログイン閲覧、非公開トピックの権限チェック（未ログイン拒否、オーナー許可）が適切にカバーされている
- **UseCase の設計**: `buildUserMap` で重複排除を行い、N+1問題を回避している。権限チェック（非公開トピックのアクセス制御）もUseCase内で適切に実装されている
- **一覧画面のリンク追加**: `index.templ` の各編集提案項目が `<div>` から `<a>` タグに変更され、詳細画面へのリンクが追加されている
