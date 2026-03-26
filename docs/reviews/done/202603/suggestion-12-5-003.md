# コードレビュー: suggestion-12-5

## レビュー情報

| 項目                       | 内容                                   |
| -------------------------- | -------------------------------------- |
| レビュー日                 | 2026-03-26                             |
| 対象ブランチ               | suggestion-12-5                        |
| ベースブランチ             | suggestion-12-4                        |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md       |
| 変更ファイル数             | 27 ファイル                            |
| 変更行数（実装）           | +694 行（うち生成コード +297 行）      |
| 変更行数（テスト）         | +974 行（テストヘルパー +88 行を含む） |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - 3 層アーキテクチャ
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templ テンプレートガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/db/queries/suggestion_comments.sql`
- [x] `go/internal/query/suggestion_comments.sql.go`（自動生成）
- [x] `go/internal/handler/suggestion_comment/handler.go`
- [x] `go/internal/handler/suggestion_comment/edit.go`
- [x] `go/internal/handler/suggestion_comment/update.go`
- [x] `go/internal/repository/suggestion_comment.go`
- [x] `go/internal/usecase/get_suggestion_comment.go`
- [x] `go/internal/usecase/update_suggestion_comment.go`
- [x] `go/internal/validator/suggestion_comment.go`
- [x] `go/internal/templates/pages/suggestion_comment/edit.templ`
- [x] `go/internal/templates/pages/suggestion_comment/edit_templ.go`（自動生成）
- [x] `go/internal/templates/page_name.go`
- [x] `go/internal/templates/path.go`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`

### テストファイル

- [x] `go/internal/handler/suggestion_comment/create_test.go`
- [x] `go/internal/handler/suggestion_comment/edit_test.go`
- [x] `go/internal/handler/suggestion_comment/update_test.go`
- [x] `go/internal/usecase/get_suggestion_comment_test.go`
- [x] `go/internal/usecase/update_suggestion_comment_test.go`
- [x] `go/internal/repository/suggestion_comment_test.go`
- [x] `go/internal/validator/suggestion_comment_test.go`
- [x] `go/internal/testutil/suggestion_comment_builder.go`

### 設定・その他

- [x] `docs/plans/1_doing/suggestion.md`
- [x] `docs/reviews/done/suggestion-12-5-001.md`
- [x] `docs/reviews/suggestion-12-5-002.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。全ファイルがガイドラインに準拠しています。

## 設計改善の提案

### `go/internal/handler/suggestion_comment/handler.go`: Handler 構造体のフィールド数

**ステータス**: 要確認

**現状**:

Handler 構造体のフィールドが 9 個あり、handler-guide の目安（8 個以下）を 1 つ超えている。

```go
type Handler struct {
    cfg                            *config.Config
    flashMgr                       *session.FlashManager
    getSuggestionDetailUsecase     *usecase.GetSuggestionDetailUsecase
    getSuggestionCommentUsecase    *usecase.GetSuggestionCommentUsecase
    createSuggestionCommentUsecase *usecase.CreateSuggestionCommentUsecase
    updateSuggestionCommentUsecase *usecase.UpdateSuggestionCommentUsecase
    sidebarHelper                  *sidebar.Helper
    createValidator                *validator.SuggestionCommentCreateValidator
    updateValidator                *validator.SuggestionCommentUpdateValidator
}
```

**提案**:

現時点ではリソース分割のコストが高く、9 個であれば許容範囲と判断できる。ただし、今後コメント削除等が追加される場合はリソース分割（例: `suggestion_comment_edit/` と `suggestion_comment/` への分離）を検討すべき。

**メリット**:

- 分割することで各 Handler の責務が明確になる
- フィールド数が減り可読性が向上する

**トレードオフ**:

- 現時点では 1 つ超過のみであり、分割のコスト（ディレクトリ追加、ルーティング変更）に見合わない
- コメントは単一リソースとして扱うのが自然であり、無理に分割すると直感に反する

**対応方針**:

- [x] 提案通り変更する
- [ ] 現状のまま（理由を回答欄に記入）
- [ ] その他（下の回答欄に記入）

**回答**:

```
（ここに回答を記入）
```

## 総合評価

**評価**: Approve

**総評**:

タスク 12-5（コメントの編集機能）が作業計画書の仕様通りに正確に実装されている。

**良い点**:

- **アーキテクチャ準拠**: Handler → UseCase → Repository の依存方向が正しく、Handler から Repository への直接依存がない。認可チェックは Handler で Policy を使って実行しており、UseCase は Policy に依存していない
- **セキュリティ**: CSRF トークン、Method Override、space_id によるクエリスコープがすべて適切。認証・認可チェックが Edit/Update の両方に実装されている
- **既存パターンとの一貫性**: suggestion/edit.go, suggestion/update.go と同じ処理フローを採用しており、コードベース全体の一貫性が保たれている
- **テストカバレッジ**: Handler、UseCase、Repository、Validator の全レイヤーにテストが追加されている。正常系・異常系（認証なし、404、バリデーションエラー、異なるスペース ID）を網羅
- **i18n 対応**: ja.toml / en.toml の両方に翻訳が追加され、description フィールドも適切に記述されている
- **テストヘルパーの整備**: SuggestionCommentBuilder / SuggestionCommentBuilderDB が追加され、他のテストからも再利用可能
