# コードレビュー: suggestion-3-2

## レビュー情報

| 項目                       | 内容                                        |
| -------------------------- | ------------------------------------------- |
| レビュー日                 | 2026-03-17                                  |
| 対象ブランチ               | suggestion-3-2                              |
| ベースブランチ             | suggestion-3-1                              |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md            |
| 変更ファイル数             | 18 ファイル                                 |
| 変更行数（実装）           | +992 / -16 行（自動生成・ドキュメント除く） |
| 変更行数（テスト）         | +0 行                                       |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templ テンプレートガイド
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/suggestion/handler.go`
- [ ] `go/internal/handler/suggestion/new.go`
- [x] `go/internal/handler/suggestion/create.go`
- [x] `go/internal/usecase/get_suggestion_new.go`
- [x] `go/internal/viewmodel/suggestion.go`
- [ ] `go/internal/templates/pages/suggestion/new.templ`
- [x] `go/internal/templates/pages/suggestion/new_templ.go`（自動生成）
- [ ] `go/internal/templates/path.go`
- [x] `go/internal/templates/page_name.go`
- [x] `go/db/queries/draft_pages.sql`
- [x] `go/internal/query/draft_pages.sql.go`（自動生成）
- [x] `go/internal/repository/draft_page.go`

### テストファイル

- [x] `go/internal/handler/suggestion/index_test.go`（既存テストの Handler 初期化変更のみ）

### 設定・その他

- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`
- [x] `docs/plans/1_doing/suggestion.md`（タスクチェック更新）
- [x] `docs/reviews/done/202603/suggestion-3-2-001.md`（前回レビュー）

## ファイルごとのレビュー結果

### `go/internal/handler/suggestion/new.go`: 未使用のテンプレートヘルパーインポート

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - テンプレート関数の引数パターン

**問題点・改善提案**:

- **[@go/docs/templ-guide.md#テンプレート関数の引数パターン]**: `renderNewForm` で `flash` を取得しているが、`New` ハンドラーの初回表示時にフラッシュメッセージが必要になるケースは考えにくい。ただし Create からのリダイレクト後にフラッシュが必要になる可能性もあるので、現状維持でも問題ない。

  **対応方針**:
  - [x] 現状のまま（将来の拡張で必要になる可能性があるため）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/templates/pages/suggestion/new.templ`: `isSelected` ヘルパー関数の配置

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - ヘルパー関数

**問題点・改善提案**:

- `isSelected` 関数が `new.templ` ファイル内の Go 関数として定義されている（行 157-163）。templ ファイル内に Go 関数を定義すること自体は問題ないが、この関数はテンプレート描画のヘルパーとして適切な位置にあるか確認したい。他のテンプレートで同様のパターンが使われていれば一貫性がある。

  **対応方針**:
  - [x] 現状のまま（テンプレート内のヘルパー関数として適切）
  - [ ] `internal/templates/helper.go` に移動する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/templates/path.go`: `SuggestionCreatePath` と `SuggestionNewPath` の使用状況

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約

**問題点・改善提案**:

- **未使用の可能性**: `SuggestionNewPath` が定義されている（行 98-100）が、コードベース内で一度も使用されていない。`SuggestionCreatePath` はテンプレートの `<form action>` で使用されているが、`SuggestionNewPath` の呼び出し元が見つからない。

  **修正案**: 使用されていない場合は削除するか、使用予定がある場合は今後のタスクで利用するか確認する。

  **対応方針**:
  - [ ] 削除する（未使用のため）
  - [x] 現状のまま（今後の編集提案一覧画面の「新規作成」リンクで使用予定）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

- **`SuggestionCreatePath` と `SuggestionListPath` の重複**: 両関数は同一のパスを生成する（`/s/{space}/topics/{topic}/suggestions`）。HTTP メソッドの違い（POST vs GET）で区別されるため意図的な設計であるが、将来の保守性のためにコメントで明記されている点は良い。問題なし。

## 設計改善の提案

### テスト不足

**ステータス**: 要修正

**現状**:

このPRでは以下の実装が追加されているが、対応するテストが一切追加されていない:

- `internal/handler/suggestion/new.go` - New ハンドラー
- `internal/handler/suggestion/create.go` - Create ハンドラー
- `internal/usecase/get_suggestion_new.go` - GetSuggestionNewUsecase
- `internal/viewmodel/suggestion.go` - DraftPageForSuggestionNew / NewDraftPagesForSuggestionNew
- `internal/repository/draft_page.go` - ListByMemberAndTopic

既存の `index_test.go` の変更は、Handler コンストラクタのシグネチャ変更に伴う修正のみ。

**提案**:

[@CLAUDE.md#Pull Requestのガイドライン](/workspace/CLAUDE.md) に「実装コードとそのテストコードは同じ PR に含める」「テストがない実装は原則としてマージしない」と明記されている。以下のテストを追加すべき:

1. **ハンドラーテスト** (`new_test.go` / `create_test.go` or `handler_test.go`):
   - 未ログイン時のリダイレクト
   - 存在しないスペース/トピックでの 404
   - スペースメンバーでない場合の 404
   - 正常なフォーム表示（200）
   - バリデーションエラー時のフォーム再表示（422）
   - 正常な作成とリダイレクト（303）

2. **UseCaseテスト** (`get_suggestion_new_test.go`):
   - 正常なデータ取得
   - スペースが存在しない場合
   - スペースメンバーでない場合
   - 非公開トピックのアクセス制御

3. **ViewModel テスト**:
   - `DraftPageForSuggestionNew.DisplayTitle` の動作確認
   - `NewDraftPagesForSuggestionNew` の変換ロジック

4. **Repository テスト**:
   - `ListByMemberAndTopic` の正常系・異常系

**メリット**:

- PR のガイドラインに準拠
- リグレッション防止
- コードの動作保証

**トレードオフ**:

- PR サイズが増加する（ただしテストコードの行数制限はないため問題なし）

**対応方針**:

- [x] テストを追加してからマージする
- [ ] テストは別PRで追加する（理由を回答欄に記入）
- [ ] その他（下の回答欄に記入）

**回答**:

```
（ここに回答を記入）
```

## 総合評価

**評価**: Request Changes

**総評**:

実装自体は既存のパターンに忠実で品質が高い。具体的に良い点:

- **アーキテクチャ準拠**: 3 層アーキテクチャの依存関係ルールに正しく従っている（Handler → UseCase → Repository）
- **命名規則の遵守**: ファイル名（`new.go`, `create.go`, `handler.go`）、構造体名、メソッド名がハンドラーガイドラインに完全に準拠
- **国際化の徹底**: すべてのユーザー向けメッセージが `i18n.T()` / `templates.T()` を使用、ja.toml/en.toml の両方が適切に更新されている
- **セキュリティ**: CSRF トークン、認証チェック、スペースメンバー検証が適切に実装されている
- **バリデーション設計**: 形式バリデーション→状態バリデーションの順序、Result 構造体パターンがガイドライン通り
- **既存パターンとの一貫性**: `renderNewForm` ヘルパーパターン、UseCase の入出力構造体パターンが他のハンドラーと統一されている
- **SQL クエリのスペーススコープ**: `draft_pages.sql` のクエリで `space_id` が WHERE 条件に含まれており、セキュリティガイドラインに準拠

ただし、**テストコードが一切追加されていない**点が大きな問題。PRガイドラインでは「実装とテストは同じPRに含める」「テストがない実装は原則としてマージしない」と明記されているため、テスト追加が必要。
