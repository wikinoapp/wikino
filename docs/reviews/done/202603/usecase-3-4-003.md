# コードレビュー: usecase-3-4

## レビュー情報

| 項目                       | 内容                                                          |
| -------------------------- | ------------------------------------------------------------- |
| レビュー日                 | 2026-03-27                                                    |
| 対象ブランチ               | usecase-3-4                                                   |
| ベースブランチ             | usecase-3-3                                                   |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md（3-4）   |
| 変更ファイル数             | 22 ファイル（うちドキュメント・レビュー 3 ファイルを除く 19） |
| 変更行数（実装）           | +537 / -377 行                                                |
| 変更行数（テスト）         | +342 / -478 行                                                |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
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
- [ ] `go/internal/handler/page/update.go`
- [ ] `go/internal/usecase/auto_save_draft_page.go`
- [x] `go/internal/usecase/get_save_draft_page_data.go`（削除）
- [ ] `go/internal/usecase/manual_save_draft_page.go`
- [ ] `go/internal/usecase/publish_page.go`
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

- [x] `docs/plans/1_doing/usecase-orchestration-refactor.md`（チェックボックス更新）
- [x] `docs/reviews/done/usecase-3-4-001.md`
- [x] `docs/reviews/usecase-3-4-002.md`

## ファイルごとのレビュー結果

### `go/internal/handler/page/update.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - ハンドラーガイドライン
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド

**問題点・改善提案**:

- **`renderEditWithErrors` 内の `w.WriteHeader(http.StatusUnprocessableEntity)` が削除されている**: `handleUpdateError` 側で `w.WriteHeader(http.StatusUnprocessableEntity)` を呼び出してから `renderEditWithErrors` を呼ぶ構造に変更されている。`renderEditWithErrors` から `w.WriteHeader` の呼び出しが削除されたのは正しいが、**この関数が他の場所からも呼ばれる可能性**を考えると、呼び出し側でステータスコードを設定する責務が分散しないか確認が必要。現状では `handleUpdateError` からしか呼ばれていないため問題なし。ただし、将来的に別のエラーハンドラーから呼ばれた場合にステータスコード設定が漏れるリスクがある。

  **修正案**:

  現状の実装でも機能的には正しい。ただし `renderEditWithErrors` のドキュメントコメントに「呼び出し側でステータスコードを設定すること」を明記するか、あるいは `renderEditWithErrors` 内に `w.WriteHeader` を戻して呼び出し側での重複設定を避ける。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [ ] 現状のまま（呼び出し元は1箇所なので問題なし）
  - [x] `renderEditWithErrors` 内に `w.WriteHeader` を戻す
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/usecase/publish_page.go`、`manual_save_draft_page.go`、`auto_save_draft_page.go` 共通

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約

**問題点・改善提案**:

- **3つの UseCase に同一パターンの `fetchData` + `authorize` が重複している**: `auto_save_draft_page.go`、`manual_save_draft_page.go`、`publish_page.go` の3つすべてに、ほぼ同じ構造の `fetchData` メソッド（Space → SpaceMember → Page → Topic → TopicMember の取得）と `authorize` メソッド（SpaceMember nil チェック + TopicPolicy.CanUpdatePage チェック）が存在する。各 UseCase のデータ構造体（`autoSaveData`、`manualSaveData`、`publishPageData`）もフィールドがほぼ同一。

  これは現行のアーキテクチャガイドの「Execute 内にロジックを直接書かない。ロジックは関数やメソッドとして定義」に従っており、各 UseCase が独立して動作できるため機能的には問題ない。しかし、同じロジックが3箇所にコピーされている状態はバグ修正漏れのリスクがある。

  **修正案**:

  `internal/usecase/` 内に共通のヘルパーファイル（例: `page_access.go`）を作成し、Space → SpaceMember → Page → Topic → TopicMember の取得と認可チェックを共通化する。既存の `linked_page.go` パターンと同様。

  ```go
  // internal/usecase/page_access.go
  type pageAccessData struct {
      space       *model.Space
      spaceMember *model.SpaceMember
      page        *model.Page
      topic       *model.Topic
      topicMember *model.TopicMember
  }

  func fetchPageAccessData(ctx context.Context, spaceIdentifier model.SpaceIdentifier, pageNumber int32, userID model.UserID, repos ...) (*pageAccessData, error) { ... }

  func authorizePageUpdate(ctx context.Context, data *pageAccessData) error { ... }
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 共通ヘルパーを作成して重複を排除する
  - [ ] 現状のまま（各 UseCase の独立性を優先、将来の共通化はフェーズ5で検討）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

- **`publishPageData` は `draftPage` フィールドを含むが、他2つのデータ構造体には含まれていない**: publish_page だけが `draftPage` を必要とする。これは正しい差異だが、共通化する場合の考慮点として記録。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 認識済み、問題なし

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

### `go/internal/usecase/publish_page.go`: `PublishPageInput.Title` の型を `string` にした影響

**ステータス**: 要確認

**現状**:

変更前の `PublishPageInput.Title` は `*string`（nil = タイトルなし）だったが、変更後は `string`（空文字列 = タイトルなし）に変更された。`publishPage` メソッド内で空文字列をポインタに変換している。

```go
// publishPage 内
var titlePtr *string
if input.Title != "" {
    titlePtr = &input.Title
}
```

一方、`ManualSaveDraftPageInput.Title` と `AutoSaveDraftPageInput.Title` は引き続き `*string` のまま。

**提案**:

3つの UseCase の Input で Title の型が異なる理由を明確にする。`PublishPageInput` は Validator を通すため `string`（必須フィールド）として受け取り、`ManualSaveDraftPageInput` / `AutoSaveDraftPageInput` はバリデーション不要のため `*string`（省略可能）として残すのは妥当。ただし、コメントで意図を説明すると将来の開発者が混乱しにくい。

**メリット**:

- 意図が明確になり保守性が向上

**トレードオフ**:

- コメント追加のみなので影響は軽微

**対応方針**:

- [x] `PublishPageInput.Title` に「バリデーション済みの必須フィールド」とコメントを追加
- [ ] 現状のまま（コードから読み取れる範囲で十分）
- [ ] その他（下の回答欄に記入）

**回答**:

```
（ここに回答を記入）
```

## 総合評価

**評価**: Approve

**総評**:

タスク 3-4（publish_page, manual_save_draft_page, auto_save_draft_page UseCase の移行）は作業計画書の設計通りに正しく実装されている。

**良かった点**:

- 3つの UseCase すべてで、データ取得 → 認可 → バリデーション/永続化の処理順序が作業計画書の確定方針に一致している
- Handler が薄い Adapter になり、`errors.As` パターンでエラーを判別する構造が他の移行済み UseCase と一貫している
- `GetSaveDraftPageDataUsecase` の削除により、不要になった読み取り UseCase が適切にクリーンアップされている
- Validator の Result 型が廃止され `(data, error)` の2値返しに正しく変更されている
- テストが既存のカバレッジを維持しつつ、新しいインターフェースに適切に更新されている
- `ManualSaveDraftPageOutput.TopicNumber` の追加により、Handler が UseCase の内部データに依存しない形でリダイレクトパスを構築できるようになった

**指摘事項**:

- 3つの UseCase に同一パターンの `fetchData` + `authorize` が重複している点は、即座の修正ではなく将来の共通化検討として記録
- `renderEditWithErrors` の `WriteHeader` 呼び出し責務の移動は機能的に問題ないが、呼び出し規約の明確化を推奨
