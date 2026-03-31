# コードレビュー: usecase-3-4

## レビュー情報

| 項目                       | 内容                                                 |
| -------------------------- | ---------------------------------------------------- |
| レビュー日                 | 2026-03-27                                           |
| 対象ブランチ               | usecase-3-4                                          |
| ベースブランチ             | usecase-3-3                                          |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md |
| 変更ファイル数             | 20 ファイル（実装 13 + テスト 7）                    |
| 変更行数（実装）           | +414 / -375 行                                       |
| 変更行数（テスト）         | +342 / -478 行                                       |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - 3層アーキテクチャ
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/draft_page/handler.go`
- [x] `go/internal/handler/draft_page/update.go`
- [x] `go/internal/handler/draft_page_revision/handler.go`
- [x] `go/internal/handler/draft_page_revision/update.go`
- [x] `go/internal/handler/page/handler.go`
- [x] `go/internal/handler/page/update.go`
- [x] `go/internal/usecase/auto_save_draft_page.go`
- [x] `go/internal/usecase/get_save_draft_page_data.go`（削除）
- [x] `go/internal/usecase/manual_save_draft_page.go`
- [x] `go/internal/usecase/page_access.go`（新規）
- [x] `go/internal/usecase/publish_page.go`
- [x] `go/internal/validator/page.go`

### テストファイル

- [x] `go/internal/handler/draft_page/update_test.go`
- [x] `go/internal/handler/draft_page_revision/update_test.go`
- [x] `go/internal/handler/page/edit_test.go`
- [x] `go/internal/usecase/auto_save_draft_page_test.go`
- [x] `go/internal/usecase/manual_save_draft_page_test.go`
- [x] `go/internal/usecase/publish_page_test.go`
- [x] `go/internal/validator/page_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/usecase-orchestration-refactor.md`

## ファイルごとのレビュー結果

問題のあるファイルのみ記載。問題がないファイルは「変更ファイル一覧」のチェックボックスにチェック済み。

（問題のあるファイルなし）

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

作業計画書の「検討事項 2: Read UseCase と Write UseCase の統合」および「検討事項 4: 書き込み UseCase のルール見直し」に沿って、3 つの書き込み UseCase（AutoSaveDraftPage、ManualSaveDraftPage、PublishPage）のオーケストレーション責務を Handler から UseCase に移動するリファクタリングが適切に実装されている。

**良い点**:

- **共通ヘルパーの抽出**: `page_access.go` に `fetchPageAccessData` と `authorizePageUpdate` を切り出し、3 つの UseCase で重複なく再利用している。ファイル名も「関数の責務を表す名詞」というガイドラインに従っている
- **Input の簡素化**: Handler が事前に解決した ID を渡す設計から、UseCase が識別子（SpaceIdentifier, PageNumber, UserID）を受け取って自ら解決する設計に変更。Handler の責務が HTTP 入出力変換に絞られ、作業計画書の意図通り
- **Validator の (data, error) パターン**: `PageUpdateValidator` が `*PageUpdateValidatorResult` から `(*model.PageID, error)` に変更され、作業計画書の確定方針に合致
- **エラーハンドリング**: Handler での `model.AsAppError` / `model.AsValidationError` による型判別パターンが一貫しており、作業計画書の設計通り
- **テストの適切な更新**: 新しい Input 型への対応に加え、UseCase テストに TopicMember セットアップを追加し、認可チェックが UseCase 内で正しく動作することを確認している
- **不要なテストの削除**: Handler テストから `TestUpdate_InvalidTopicNumber` と `TestUpdate_TopicNotFound` が削除され、UseCase 側に責務が移動したことを反映している
- **`get_save_draft_page_data.go` の削除**: Handler がオーケストレーターだった時代の読み取り UseCase が不要になり、きれいに削除されている
