# コードレビュー: suggestion-7-1

## レビュー情報

| 項目                       | 内容                                                 |
| -------------------------- | ---------------------------------------------------- |
| レビュー日                 | 2026-03-19                                           |
| 対象ブランチ               | suggestion-7-1                                       |
| ベースブランチ             | suggestion-6-2                                       |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md                     |
| 変更ファイル数             | 8 ファイル（うちレビュー済み docs 4 ファイルを除く） |
| 変更行数（実装）           | +215 行                                              |
| 変更行数（テスト）         | +375 行                                              |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - 3層アーキテクチャ、UseCase、WithTxパターン
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約、コメント、ログ出力
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テスト戦略、TestMainパターン
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - SpaceIDスコープ

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/usecase/apply_suggestion.go`
- [x] `go/internal/testutil/suggestion_page_builder.go`

### テストファイル

- [x] `go/internal/usecase/apply_suggestion_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/suggestion.md`
- [x] `docs/reviews/done/202603/suggestion-7-1-001.md`
- [x] `docs/reviews/done/202603/suggestion-7-1-002.md`
- [x] `docs/reviews/done/202603/suggestion-7-1-003.md`
- [x] `docs/reviews/done/202603/suggestion-7-1-004.md`

## ファイルごとのレビュー結果

問題のあるファイルのみ記載。問題がないファイルは「変更ファイル一覧」のチェックボックスにチェック済み。

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 7-1（編集提案反映UseCase）の実装は、既存の `publish_page.go` のパターンに忠実に従っており、アーキテクチャガイドラインに準拠した高品質な実装です。

**良かった点**:

- **既存パターンとの一貫性**: `publish_page.go` の Page更新 → PageRevision作成 → PageEditor追加・更新 → TopicMember更新のフローを正確に再現している
- **WithTxパターンの正しい使用**: すべてのリポジトリに対して `WithTx(tx)` を呼び出し、トランザクション内で操作している
- **SpaceIDスコープ**: セキュリティガイドラインに従い、すべてのリポジトリ呼び出しで `SpaceID` を渡している
- **テストカバレッジ**: 正常系2件（単一ページ・複数ページ）と異常系3件（下書き・クローズ・反映済み・存在しないID）で主要パターンを網羅
- **段階的実装の計画**: `LinkedPageIDs` と `FeaturedImageAttachmentID` の扱いをTODOコメントで明示し、後続タスク（7-1a, 7-1b）として作業計画書に整理されている
- **テストビルダー**: 既存のビルダーパターン（Tx版/DB版の2バリアント、Raw SQL INSERT、必須フィールドのバリデーション）を正確に踏襲している
