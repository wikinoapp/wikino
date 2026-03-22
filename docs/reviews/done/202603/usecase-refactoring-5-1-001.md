# コードレビュー: usecase-refactoring-5-1

## レビュー情報

| 項目                       | 内容                                                  |
| -------------------------- | ----------------------------------------------------- |
| レビュー日                 | 2026-03-22                                            |
| 対象ブランチ               | usecase-refactoring-5-1                               |
| ベースブランチ             | develop                                               |
| 作業計画書（指定があれば） | docs/plans/1_doing/write-usecase-refactoring.md       |
| 変更ファイル数             | 11 ファイル                                           |
| 変更行数（実装）           | +163 / -247 行（実装 6 ファイル + 削除 2 ファイル）   |
| 変更行数（テスト）         | +201 / -303 行（テスト 3 ファイル + 削除 2 ファイル） |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - 3 層アーキテクチャ、UseCase
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約、コメント
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テスト戦略

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/usecase/create_suggestion.go`
- [x] `go/internal/handler/suggestion/create.go`
- [x] `go/internal/handler/suggestion/handler.go`
- [x] `go/cmd/server/main.go`
- [x] `go/internal/usecase/get_suggestion_body_html.go`（削除）
- [x] `go/internal/usecase/get_latest_page_revisions.go`（削除）

### テストファイル

- [x] `go/internal/usecase/create_suggestion_test.go`
- [x] `go/internal/handler/suggestion/index_test.go`
- [x] `go/internal/usecase/get_suggestion_body_html_test.go`（削除）
- [x] `go/internal/usecase/get_latest_page_revisions_test.go`（削除）

### 設定・その他

- [x] `docs/plans/1_doing/write-usecase-refactoring.md`

## ファイルごとのレビュー結果

問題のあるファイルはありませんでした。すべてのファイルがガイドラインに準拠しています。

## 設計との整合性チェック

### 作業計画書タスク 5-1 との整合性

作業計画書のタスク 5-1 に記載された要件をすべて確認しました：

- [x] `CreateSuggestionUsecase` に `topicRepo`, `pageRepo`, `pageRevisionRepo` を追加
- [x] `GetSuggestionBodyHTMLUsecase` のWikiリンク解決・Markdownレンダリングロジックを `CreateSuggestionUsecase` のトランザクション前に移動
- [x] `GetLatestPageRevisionsUsecase` のページリビジョン取得を `CreateSuggestionUsecase` のトランザクション前に移動
- [x] `CreateSuggestionInput` から `BodyHTML`, `PageRevisions` を削除し、`Body`, `CurrentTopicName`, `SpaceIdentifier` を追加
- [x] `get_suggestion_body_html.go`, `get_latest_page_revisions.go` を削除
- [x] Handler（`suggestion/create.go`）から2つの読み取りUseCaseの呼び出しを削除
- [x] Handler（`suggestion/handler.go`）から依存を削除
- [x] `cmd/server/main.go` のUseCase構築と依存注入を更新
- [x] 関連テストの更新

### 削除されたテストのカバレッジ

削除された2つのテストファイルのテストケースが新しいテストで適切にカバーされていることを確認しました：

| 削除されたテストケース                                       | 新しいカバレッジ                                             |
| ------------------------------------------------------------ | ------------------------------------------------------------ |
| `GetSuggestionBodyHTML`: Markdownが正しくHTMLに変換される    | `CreateSuggestion`: Markdownの本文が正しくHTMLに変換される   |
| `GetSuggestionBodyHTML`: Wikiリンクが解決される              | `CreateSuggestion`: Wikiリンクが解決される                   |
| `GetSuggestionBodyHTML`: 空の本文でも正しく処理される        | 既存テストケースが空の `Body` で実行しており暗黙的にカバー   |
| `GetLatestPageRevisions`: 各下書きページの最新リビジョン取得 | `CreateSuggestion`: 1つの下書きページから編集提案を作成      |
| `GetLatestPageRevisions`: 複数ページのリビジョン取得         | `CreateSuggestion`: 複数の下書きページから編集提案を作成     |
| `GetLatestPageRevisions`: ページリビジョンが存在しない場合   | `CreateSuggestion`: ページリビジョンが存在しない場合はエラー |

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

作業計画書タスク 5-1 の要件通りに実装されており、ガイドラインにも準拠しています。

良かった点：

- `CreateSuggestionUsecase` の `Execute` メソッドが「トランザクション前のデータ取得」と「トランザクション内の永続化」に明確に分離されており、作業計画書フェーズ5の方針（書き込みUseCase内でもトランザクション前ならデータ取得を許可）に沿っている
- `renderBodyHTML` と `fetchLatestPageRevisions` のメソッド分離により、Execute内にロジックを直接書かないルール（作業計画書 5-4 で予定されているガイドライン更新の先取り）を遵守している
- `resolveLinkedPages` 関数が `create_suggestion.go` に配置されており、共有ヘルパー（`uniqueTopicNames`）は `linked_page.go` から正しく参照されている
- 削除されたテストファイルのカバレッジが新しいテストで維持されている
- Handler の依存関係が2つ減り、シンプルになっている
- ビルドとリントが通っている
