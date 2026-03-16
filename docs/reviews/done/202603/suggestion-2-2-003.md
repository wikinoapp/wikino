# コードレビュー: suggestion-2-2

## レビュー情報

| 項目                       | 内容                                  |
| -------------------------- | ------------------------------------- |
| レビュー日                 | 2026-03-16                            |
| 対象ブランチ               | suggestion-2-2                        |
| ベースブランチ             | develop                               |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md      |
| 変更ファイル数             | 16 ファイル（自動生成 1、レビュー 2） |
| 変更行数（実装）           | +486 / -25 行                         |
| 変更行数（テスト）         | +371 / -11 行                         |
| 変更行数（自動生成）       | +518 / -0 行（index_templ.go）        |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templ テンプレートガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/suggestion/handler.go`
- [x] `go/internal/handler/suggestion/index.go`
- [x] `go/internal/templates/page_name.go`
- [x] `go/internal/templates/path.go`
- [x] `go/internal/templates/pages/suggestion/index.templ`
- [x] `go/internal/usecase/get_suggestion_list.go`
- [x] `go/internal/viewmodel/suggestion.go`（差分なし、前タスクで作成済み）

### テストファイル

- [x] `go/internal/handler/suggestion/index_test.go`
- [x] `go/internal/handler/suggestion/main_test.go`
- [x] `go/internal/usecase/get_suggestion_list_test.go`

### 自動生成ファイル

- [x] `go/internal/templates/pages/suggestion/index_templ.go`（templ 自動生成、レビュー対象外）

### 翻訳ファイル

- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`

### ドキュメント

- [x] `docs/plans/1_doing/suggestion.md`
- [x] `docs/reviews/done/202603/suggestion-2-2-001.md`
- [x] `docs/reviews/done/202603/suggestion-2-2-002.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。全ファイルが各ガイドラインに準拠しています。

### チェック結果の詳細

**`go/cmd/server/main.go`**:

- リポジトリとユースケースの初期化パターンが既存コードと一貫
- ルーティング登録が `authMiddleware.SetUser` グループ（公開トピックは未ログインでも閲覧可能）に配置されており、作業計画書の要件に合致
- import エイリアス `suggestionhandler` は既存の `topichandler` と同じパターン

**`go/internal/handler/suggestion/handler.go`**:

- Handler 構造体のフィールド数は 3（上限 8 以下）
- 命名規則（`Handler`, `NewHandler`）がガイドラインに準拠
- 依存性注入パターンが既存ハンドラー（topic 等）と一貫

**`go/internal/handler/suggestion/index.go`**:

- UseCase 経由でデータ取得（Repository への直接依存なし）
- ViewModel を使って表示用データに変換
- 権限チェックは UseCase 内で実施（非公開トピックのアクセス制御）
- `slog.ErrorContext` でエラーログを出力（`log` パッケージ不使用）
- タイトル構築に `i18n.T` を使用（国際化対応済み）

**`go/internal/templates/pages/suggestion/index.templ`**:

- `IndexData` 構造体で ViewModel を構成要素として使用（templ-guide 準拠）
- `ctx` を明示的に渡さず templ の暗黙的 ctx を使用（templ-guide 準拠）
- 翻訳はすべて `templates.T(ctx, ...)` 経由（ハードコードなし）
- パス生成に `templates.SuggestionListPath` を使用

**`go/internal/usecase/get_suggestion_list.go`**:

- 読み取り UseCase（`Get` プレフィックス）で命名規則に準拠
- トランザクションなし（読み取り専用）
- 非公開トピックの権限チェック（スペースオーナーまたはトピックメンバーのみ）が正しく実装
- ステータスフィルタリングが UseCase 内で実施（ハンドラーではなく）
- `buildUserMap` で SpaceMemberID → User のマッピングを効率的に構築（N+1 回避）

**`go/internal/handler/suggestion/index_test.go`**:

- `TestMain` パターンで DB 接続を共有（testing-guide 準拠）
- `t.Parallel()` で並行実行
- テストヘルパー（`newIndexRequest`, `setupHandler`）でセットアップを共通化
- 正常系・異常系のテストケースが網羅的:
  - 存在しないスペース/トピック → 404
  - 不正なトピック番号 → 404
  - 公開トピック未ログイン閲覧 → 200
  - 非公開トピック未ログイン → 404
  - 非公開トピックオーナー閲覧 → 200
  - クローズタブのフィルタリング動作

**`go/internal/i18n/locales/ja.toml` / `en.toml`**:

- 命名規則 `{機能名}_{種別}_{詳細}` に準拠（例: `suggestion_index_open_tab`）
- `description` フィールドが全キーに記述
- 日本語・英語の両方が追加

## 設計との整合性チェック

作業計画書のタスク **2-2** の要件:

| 要件                                                                               | 状態 |
| ---------------------------------------------------------------------------------- | ---- |
| `internal/handler/suggestion/handler.go` に Handler 構造体を定義                   | ✅   |
| `internal/handler/suggestion/index.go` に `Index` メソッドを実装                   | ✅   |
| GET /s/{space}/topics/{topic}/suggestions のルーティング                           | ✅   |
| `internal/templates/pages/suggestion/index.templ` に一覧テンプレートを作成         | ✅   |
| オープン/クローズのタブ切り替え                                                    | ✅   |
| `cmd/server/main.go` にルーティング登録                                            | ✅   |
| 翻訳ファイル（ja.toml, en.toml）にメッセージ追加                                   | ✅   |
| ステータスフィルタリング（オープン: 下書き+オープン、クローズ: 反映済み+クローズ） | ✅   |
| 件数表示（タブにオープン/クローズの件数）                                          | ✅   |

設計との乖離はありません。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 2-2（編集提案一覧のハンドラーとテンプレート）の実装が作業計画書の要件通りに完了しています。アーキテクチャガイドライン（3 層アーキテクチャ、UseCase 経由のデータアクセス、ViewModel によるデータ変換）に正しく従っており、既存のハンドラー（topic/show.go 等）と一貫したパターンで実装されています。テストも正常系・異常系を網羅しており、品質は十分です。
