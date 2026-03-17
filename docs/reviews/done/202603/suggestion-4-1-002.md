# コードレビュー: suggestion-4-1

## レビュー情報

| 項目                       | 内容                                   |
| -------------------------- | -------------------------------------- |
| レビュー日                 | 2026-03-17                             |
| 対象ブランチ               | suggestion-4-1                         |
| ベースブランチ             | suggestion-3-2                         |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md       |
| 変更ファイル数             | 8 ファイル                             |
| 変更行数（実装）           | +301 行（自動生成の sqlc 31 行を除く） |
| 変更行数（テスト）         | +382 行                                |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧

## 変更ファイル一覧

### 実装ファイル

- [x] `go/db/queries/suggestions.sql`
- [x] `go/internal/query/suggestions.sql.go`（自動生成）
- [x] `go/internal/repository/suggestion.go`
- [x] `go/internal/usecase/get_suggestion_detail.go`
- [x] `go/internal/viewmodel/suggestion.go`

### テストファイル

- [x] `go/internal/usecase/get_suggestion_detail_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/suggestion.md`（タスクチェックボックスの更新のみ）
- [x] `docs/reviews/done/202603/suggestion-4-1-001.md`（前回レビュー）

## ファイルごとのレビュー結果

問題のあるファイルはありません。すべてのファイルがガイドラインに準拠しています。

### チェック結果サマリー

**`go/db/queries/suggestions.sql`**:

- [@go/docs/security-guide.md#スペースIDによるクエリスコープ](/workspace/go/docs/security-guide.md): `space_id` でスコープされている ✅
- 命名規則: `FindSuggestionByTopicAndNumber` は既存パターンに合致 ✅

**`go/internal/repository/suggestion.go`**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md): Repository → Query の依存のみ ✅
- `FindByTopicAndNumber` は `sql.ErrNoRows` を `nil` に変換する既存パターンに準拠 ✅
- ドメイン ID 型の変換が正しい ✅

**`go/internal/usecase/get_suggestion_detail.go`**:

- [@go/docs/architecture-guide.md#ユースケース](/workspace/go/docs/architecture-guide.md): 読み取り UseCase のパターンに準拠（`Get` プレフィックス、トランザクションなし） ✅
- ファイル名 `get_suggestion_detail.go` は `{action}_{entity}.go` パターンに合致 ✅
- 構造体名 `GetSuggestionDetailUsecase` は既存パターンに合致 ✅
- Query への直接依存なし（Repository 経由のみ） ✅
- 非公開トピックの権限チェックが `get_suggestion_list.go` と同一パターンで実装されている ✅
- エラーメッセージは日本語（開発者向け）で統一 ✅
- `buildUserMap` の SpaceMemberID → User 変換パターンは `get_suggestion_list.go` と一貫 ✅

**`go/internal/viewmodel/suggestion.go`**:

- [@go/docs/architecture-guide.md#ビューモデル](/workspace/go/docs/architecture-guide.md): Model → ViewModel の変換のみ ✅
- 画面の要件に応じた ViewModel 定義（`SuggestionForDetail`, `SuggestionCommentForList`, `SuggestionPageForList`） ✅
- 入力パラメータを構造体で受け取るパターンに準拠 ✅
- Repository / Query への依存なし ✅

**`go/internal/usecase/get_suggestion_detail_test.go`**:

- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md): `SetupTx` + `t.Parallel()` パターンに準拠 ✅
- テストビルダーパターンを使用 ✅
- 正常系（詳細取得、ページ取得、コメント取得、UserMap、SpaceMember/TopicMember） ✅
- 異常系（存在しないスペース、トピック、編集提案番号） ✅
- 非公開トピックのアクセス制御テスト（未ログイン、オーナー、トピックメンバー、非メンバー） ✅

### 設計との整合性

作業計画書のタスク **4-1**「編集提案詳細の UseCase・ViewModel」の要件:

- [x] `internal/usecase/get_suggestion_detail.go` に `GetSuggestionDetailUsecase` を作成
- [x] 編集提案 + コメント一覧 + 編集ページ一覧取得
- [x] `internal/viewmodel/suggestion.go` に `SuggestionForDetail`, `SuggestionCommentForList` ViewModel を追加
- [x] 追加で `SuggestionPageForList` ViewModel も作成（作業計画書には明記されていないが、タスク 4-2 のテンプレートで必要になるため妥当）

## 設計改善の提案

### `go/internal/usecase/get_suggestion_detail.go`: buildUserMap の重複

**ステータス**: 要確認

**現状**:

`get_suggestion_detail.go` の `buildUserMap` と `get_suggestion_list.go` の `buildUserMap` は、SpaceMemberID → User の解決ロジック（ID 収集 → SpaceMember 一括取得 → User 一括取得 → マップ構築）がほぼ同一です。唯一の違いは、detail 版がコメント投稿者の SpaceMemberID も収集する点です。

```go
// get_suggestion_list.go: buildUserMap
func (uc *GetSuggestionListUsecase) buildUserMap(ctx context.Context, suggestions []*model.Suggestion, spaceID model.SpaceID) (map[model.SpaceMemberID]*model.User, error) {
    // SpaceMemberIDを収集（suggestionsのみ）
    // SpaceMember一括取得 → User一括取得 → マップ構築
}

// get_suggestion_detail.go: buildUserMap
func (uc *GetSuggestionDetailUsecase) buildUserMap(ctx context.Context, suggestion *model.Suggestion, comments []*model.SuggestionComment, spaceID model.SpaceID) (map[model.SpaceMemberID]*model.User, error) {
    // SpaceMemberIDを収集（suggestion + comments）
    // SpaceMember一括取得 → User一括取得 → マップ構築（同一ロジック）
}
```

**提案**:

SpaceMemberID のスライスを受け取って User マップを返す共通関数を `usecase` パッケージ内に抽出する。

```go
// internal/usecase/build_user_map.go
func buildUserMap(ctx context.Context, spaceMemberRepo *repository.SpaceMemberRepository, userRepo *repository.UserRepository, memberIDs []model.SpaceMemberID, spaceID model.SpaceID) (map[model.SpaceMemberID]*model.User, error) {
    // 共通ロジック
}
```

各 UseCase では SpaceMemberID の収集のみを担当し、共通関数を呼び出す。

**メリット**:

- 重複コードの削減（約 30 行の節約）
- 今後のコメント機能等でも同じパターンが必要になる可能性が高い

**トレードオフ**:

- YAGNI 原則との兼ね合い（現時点では 2 箇所のみの重複）
- UseCase が自己完結しなくなる（ただしパッケージ内の共通関数なので影響は軽微）

**対応方針**:

<!-- 開発者が回答を記入してください -->

- [x] 提案通り共通関数を抽出する
- [ ] 現状のまま（重複は 2 箇所のみで許容範囲）
- [ ] その他（下の回答欄に記入）

**回答**:

```
（ここに回答を記入）
```

## 総合評価

**評価**: Approve

**総評**:

タスク 4-1 の要件を正確に実装しています。以下の点が良好です:

- **既存パターンとの一貫性**: `get_suggestion_list.go` と同一のスペース→トピック→権限チェックのフローを踏襲しており、コードベース全体の統一感が保たれている
- **セキュリティ**: SQL クエリの `space_id` スコープ、非公開トピックのアクセス制御が適切に実装されている
- **テストの網羅性**: 正常系（5 ケース）+ 異常系（3 ケース）+ 非公開トピック（4 ケース）= 計 12 テストケースで主要なシナリオをカバーしている
- **ViewModel の設計**: 画面要件に応じた適切な粒度で ViewModel が定義されている

`buildUserMap` の重複は設計改善の提案として記載しましたが、現時点ではブロッカーではなく、マージ可能と判断します。
