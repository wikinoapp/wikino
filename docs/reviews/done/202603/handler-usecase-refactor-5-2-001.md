# コードレビュー: handler-usecase-refactor-5-2

## レビュー情報

| 項目                       | 内容                                           |
| -------------------------- | ---------------------------------------------- |
| レビュー日                 | 2026-03-12                                     |
| 対象ブランチ               | handler-usecase-refactor-5-2                   |
| ベースブランチ             | handler-usecase-refactor-5-1                   |
| 作業計画書（指定があれば） | docs/plans/1_doing/handler-usecase-refactor.md |
| 変更ファイル数             | 18 ファイル                                    |
| 変更行数（実装）           | +576 / -390 行                                 |
| 変更行数（テスト）         | +35 / -19 行                                   |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/page_backlink_list/handler.go`
- [x] `go/internal/handler/page_backlink_list/show.go`
- [x] `go/internal/handler/page_backlinks/handler.go`
- [x] `go/internal/handler/page_backlinks/show.go`
- [x] `go/internal/handler/page_link_list/handler.go`
- [x] `go/internal/handler/page_link_list/show.go`
- [x] `go/internal/handler/page_location/handler.go`
- [x] `go/internal/handler/page_location/index.go`
- [x] `go/internal/usecase/get_backlink_list.go`
- [x] `go/internal/usecase/get_link_list.go`
- [x] `go/internal/usecase/get_page_backlinks.go`
- [x] `go/internal/usecase/get_page_locations.go`

### テストファイル

- [x] `go/internal/handler/page_backlink_list/show_test.go`
- [x] `go/internal/handler/page_backlinks/show_test.go`
- [x] `go/internal/handler/page_link_list/show_test.go`
- [x] `go/internal/handler/page_location/index_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/handler-usecase-refactor.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。すべてのファイルがガイドラインに準拠しています。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 5-2（ページ補助ハンドラーの UseCase 化）が作業計画書通りに正しく実装されている。4 つのハンドラー（page_location, page_backlinks, page_backlink_list, page_link_list）すべてで Repository への直接依存が除去され、UseCase 経由に統一された。

**良かった点**:

- **アーキテクチャの一貫性**: 4 つの新しい読み取り UseCase（`GetPageLocationsUsecase`, `GetPageBacklinksUsecase`, `GetBacklinkListUsecase`, `GetLinkListUsecase`）はすべて既存の UseCase パターン（Input/Output 構造体、コンストラクタ DI、`nil` 返却による Not Found 表現）に準拠している
- **Handler の簡素化**: 各 Handler が UseCase のみに依存する形に整理され、Handler 構造体のフィールド数が大幅に削減された（例: page_link_list は 6 → 1 フィールド）
- **ヘルパー関数の再利用**: `collectTopicIDsFromPages` や `buildExcludePageIDs` など、既存の `get_edit_link_data.go` で定義されたヘルパーを適切に再利用している
- **型の再利用**: `EditLinkBacklinks` 構造体を `GetLinkListOutput` でも再利用しており、不要な型の重複を避けている
- **TopicPolicy チェックの Handler 残置**: 認可（TopicPolicy）チェックは Presentation 層の関心事として Handler に残しており、UseCase に認可ロジックが混入していない
- **テストの更新**: テストの `setupHandler` ヘルパーが UseCase 経由の構築に正しく更新されている
- **Handler から `repository` パッケージの import が完全に排除されている**（実装ファイルのみ。テストファイルでは UseCase 構築のため許容）
- **命名規則**: ファイル名（`get_backlink_list.go` 等）、構造体名（`GetBacklinkListUsecase` 等）ともにガイドラインの `{action}_{entity}` パターンに準拠
- **ログ出力**: `log/slog` の `slog.ErrorContext` を適切に使用
- **エラーラップ**: `fmt.Errorf("...: %w", err)` で日本語メッセージ付きのエラーラップが統一されている
