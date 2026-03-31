# コードレビュー: usecase-3-4

## レビュー情報

| 項目                       | 内容                                                 |
| -------------------------- | ---------------------------------------------------- |
| レビュー日                 | 2026-03-27                                           |
| 対象ブランチ               | usecase-3-4                                          |
| ベースブランチ             | usecase-3-3                                          |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md |
| 変更ファイル数             | 19 ファイル                                          |
| 変更行数（実装）           | +527 / -278 行                                       |
| 変更行数（テスト）         | +342 / -478 行                                       |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/usecase/publish_page.go`
- [ ] `go/internal/usecase/auto_save_draft_page.go`
- [ ] `go/internal/usecase/manual_save_draft_page.go`
- [x] `go/internal/validator/page.go`
- [x] `go/internal/handler/page/update.go`
- [x] `go/internal/handler/page/handler.go`
- [x] `go/internal/handler/draft_page/update.go`
- [x] `go/internal/handler/draft_page/handler.go`
- [x] `go/internal/handler/draft_page_revision/update.go`
- [x] `go/internal/handler/draft_page_revision/handler.go`
- [x] `go/cmd/server/main.go`

### テストファイル

- [x] `go/internal/usecase/publish_page_test.go`
- [x] `go/internal/usecase/auto_save_draft_page_test.go`
- [x] `go/internal/usecase/manual_save_draft_page_test.go`
- [x] `go/internal/validator/page_test.go`
- [x] `go/internal/handler/draft_page/update_test.go`
- [x] `go/internal/handler/draft_page_revision/update_test.go`
- [x] `go/internal/handler/page/edit_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/usecase-orchestration-refactor.md`

## ファイルごとのレビュー結果

### `go/internal/usecase/auto_save_draft_page.go`: authorize 内での DB アクセスが fetchData との一貫性を欠く

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - UseCase の処理順序
- 作業計画書 - 検討事項 4: 書き込み UseCase のルール（fetchData → authorize → validate → 永続化）

**問題点・改善提案**:

- **一貫性**: `auto_save_draft_page.go` と `manual_save_draft_page.go` では `topicMember` の取得を `authorize()` メソッド内で行っているが、`publish_page.go` では `fetchData()` 内で取得して `publishPageData` 構造体に含めている。

  `publish_page.go` のパターン:

  ```go
  // fetchData 内で取得
  type publishPageData struct {
      topicMember *model.TopicMember  // fetchDataで取得済み
  }

  func (uc *PublishPageUsecase) authorize(ctx context.Context, data *publishPageData) error {
      topicPolicy := policy.NewTopicPolicy(data.spaceMember, data.topicMember)
      // ...
  }
  ```

  `auto_save_draft_page.go` / `manual_save_draft_page.go` のパターン:

  ```go
  // fetchData では取得しない
  type autoSaveData struct {
      // topicMember なし
  }

  func (uc *AutoSaveDraftPageUsecase) authorize(ctx context.Context, data *autoSaveData) error {
      // authorize 内で DB アクセス
      topicMember, err := uc.topicMemberRepo.FindBySpaceMemberAndTopic(...)
      topicPolicy := policy.NewTopicPolicy(data.spaceMember, topicMember)
      // ...
  }
  ```

  同じリファクタリングタスク内で 2 種類のパターンが混在している。`authorize()` は認可チェックの判定に専念し、データ取得は `fetchData()` に統一するのが作業計画書の意図に沿う（フェーズ内の一貫性）。

  **修正案**:

  `auto_save_draft_page.go` と `manual_save_draft_page.go` の `topicMember` 取得を `fetchData()` に移動し、データ構造体に含める。`publish_page.go` と同じパターンに揃える。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 修正案の通り、fetchData に統一する
  - [ ] authorize 内での取得を意図的に使い分けている（理由を回答欄に記入）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/usecase/auto_save_draft_page.go` / `manual_save_draft_page.go`: 未使用の `GetSaveDraftPageDataUsecase` が残存

**ステータス**: 要確認

**チェックしたガイドライン**:

- 作業計画書 - タスク 3-4 の内容

**問題点・改善提案**:

- `go/internal/usecase/get_save_draft_page_data.go` が存在するが、`main.go` からの参照が削除され、どの Handler からも使用されていない。`draft_page/handler.go` から `getSaveDraftPageDataUC` フィールドも削除されている。このファイルはデッドコードになっている。

  **修正案**:

  `go/internal/usecase/get_save_draft_page_data.go` を削除する。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] このタスクで削除する
  - [ ] 別タスク（3a-3 等）で対応する
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

作業計画書タスク 3-4 の要件（publish_page, manual_save_draft_page, auto_save_draft_page の 3 つの UseCase にバリデーション・認可を統合し、Handler を薄い Adapter に変更する）は正しく実装されている。

**良かった点**:

- Handler から Policy / Validator の直接呼び出しが削除され、UseCase 経由に統一されている
- `PublishPageInput` が `SpaceIdentifier` + `PageNumber` + `UserID` というシンプルな HTTP 入力に統一され、Handler が内部 ID を知る必要がなくなった
- `handleUpdateError` のエラーハンドリングパターンが既存の `suggestion/create.go` と一貫している
- Validator の `PageUpdateValidator.Validate` が `(data, error)` の 2 値返しに変更され、Result 型が廃止されている
- テストも認可・データ取得のケースが UseCase テストに移動し、Handler テストは HTTP の振る舞いに集中している

**指摘事項**:

- `authorize()` 内での DB アクセスパターンが `publish_page.go` と `auto_save_draft_page.go` / `manual_save_draft_page.go` で不統一（2 件、軽微）
- `GetSaveDraftPageDataUsecase` がデッドコードとして残存（1 件、軽微）

いずれも軽微な指摘であり、必須の修正ではないため Comment としている。
