# コードレビュー: suggestion-12-4

## レビュー情報

| 項目                       | 内容                                              |
| -------------------------- | ------------------------------------------------- |
| レビュー日                 | 2026-03-26                                        |
| 対象ブランチ               | suggestion-12-4                                   |
| ベースブランチ             | suggestion-12-3                                   |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md                  |
| 変更ファイル数             | 23 ファイル（レビュー・計画書除くと 20 ファイル） |
| 変更行数（実装）           | +563 行                                           |
| 変更行数（テスト）         | +465 行                                           |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templ テンプレートガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/handler/suggestion/handler.go`
- [x] `go/internal/handler/suggestion/edit.go`
- [x] `go/internal/handler/suggestion/update.go`
- [x] `go/internal/usecase/update_suggestion.go`
- [x] `go/internal/validator/suggestion.go`
- [x] `go/internal/repository/suggestion.go`
- [x] `go/db/queries/suggestions.sql`
- [x] `go/internal/query/suggestions.sql.go`
- [x] `go/internal/templates/pages/suggestion/edit.templ`
- [x] `go/internal/templates/pages/suggestion/edit_templ.go`
- [x] `go/internal/templates/page_name.go`
- [x] `go/cmd/server/main.go`

### テストファイル

- [x] `go/internal/handler/suggestion/edit_test.go`
- [x] `go/internal/handler/suggestion/update_test.go`
- [x] `go/internal/handler/suggestion/index_test.go`
- [x] `go/internal/usecase/update_suggestion_test.go`
- [x] `go/internal/validator/suggestion_test.go`

### 設定・その他

- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`
- [x] `go/docs/handler-guide.md`
- [x] `docs/plans/1_doing/suggestion.md`
- [ ] `docs/reviews/suggestion-12-4-001.md`
- [ ] `docs/reviews/suggestion-12-4-002.md`

## ファイルごとのレビュー結果

問題のあるファイルのみ記載。問題がないファイルは変更ファイル一覧のチェックボックスにチェック済み。

### `go/internal/handler/suggestion/edit_test.go`: テストヘルパー `newSuggestionRequest` の重複リスク

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/handler-guide.md#テストファイル](/workspace/go/docs/handler-guide.md) - テスト関数の配置ルール
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド

**問題点・改善提案**:

- **[@go/docs/handler-guide.md#テストファイル]**: `edit_test.go` にヘルパー関数 `newSuggestionRequest` が定義されているが、この関数は `update_test.go` からも呼び出される可能性がある。実際に `update_test.go` では独自に `httptest.NewRequest` + `chi.NewRouteContext` を構築しており、パターンが統一されていない。テストヘルパーは `index_test.go`（`setupHandler` のように）にまとめるか、テストパッケージ内で共有されるファイルに配置するのが一貫性の面で望ましい

  **修正案**:

  `newSuggestionRequest` を `index_test.go`（既存の `setupHandler` と同様に共有ヘルパーを配置するファイル）に移動し、`update_test.go` からも利用する

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] `newSuggestionRequest` を `index_test.go` に移動し、`update_test.go` でも使用する
  - [ ] 現状のまま（テストが独立している方が良い）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

### `go/internal/handler/suggestion/handler.go`: Handler フィールド数が 10 個に到達

**ステータス**: 要確認

**現状**:

Handler 構造体のフィールドが 10 個になっている:

```go
type Handler struct {
    cfg                        *config.Config
    flashMgr                   *session.FlashManager
    getSuggestionListUsecase   *usecase.GetSuggestionListUsecase
    getSuggestionDetailUsecase *usecase.GetSuggestionDetailUsecase
    getSuggestionNewUsecase    *usecase.GetSuggestionNewUsecase
    createSuggestionUsecase    *usecase.CreateSuggestionUsecase
    updateSuggestionUsecase    *usecase.UpdateSuggestionUsecase
    sidebarHelper              *sidebar.Helper
    createValidator            *validator.SuggestionCreateValidator
    updateValidator            *validator.SuggestionUpdateValidator
}
```

**提案**:

[@go/docs/handler-guide.md#依存性注入のガイドライン](/workspace/go/docs/handler-guide.md) では「Handler 構造体のフィールドが 8 個を超えたらリソース分割を検討」とある。現時点では即座の分割は不要だが、今後 12-5（コメント編集）以降のタスクでさらにフィールドが増える可能性がある。

今すぐ対応する必要はないが、次のタスク追加時にフィールド数が 12 以上になる場合はリソース分割を検討すべき（例: `suggestion_edit/` としてEdit/Updateを独立ハンドラーに分離）。

**メリット**:

- ガイドラインとの整合性を維持
- 各ハンドラーの責務が明確になる

**トレードオフ**:

- 編集提案の関連機能が複数パッケージに散らばる
- 現時点で10個は「検討」のラインであり、即座の対応は必須ではない

**対応方針**:

- [ ] 今回は対応せず、次のタスクで分割を検討する
- [ ] 今回のPRでEdit/Updateを `suggestion_edit/` に分離する
- [ ] その他（下の回答欄に記入）

**回答**:

```
（ここに回答を記入）
```

### `go/internal/validator/suggestion.go`: CreateValidator に body 長さチェックがない

**ステータス**: 要確認

**現状**:

今回追加された `suggestionBodyMaxLength = 10000` は `SuggestionUpdateValidator` でのみ使用されている。しかし `SuggestionCreateValidator` では body の長さチェックが行われていない。作成時にも 10000 文字を超える本文が送信される可能性がある。

```go
// 更新時: body 長さチェックあり
if input.Body != "" && utf8.RuneCountInString(input.Body) > suggestionBodyMaxLength {
    formErrors.AddField("body", i18n.T(ctx, "validation_suggestion_body_too_long"))
}

// 作成時: body 長さチェックなし
```

**提案**:

`SuggestionCreateValidator` にも body の長さチェックを追加する。定数 `suggestionBodyMaxLength` は既にこの PR で追加されており、再利用するだけで済む。

**メリット**:

- 作成・更新で一貫したバリデーションルール
- 異常に長い本文の保存を防止

**トレードオフ**:

- 既存の CreateValidator の変更が必要（このPRのスコープ外）
- 作成時は body が省略可能な場合、影響は限定的

**対応方針**:

- [ ] このPRで CreateValidator にも body 長さチェックを追加する
- [ ] 別タスクとして対応する
- [ ] 対応不要（理由を回答欄に記入）

**回答**:

```
（ここに回答を記入）
```

## 総合評価

**評価**: Approve

**総評**:

タスク 12-4（編集提案本文の編集）の実装として、作業計画書の仕様を正確に満たしている。アーキテクチャガイドラインへの準拠も良好。

**良い点**:

- **3層アーキテクチャの厳守**: Handler → UseCase → Repository の依存方向が正しく、Handler から Repository への直接依存がない
- **読み取り → 検証 → 書き込みの処理フロー**: `edit.go`/`update.go` が読み取り UseCase → Policy → Validator → 書き込み UseCase の順序を正確に踏襲
- **書き込み UseCase のルール準拠**: トランザクション前にデータ取得（`renderBodyHTML`）、トランザクション内は永続化のみ（`updateSuggestion`）、Execute 内にロジックを直接書かず関数に分離
- **セキュリティ**: SQL クエリが `space_id` でスコープ、CSRF トークン、Method Override パターンの適切な使用
- **i18n の完全対応**: ja/en 両方の翻訳ファイルに必要なメッセージが追加済み
- **テストカバレッジ**: Handler・UseCase・Validator の各レイヤーにテストが存在し、テストガイドの方針を満たしている
- **templ テンプレートのデータ構造体パターン**: `EditData` 構造体を使用し、`ctx` を明示的に渡さないガイドライン準拠のパターン
- **ドキュメントの改善**: `handler-guide.md` にテスト関数の配置ルールを追記（実装で得た知見のフィードバック）
