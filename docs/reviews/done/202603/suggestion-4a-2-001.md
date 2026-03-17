# コードレビュー: suggestion-4a-2

## レビュー情報

| 項目                       | 内容                             |
| -------------------------- | -------------------------------- |
| レビュー日                 | 2026-03-17                       |
| 対象ブランチ               | suggestion-4a-2                  |
| ベースブランチ             | suggestion-4a-1                  |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md |
| 変更ファイル数             | 13 ファイル                      |
| 変更行数（実装）           | +50 / -40 行                     |
| 変更行数（テスト）         | +7 / -32 行                      |

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

- [x] `go/cmd/server/main.go`
- [x] `go/db/queries/suggestions.sql`
- [x] `go/internal/handler/suggestion/show.go`
- [x] `go/internal/middleware/reverse_proxy.go`
- [x] `go/internal/query/suggestions.sql.go`
- [x] `go/internal/repository/suggestion.go`
- [x] `go/internal/templates/pages/suggestion/index.templ`
- [x] `go/internal/templates/pages/suggestion/index_templ.go`
- [x] `go/internal/templates/path.go`
- [x] `go/internal/usecase/get_suggestion_detail.go`

### テストファイル

- [x] `go/internal/handler/suggestion/show_test.go`
- [x] `go/internal/usecase/get_suggestion_detail_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/suggestion.md`

## ファイルごとのレビュー結果

すべてのファイルを確認し、問題は見つかりませんでした。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 4a-2（編集提案詳細URLから `topics/{topic}` を削除）が作業計画書の仕様通りに正しく実装されている。

変更内容:

- **ルーティング**: `GET /s/{space}/topics/{topic}/suggestions/{number}` → `GET /s/{space}/suggestions/{number}` に変更済み
- **UseCase**: `GetSuggestionDetailInput` から `TopicNumber` を削除し、`FindBySpaceAndNumber` でスペースIDと番号ベースの検索に変更。トピックは編集提案の `TopicID` から逆引きする設計に変更済み
- **Repository**: `FindBySpaceAndNumber` メソッドを追加し、対応するsqlcクエリも追加済み
- **Handler**: URLパラメータから `topic_number` の取得を削除し、シンプル化済み
- **テンプレート**: `SuggestionShowPath` の呼び出しを2引数（spaceIdentifier, suggestionNumber）に更新済み
- **フィーチャーフラグ**: 新URLパターン `^/s/[^/]+/suggestions/\d+` をフィーチャーフラグのパターンに追加済み
- **テスト**: 全テストのURL・パラメータを更新し、不要になった「存在しないトピックの場合はnilが返る」テストケースを削除済み

アーキテクチャガイドラインの依存関係ルール（Handler → UseCase → Repository → Query）に従っており、セキュリティ面でもスペースIDによるクエリスコープが維持されている。変更は小さく焦点が絞られており、PRサイズのガイドラインにも適合している。
