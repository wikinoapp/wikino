# コードレビュー: page-edit-5-4

## レビュー情報

| 項目                       | 内容                                                      |
| -------------------------- | --------------------------------------------------------- |
| レビュー日                 | 2026-02-19                                                |
| 対象ブランチ               | page-edit-5-4                                             |
| ベースブランチ             | page-edit                                                 |
| 作業計画書（指定があれば） | docs/plans/1_doing/page-edit-go-migration.md（タスク5-4） |
| 変更ファイル数             | 9 ファイル                                                |
| 変更行数（実装）           | +237 / -1 行                                              |
| 変更行数（テスト）         | +427 / -0 行                                              |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/db/queries/pages.sql`
- [x] `go/internal/handler/page_location/handler.go`
- [x] `go/internal/handler/page_location/index.go`
- [x] `go/internal/query/pages.sql.go`
- [x] `go/internal/repository/page.go`

### テストファイル

- [x] `go/internal/handler/page_location/index_test.go`
- [x] `go/internal/handler/page_location/main_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/page-edit-go-migration.md`

## ファイルごとのレビュー結果

### `go/internal/repository/page.go`: SearchPageLocationsのLIKEワイルドカード未エスケープ

**ステータス**: 対応済み

**チェックしたガイドライン**:

- [@go/docs/security-guide.md#SQLインジェクション対策](/workspace/go/docs/security-guide.md)

**問題点・改善提案**:

- **[@go/docs/security-guide.md#入力バリデーション]**: `SearchPageLocations`メソッド（167-204行目）で、ユーザー入力のクエリ文字列をILIKEパターンに変換する際、PostgreSQLのLIKE特殊文字（`%`、`_`、`\`）をエスケープしていない

  SQLパラメータ化クエリを使用しているためSQLインジェクションではないが、ユーザーが`%`や`_`を含む入力をした場合、意図しないパターンマッチが発生する（LIKEパターンインジェクション）。例えば`_`を入力すると任意の1文字にマッチし、想定外の検索結果が返る。

  ```go
  // 問題のあるコード（171-173行目）
  for i, word := range words {
      patterns[i] = fmt.Sprintf("%%%s%%", word)
  }
  ```

  **修正案**:

  ```go
  // PostgreSQLのLIKE特殊文字をエスケープしてからワイルドカードで囲む
  func escapeLikePattern(s string) string {
      s = strings.ReplaceAll(s, `\`, `\\`)
      s = strings.ReplaceAll(s, `%`, `\%`)
      s = strings.ReplaceAll(s, `_`, `\_`)
      return s
  }

  // 使用箇所
  for i, word := range words {
      patterns[i] = fmt.Sprintf("%%%s%%", escapeLikePattern(word))
  }
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 修正案の通りエスケープ関数を追加する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  修正案の通り、escapeLikePattern関数を追加してSearchPageLocationsメソッド内で使用するようにしました。
  ```

### `go/internal/handler/page_location/index.go`: エラーレスポンスのContent-Type未設定

**ステータス**: 対応済み

**チェックしたガイドライン**:

- [@go/docs/security-guide.md#エラーメッセージ](/workspace/go/docs/security-guide.md)

**問題点・改善提案**:

- **一貫性の確認**: 正常時はContent-Typeを`application/json`に設定しているが（85行目）、エラー時の`http.Error()`はデフォルトで`text/plain`を返す。JSONを期待するクライアント（Wikiリンク補完のfetch）がエラー時にテキストレスポンスを受け取ることになる。

  現在のフロントエンドの実装次第では問題にならないが、API一貫性の観点で確認が必要。

  ```go
  // 現状（42-44行目）: エラー時はtext/plainで返る
  slog.ErrorContext(ctx, "スペースの取得に失敗", "error", err)
  http.Error(w, "Internal Server Error", http.StatusInternalServerError)
  ```

  **修正案**:

  既存ハンドラーの他のJSON APIエンドポイントでも同様のパターンを使用しているか確認の上、統一する方針があるなら合わせる。現状の実装で他のJSON APIと一貫しているなら問題なし。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [ ] 既存パターンと一貫しているため現状のまま
  - [x] エラー時もJSONレスポンスに統一する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  writeJSONError ヘルパー関数を追加し、http.Error() と http.NotFound() を全て
  writeJSONError() に置き換えました。エラー時もContent-Type: application/jsonで
  統一されるようになりました。
  ```

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Request Changes

**総評**:

全体的に作業計画書（タスク5-4）の仕様に忠実に実装されており、アーキテクチャの依存関係ルール（Handler → Repository → Query）も正しく守られている。テストも正常系・異常系を幅広くカバーしており、品質が高い。

**良い点**:

- ハンドラーのディレクトリ構造、ファイル命名（`handler.go`, `index.go`）がガイドライン通り
- SQLクエリで`space_id`によるスコープが正しく適用されている（セキュリティガイドライン準拠）
- 認証チェック（ログイン確認）と認可チェック（スペースメンバー確認）が適切に実装されている
- 公開済み・未廃棄・未ゴミ箱のフィルタリングが仕様通り
- テストが6ケース（正常検索、複数ワードAND検索、空クエリ、非公開ページ除外、未ログイン、非メンバー、スペース不存在）を網羅
- `main_test.go`のTestMainパターンが正しく使用されている

**必須対応**:

1. LIKEワイルドカード文字のエスケープ漏れ（セキュリティ）— 機能的なバグにもなり得るため修正が必要
