# コードレビュー: handler-usecase-refactor-4-1

## レビュー情報

| 項目                       | 内容                                           |
| -------------------------- | ---------------------------------------------- |
| レビュー日                 | 2026-03-12                                     |
| 対象ブランチ               | handler-usecase-refactor-4-1                   |
| ベースブランチ             | handler-usecase-refactor-3-1                   |
| 作業計画書（指定があれば） | docs/plans/1_doing/handler-usecase-refactor.md |
| 変更ファイル数             | 7 ファイル                                     |
| 変更行数（実装）           | +165 / -103 行                                 |
| 変更行数（テスト）         | +234 / -5 行                                   |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/usecase/get_topic_detail.go`
- [x] `go/internal/handler/topic/handler.go`
- [x] `go/internal/handler/topic/show.go`
- [x] `go/cmd/server/main.go`

### テストファイル

- [x] `go/internal/usecase/get_topic_detail_test.go`
- [x] `go/internal/handler/topic/show_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/handler-usecase-refactor.md`

## ファイルごとのレビュー結果

問題のあるファイルはありませんでした。各ファイルの確認結果を以下にまとめます。

### 確認サマリー

**`go/internal/usecase/get_topic_detail.go`**:

- アーキテクチャ: UseCase は Repository のみに依存しており、3層アーキテクチャの依存ルールに準拠 ✅
- 命名: `GetTopicDetailUsecase`（動詞先頭、Usecase の c は小文字）— 既存パターンと一致 ✅
- Input/Output: `GetTopicDetailInput` / `GetTopicDetailOutput` — 既存パターンと一致 ✅
- エラーハンドリング: `fmt.Errorf` で `%w` を使ってラップ、日本語メッセージ ✅
- nil 返却パターン: space/topic が見つからない場合や権限不足時に `(nil, nil)` を返す設計が明確 ✅
- 権限チェック: 非公開トピックの閲覧権限ロジックが元の Handler から正確に移植されている ✅
- セキュリティ: space_id スコープのクエリを Repository 経由で使用 ✅

**`go/internal/handler/topic/handler.go`**:

- Handler 構造体から 5 つの Repository フィールドを削除し、UseCase 1 つに置換 ✅
- `repository` パッケージの import が完全に除去され、`usecase` パッケージに置換 ✅
- フィールド数は 4 個（8 個以下の上限を満たす） ✅
- 命名規則: `Handler` / `NewHandler`（ガイドライン通り） ✅

**`go/internal/handler/topic/show.go`**:

- Repository への直接依存が完全に除去され、UseCase 経由でデータ取得 ✅
- Handler の責務が HTTP 処理（URL パラメータ取得、UseCase 呼び出し、ViewModel 変換、レンダリング）に限定 ✅
- ログ出力: `slog.ErrorContext` を使用（coding-guide.md 準拠） ✅
- `model` パッケージへの直接依存は残存するが、これは `SpaceIdentifier` 型変換と `UserID` 型のためであり、ドメイン ID 型の使用ガイドラインに沿った正当な依存 ✅
- `usecase` パッケージの import が追加されている ✅

**`go/cmd/server/main.go`**:

- UseCase を main.go で構築し Handler に注入するパターン — Validator パッケージ分離時に確立されたパターンと一致 ✅
- `topicHandler` の初期化が簡潔になった（5 つの Repository → 1 つの UseCase） ✅

**`go/internal/usecase/get_topic_detail_test.go`**:

- `TestMain` パターン: 既存の `main_test.go` を共有（`usecase` パッケージの内部テスト） ✅
- `t.Parallel()` による並行実行 ✅
- `testutil.SetupTx` / `testutil.QueriesWithTx` を使用したトランザクション分離 ✅
- テストデータ作成: ビルダーパターンを使用 ✅
- テストケースの網羅性: 公開トピック（未ログイン、ログイン）、非公開トピック（未ログイン、オーナー、非メンバー）を網羅 ✅

**`go/internal/handler/topic/show_test.go`**:

- `setupHandler` ヘルパーが UseCase を構築して Handler に渡すように更新 ✅
- 既存のテストケースに変更なし（リファクタリングによる振る舞いの変化なし） ✅

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 4-1（GetTopicDetailUsecase の作成と topic ハンドラーの修正）が作業計画書通りに正確に実装されている。

良かった点:

- Handler から Repository への直接依存が完全に除去され、UseCase 経由に統一された
- 権限チェックロジック（非公開トピックの閲覧権限）が正確に UseCase に移植されている
- 既存の UseCase パターン（命名規則、構造体設計、エラーハンドリング）との一貫性が保たれている
- UseCase テストが権限パターンを網羅しており、テスト品質が高い
- Handler テストの既存テストケースが維持されており、リファクタリングの安全性が確保されている
- 実装コード +165/-103 行は PR サイズガイドライン（300 行以下目安）に収まっている
