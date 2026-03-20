# コードレビュー: usecase-refactoring-3-1

## レビュー情報

| 項目                       | 内容                                                      |
| -------------------------- | --------------------------------------------------------- |
| レビュー日                 | 2026-03-20                                                |
| 対象ブランチ               | usecase-refactoring-3-1                                   |
| ベースブランチ             | develop                                                   |
| 作業計画書（指定があれば） | docs/plans/1_doing/write-usecase-refactoring.md           |
| 変更ファイル数             | 12 ファイル（うちドキュメント 2、Go 実装 6、Go テスト 4） |
| 変更行数（実装）           | +249 / -136 行                                            |
| 変更行数（テスト）         | +320 / -86 行                                             |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - 3 層アーキテクチャ、UseCase、Handler での処理フロー
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラー、依存性注入のガイドライン
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約、コメント、ログ出力
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テスト戦略

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/suggestion/create.go`
- [x] `go/internal/handler/suggestion/handler.go`
- [x] `go/internal/usecase/create_suggestion.go`
- [x] `go/internal/usecase/get_latest_page_revisions.go`
- [x] `go/internal/usecase/get_suggestion_body_html.go`

### テストファイル

- [x] `go/internal/handler/suggestion/index_test.go`
- [x] `go/internal/usecase/create_suggestion_test.go`
- [x] `go/internal/usecase/get_latest_page_revisions_test.go`
- [x] `go/internal/usecase/get_suggestion_body_html_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/write-usecase-refactoring.md`
- [x] `docs/reviews/usecase-refactoring-3-1-001.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。全ファイルがガイドラインに準拠しています。

## 設計改善の提案

### `go/internal/handler/suggestion/handler.go`: Handler 構造体のフィールド数が 10 に増加

**ステータス**: 要確認

**現状**:

リファクタリングにより `getSuggestionBodyHTMLUsecase` と `getLatestPageRevisionsUsecase` が追加され、Handler 構造体のフィールド数が 8 から 10 に増加した。

```go
type Handler struct {
    cfg                           *config.Config
    flashMgr                      *session.FlashManager
    getSuggestionListUsecase      *usecase.GetSuggestionListUsecase
    getSuggestionDetailUsecase    *usecase.GetSuggestionDetailUsecase
    getSuggestionNewUsecase       *usecase.GetSuggestionNewUsecase
    getSuggestionBodyHTMLUsecase  *usecase.GetSuggestionBodyHTMLUsecase
    getLatestPageRevisionsUsecase *usecase.GetLatestPageRevisionsUsecase
    createSuggestionUsecase       *usecase.CreateSuggestionUsecase
    sidebarHelper                 *sidebar.Helper
    createValidator               *validator.SuggestionCreateValidator
}
```

**提案**:

[@go/docs/handler-guide.md#依存性注入のガイドライン](/workspace/go/docs/handler-guide.md) では「Handler 構造体のフィールドが 8 個を超えたら、リソース分割を検討」としている。ただし、今回のフィールド増加はリファクタリングにより書き込み UseCase から読み取りロジックを分離した結果であり、このリファクタリングの性質上やむを得ない面がある。

将来的に `suggestion_close` / `suggestion_apply` ハンドラー側でも同様のリファクタリングが進むと、さらにフィールドが増える可能性がある。suggestion パッケージ全体の分割（例: `suggestion_create/` への分離）を将来のフェーズで検討する価値がある。

**メリット**:

- ハンドラーの責務が明確になる
- 各ハンドラーのコンストラクタ引数が減る

**トレードオフ**:

- 現時点で分割すると、このリファクタリング PR の範囲を超える
- 10 フィールドはガイドラインの「検討」ラインであり、直ちに問題ではない

**対応方針**:

<!-- 開発者が回答を記入してください -->

- [ ] 将来のフェーズ（フェーズ 4 以降）でリソース分割を検討する
- [ ] このタスクの範囲内で分割する
- [ ] 現状のまま（理由を回答欄に記入）

**回答**:

```
（ここに回答を記入）
```

## 設計との整合性チェック

作業計画書タスク **3-1** の要件との整合性を確認した。

| 要件                                                                       | 状態 | 備考                                                                                                             |
| -------------------------------------------------------------------------- | ---- | ---------------------------------------------------------------------------------------------------------------- |
| Wikiリンク解決（`resolveLinkedPages`）をUseCase外で実行するように変更      | ✅   | `GetSuggestionBodyHTMLUsecase` として読み取り UseCase に分離                                                     |
| `resolveLinkedPages` を Handler 側で呼び出し、結果を Input に渡す          | ✅   | Handler が `getSuggestionBodyHTMLUsecase.Execute` を呼び出し、`BodyHTML` を Input に渡している                   |
| ページリビジョン取得を読み取り UseCase で事前に行い、結果を Input に含める | ✅   | `GetLatestPageRevisionsUsecase` として実装                                                                       |
| `CreateSuggestionInput` に `BodyHTML`, `PageRevisions` を追加              | ✅   | 追加済み                                                                                                         |
| `CreateSuggestionInput` に `PageLocations` を追加（作業計画書の記載）      | ⚠️   | 追加されていないが、`BodyHTML` に Wikiリンク解決済みの HTML が含まれるため不要。作業計画書の記載より合理的な設計 |
| `CreateSuggestionInput` から `SpaceIdentifier`, `CurrentTopicName` を削除  | ✅   | Wikiリンク解決が UseCase 外に移動したため不要になり、正しく削除されている                                        |
| UseCase から `pageRevisionRepo`, `topicRepo`, `pageRepo` の依存を削除      | ✅   | 削除済み                                                                                                         |
| 関連テストの更新                                                           | ✅   | UseCase テスト、Handler テストともに更新済み                                                                     |
| 外部から見た挙動（HTTPレスポンス、永続化結果）は変わらない                 | ✅   | リファクタリングのみで振る舞いに変更なし                                                                         |

**補足**: 作業計画書では `PageLocations` を `CreateSuggestionInput` に追加する設計だったが、実装では `GetSuggestionBodyHTMLUsecase` が Wikiリンク解決済みの HTML を返すため、`PageLocations` は不要となった。これはより合理的な設計判断であり、作業計画書の更新を推奨する。

## 総合評価

**評価**: Approve

**総評**:

`create_suggestion.go` のリファクタリングが作業計画書の方針に沿って適切に実施されている。

**良かった点**:

- 書き込み UseCase からデータ取得ロジック（Wikiリンク解決、ページリビジョン取得）が完全に分離され、アーキテクチャガイドの「Handler での処理フロー（読み取り → 検証 → 書き込み）」パターンに準拠している
- 新規読み取り UseCase（`GetSuggestionBodyHTMLUsecase`, `GetLatestPageRevisionsUsecase`）の命名が `Get` プレフィックスの規約に従っている
- テストカバレッジが十分（正常系・異常系を網羅）
- 作業計画書の `PageLocations` を Input に渡す設計よりも、`BodyHTML` に解決済み HTML を含める設計のほうがシンプルで良い判断

**注意点**:

- Handler 構造体のフィールド数が 10 に増加（ガイドライン上限 8 を超過）。将来的なリソース分割を検討する価値がある
- 作業計画書のタスク 3-1 の説明で `PageLocations` を Input に追加する記載があるが、実装と乖離がある。作業計画書の更新を推奨
