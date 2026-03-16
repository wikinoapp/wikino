# コードレビュー: handler-usecase-refactor-6-1

## レビュー情報

| 項目                       | 内容                                           |
| -------------------------- | ---------------------------------------------- |
| レビュー日                 | 2026-03-13                                     |
| 対象ブランチ               | handler-usecase-refactor-6-1                   |
| ベースブランチ             | handler-usecase-refactor-5-3                   |
| 作業計画書（指定があれば） | docs/plans/1_doing/handler-usecase-refactor.md |
| 変更ファイル数             | 5 ファイル                                     |
| 変更行数（実装）           | +21 / -56 行（handler.go, show.go, main.go）   |
| 変更行数（テスト）         | +14 / -4 行（update_test.go）                  |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/draft_page/handler.go`
- [x] `go/internal/handler/draft_page/show.go`

### テストファイル

- [x] `go/internal/handler/draft_page/update_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/handler-usecase-refactor.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。すべてのファイルがガイドラインに従っています。

### レビュー詳細

**`go/internal/handler/draft_page/show.go`**:

チェックしたガイドライン:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - 依存関係ルール
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - ハンドラーガイドライン
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - ログ出力
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - 認可チェック

確認結果:

- ✅ Repository への直接依存が `getPageDetailUC` 経由に正しくリファクタリングされている
- ✅ UseCase の出力 (`output.Space`, `output.SpaceMember`, `output.Page`, `output.TopicMember`, `output.DraftPage`) を適切に使用
- ✅ 認可チェック（`policy.NewTopicPolicy`）が UseCase 出力のデータで正しく行われている
- ✅ エラーハンドリングが適切（`output == nil` で 404、`err != nil` で 500）
- ✅ ログ出力は `slog.ErrorContext(ctx, ...)` を使用

**`go/internal/handler/draft_page/handler.go`**:

確認結果:

- ✅ `draftPageRepo` フィールドが `getPageDetailUC` に置き換えられている
- ✅ 残りの Repository フィールド（`spaceRepo`, `spaceMemberRepo`, `pageRepo`, `topicRepo`, `topicMemberRepo`）は `update.go` で使用されるため残存が正当（タスク 6-2 のスコープ）

**`go/cmd/server/main.go`**:

確認結果:

- ✅ `draftPageRepo` の代わりに `getPageDetailUC` を `draft_page.NewHandler` に渡すように正しく更新

**`go/internal/handler/draft_page/update_test.go`**:

確認結果:

- ✅ `setupHandler` が `getPageDetailUC` を構築して Handler に渡すように更新
- ✅ Repository の変数を事前に作成して `NewGetPageDetailUsecase` と `NewHandler` の両方で再利用（重複作成を回避）

## 設計改善の提案

設計改善の提案はありません。

## 設計との整合性チェック

タスク 6-1 の説明:

> draft_page ハンドラーの参照系 UseCase 化（show, new, edit）
>
> - 下書きページ表示・新規作成フォーム・編集フォーム用の読み取り UseCase を作成
> - Handler 構造体から Repository フィールドを削除（書き込み UseCase は既存のものを使用）

**確認結果**:

- ✅ `show.go` の Repository 直接呼び出しが `getPageDetailUC` 経由に変更された
- ✅ `new.go` と `edit.go` は `draft_page` ハンドラーに存在しない（ページの new/edit は `page` ハンドラーのスコープ）。`draft_page` ハンドラーは `show`（SSE フラグメント返却）と `update`（自動保存）のみのため、show の UseCase 化でタスクのスコープは完了
- ✅ `draftPageRepo` フィールドが削除され `getPageDetailUC` に置き換えられた。他の Repository フィールドは `update.go`（タスク 6-2 スコープ）で使用されるため残存が正当
- ✅ 既存の `GetPageDetailUsecase` を再利用（`page/show.go` と同じ UseCase を共有）

## 総合評価

**評価**: Approve

**総評**:

`draft_page/show.go` の Repository 直接呼び出しが既存の `GetPageDetailUsecase` を活用して正しくリファクタリングされている。変更は最小限で、個々の Repository 呼び出し（スペース取得、メンバー確認、ページ取得、トピックメンバー取得、下書き取得）が 1 つの UseCase 呼び出しに集約され、コードが大幅に簡潔になった（52 行削除、17 行追加）。認可チェックやエラーハンドリングのパターンも既存コードと一貫している。テストの更新も適切に行われている。
