# コードレビュー: suggestion-6-2

## レビュー情報

| 項目                       | 内容                                   |
| -------------------------- | -------------------------------------- |
| レビュー日                 | 2026-03-19                             |
| 対象ブランチ               | suggestion-6-2                         |
| ベースブランチ             | develop                                |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md       |
| 変更ファイル数             | 19 ファイル（うち自動生成 2 ファイル） |
| 変更行数（実装）           | +457 / -13 行                          |
| 変更行数（テスト）         | +113 / -0 行                           |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templ テンプレートガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/db/queries/page_revisions.sql`
- [x] `go/internal/handler/suggestion_change/handler.go`
- [ ] `go/internal/handler/suggestion_change/index.go`
- [x] `go/internal/i18n/locales/en.toml`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/query/page_revisions.sql.go`（自動生成）
- [x] `go/internal/repository/page_revision.go`
- [x] `go/internal/templates/page_name.go`
- [x] `go/internal/templates/pages/suggestion/show.templ`
- [x] `go/internal/templates/pages/suggestion/show_templ.go`（自動生成）
- [ ] `go/internal/templates/pages/suggestion_change/index.templ`
- [x] `go/internal/templates/pages/suggestion_change/index_templ.go`（自動生成）
- [x] `go/internal/templates/path.go`
- [x] `go/internal/usecase/get_suggestion_diff.go`
- [x] `go/internal/viewmodel/suggestion.go`

### テストファイル

- [x] `go/internal/usecase/get_suggestion_diff_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/suggestion.md`
- [x] `docs/reviews/done/202603/suggestion-6-2-001.md`

## ファイルごとのレビュー結果

### `go/internal/handler/suggestion_change/index.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - UseCase の命名規則
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - ハンドラー実装パターン

**問題点・改善提案**:

- **[@go/docs/architecture-guide.md#UseCase の命名規則]**: `GetSuggestionDiffUsecase` の `Execute` が `nil` を返した場合（`FindByID` が `nil, nil` を返すケース）のハンドリングがない。`PageRevisionRepository.FindByID` は `sql.ErrNoRows` の場合に `nil, nil` を返す設計のため、存在しない `PageRevisionID` が渡されるとベースリビジョンが `nil` になる。`NewSuggestionPageDiffs` 内で `baseRev` が `nil` の場合のフォールバック処理はあるが、UseCase 側で `nil` チェック後にエラーを返すほうが安全ではないか。

  ```go
  // 現状のコード (get_suggestion_diff.go:41-46)
  rev, err := uc.pageRevisionRepo.FindByID(ctx, sp.PageRevisionID, input.SpaceID)
  if err != nil {
      return nil, fmt.Errorf("ベースリビジョンの取得に失敗: %w", err)
  }
  baseRevisions[sp.ID] = rev
  ```

  **修正案**:

  ```go
  rev, err := uc.pageRevisionRepo.FindByID(ctx, sp.PageRevisionID, input.SpaceID)
  if err != nil {
      return nil, fmt.Errorf("ベースリビジョンの取得に失敗: %w", err)
  }
  if rev == nil {
      return nil, fmt.Errorf("ベースリビジョンが見つかりません: pageRevisionID=%s, spaceID=%s", sp.PageRevisionID, input.SpaceID)
  }
  baseRevisions[sp.ID] = rev
  ```

  **対応方針**:
  - [x] 修正案の通り nil チェックを追加する
  - [ ] ViewModel 側のフォールバックで十分なため現状のまま
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/templates/pages/suggestion_change/index.templ`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - コンポーネントの再利用

**問題点・改善提案**:

- **[@go/docs/templ-guide.md#コンポーネントの再利用]**: `showStatusBadge` テンプレートが `suggestion/show.templ` と `suggestion_change/index.templ` で完全に重複している（約 22 行）。異なるパッケージ（`suggestion` と `suggestion_change`）に同じテンプレートが存在するため、ステータス表示の変更時に両方を修正する必要がある。`internal/templates/components/` に共通コンポーネントとして切り出すことを検討すべき。

  **修正案**:

  `internal/templates/components/suggestion_status_badge.templ` として共通コンポーネント化する：

  ```templ
  // components/suggestion_status_badge.templ
  package components

  import (
      "github.com/wikinoapp/wikino/go/internal/model"
      "github.com/wikinoapp/wikino/go/internal/templates"
  )

  templ SuggestionStatusBadge(status model.SuggestionStatus) {
      // 現在のshowStatusBadgeと同じ内容
  }
  ```

  **対応方針**:
  - [x] 修正案の通り共通コンポーネントとして切り出す
  - [ ] 現時点では重複を許容し、将来的に共通化する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Comment

**総評**:

タスク 6-2（「編集したページ」タブの実装）の要件を満たす実装になっている。アーキテクチャの依存関係ルール（Handler → UseCase → Repository）に正しく従っており、既存の `suggestion/show.go` ハンドラーと一貫したパターンで実装されている。

良い点：

- UseCase の責務が適切に分離されている（`GetSuggestionDetailUsecase` でデータ取得、`GetSuggestionDiffUsecase` でベースリビジョン取得）
- ViewModel の `NewSuggestionPageDiffs` で差分計算ロジックが適切にカプセル化されている
- テストが正常系・空配列の 2 ケースをカバーしている
- SQLクエリが `space_id` でスコープされておりセキュリティガイドラインに準拠
- 国際化が適切に行われている

指摘事項は軽微であり、いずれも必須対応ではない。
