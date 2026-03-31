# コードレビュー: usecase-3a-3

## レビュー情報

| 項目                       | 内容                                                 |
| -------------------------- | ---------------------------------------------------- |
| レビュー日                 | 2026-03-30                                           |
| 対象ブランチ               | usecase-3a-3                                         |
| ベースブランチ             | usecase-3a-2                                         |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md |
| 変更ファイル数             | 12 ファイル                                          |
| 変更行数（実装）           | +156 / -22 行                                        |
| 変更行数（テスト）         | +295 / -8 行                                         |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/usecase/get_suggestion_edit.go`
- [x] `go/internal/handler/suggestion/edit.go`
- [x] `go/internal/handler/suggestion/handler.go`
- [x] `go/internal/handler/suggestion/update.go`
- [x] `go/internal/handler/suggestion_comment_edit/edit.go`
- [x] `go/internal/handler/suggestion_comment_edit/handler.go`
- [x] `go/internal/handler/suggestion_comment_edit/update.go`
- [x] `go/cmd/server/main.go`

### テストファイル

- [x] `go/internal/usecase/get_suggestion_edit_test.go`
- [x] `go/internal/handler/suggestion/index_test.go`
- [x] `go/internal/handler/suggestion_comment_edit/edit_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/usecase-orchestration-refactor.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。全ファイルがガイドラインに準拠しています。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 3a-3「フォーム表示用の読み取り UseCase の整理」が作業計画書通りに実装されている。

**良い点**:

- **不要データの排除**: `GetSuggestionDetailUsecase` が取得していた `SuggestionPages`、`Pages`、`Comments`、`SpaceMember`、`TopicMember` を排除し、edit フォームに必要な最小限のデータのみ取得する `GetSuggestionEditUsecase` を新設。効率的なデータ取得になっている
- **Input の型設計が適切**: Detail の `UserID *model.UserID`（未ログイン許容）に対し、Edit の `UserID model.UserID`（認証必須）とポインタ/非ポインタを正しく使い分けている
- **認可チェックの統合**: UseCase 内で Policy による認可チェックを行い、`CanUpdateSuggestion` / `CanUpdateSuggestionComment` をブール値で返す設計は、作業計画書の「UseCase がオーケストレーター」方針と整合している
- **既存パターンとの一貫性**: 既存の `GetSuggestionDetailUsecase` と同じパターン（nil リターンによるリソース不在表現、`buildUserMapBySpaceMemberIDs` の再利用等）を踏襲している
- **テストカバレッジ**: 正常系、リソース不在、非メンバー、非公開トピックの各権限パターン（オーナー、トピックメンバー、非トピックメンバー）を網羅しており十分
- **変更が対象 Handler に漏れなく適用**: `suggestion/edit.go`、`suggestion/update.go`、`suggestion_comment_edit/edit.go`、`suggestion_comment_edit/update.go` の 4 箇所すべてで新 UseCase に切り替え済み
- **PR サイズが適切**: 実装コード約 150 行、テストコード約 295 行で、ガイドラインの範囲内
