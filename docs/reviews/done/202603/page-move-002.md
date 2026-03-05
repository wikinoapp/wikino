# コードレビュー: page-move

## レビュー情報

| 項目                       | 内容                                 |
| -------------------------- | ------------------------------------ |
| レビュー日                 | 2026-03-05                           |
| 対象ブランチ               | page-move                            |
| ベースブランチ             | develop                              |
| 作業計画書（指定があれば） | docs/plans/1_doing/page-move.md      |
| 変更ファイル数             | 34 ファイル                          |
| 変更行数（実装）           | +1,074 行（Go実装 + Rails + i18n等） |
| 変更行数（テスト）         | +715 行                              |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版開発ガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@rails/CLAUDE.md](/workspace/rails/CLAUDE.md) - Rails版開発ガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/db/queries/pages.sql`
- [x] `go/internal/handler/page_move/handler.go`
- [x] `go/internal/handler/page_move/new.go`
- [x] `go/internal/handler/page_move/create.go`
- [x] `go/internal/handler/page_move/validator.go`
- [x] `go/internal/middleware/reverse_proxy.go`
- [x] `go/internal/policy/topic.go`
- [x] `go/internal/policy/topic_admin.go`
- [x] `go/internal/policy/topic_guest.go`
- [x] `go/internal/policy/topic_member.go`
- [x] `go/internal/policy/topic_owner.go`
- [x] `go/internal/query/pages.sql.go`
- [x] `go/internal/repository/page.go`
- [x] `go/internal/templates/pages/page_move/new.templ`
- [x] `go/internal/templates/pages/page_move/new_templ.go`
- [x] `go/internal/templates/path.go`
- [x] `go/internal/usecase/move_page.go`
- [x] `go/internal/viewmodel/page.go`
- [x] `go/internal/viewmodel/topic.go`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`
- [x] `rails/app/components/dropdowns/page_actions_component.html.erb`
- [x] `rails/config/locales/verbs.en.yml`
- [x] `rails/config/locales/verbs.ja.yml`
- [x] `rails/config/routes.rb`

### テストファイル

- [x] `go/internal/handler/page_move/main_test.go`
- [x] `go/internal/handler/page_move/new_test.go`
- [x] `go/internal/handler/page_move/validator_test.go`
- [x] `go/internal/usecase/move_page_test.go`
- [x] `go/internal/policy/topic_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/page-move.md`
- [x] `docs/plans/1_doing/edit-suggestion.md`
- [x] `docs/reviews/page-move-001.md`

## ファイルごとのレビュー結果

### `go/internal/handler/page_move/validator.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーション
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティ

**問題点・改善提案**:

- **整数パースのオーバーフロー**: 手動で `int32` をパースしているが（60-65行目）、非常に長い数字文字列が入力された場合に `int32` がオーバーフローし、意図しないトピック番号に変換される可能性がある。DB検索で該当するトピックが見つからなければ実害はないが、防御的プログラミングの観点から修正が望ましい。

  ```go
  // 問題のあるコード
  var destTopicNumber int32
  for _, c := range input.DestTopicNumber {
      if c < '0' || c > '9' {
          formErrors.AddField("dest_topic", i18n.T(ctx, "page_move_error_topic_required"))
          return &CreateValidatorResult{FormErrors: formErrors}
      }
      destTopicNumber = destTopicNumber*10 + int32(c-'0')
  }
  ```

  **修正案**:

  ```go
  parsed, err := strconv.ParseInt(input.DestTopicNumber, 10, 32)
  if err != nil {
      formErrors.AddField("dest_topic", i18n.T(ctx, "page_move_error_topic_required"))
      return &CreateValidatorResult{FormErrors: formErrors}
  }
  destTopicNumber := int32(parsed)
  ```

  **対応方針**:
  - [x] `strconv.ParseInt` に変更する
  - [ ] 現状のまま（DB検索で安全性が担保されるため）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/handler/page_move/new.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン

**問題点・改善提案**:

- **`CurrentPageName` の値**: `renderMoveForm` 内で `CurrentPageName: templates.PageNamePageEdit` を使用している（162行目）。ページ移動画面はページ編集画面とは異なるため、サイドバーのハイライト表示が「編集」のままになる。ページ移動用の `PageName` 定数を追加するか、意図的に編集画面と同じハイライトにしているなら問題ない。

  **修正案**:

  `PageNamePageMove` 定数を `templates/path.go` に追加し、使用する。もしくは既存の `PageNamePageEdit` で意図的ならコメントで理由を明示する。

  **対応方針**:
  - [x] `PageNamePageMove` を追加してサイドバーの表示を区別する
  - [ ] 現状のまま（編集と同じハイライトが意図的）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/handler/page_move/create.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド

**問題点・改善提案**:

- **Create ハンドラーのテストが存在しない**: `new_test.go` と `validator_test.go` は存在するが、`create.go` に対応するハンドラーテスト（正常系のリダイレクト、バリデーションエラー時のフォーム再表示など）が存在しない。作業計画書では「想定ファイル数: テスト ~4 ファイル」と記載されているが、3ファイルのみ。Create ハンドラーの統合テストを追加することが望ましい。

  **修正案**:

  `create_test.go` を追加し、以下のケースをテスト:
  - 正常系: 移動成功後のリダイレクトとフラッシュメッセージ
  - 異常系: バリデーションエラー時の 422 レスポンスとフォーム再表示

  **対応方針**:
  - [x] `create_test.go` を追加する
  - [ ] 現状のバリデーションテストで十分と判断する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Comment

**総評**:

全体的に作業計画書の仕様に忠実に実装されており、ガイドラインへの準拠も良好。

**良かった点**:

- 3層アーキテクチャの依存関係ルールが正しく守られている
- SQLクエリに `space_id` スコープが含まれ、セキュリティガイドラインに準拠
- TopicPolicy の各実装（owner/admin/member/guest）とテストが網羅的
- I18n対応が日英両方で完備
- バリデーションが形式チェック→状態チェックの順序で段階的に実行
- `FormErrors.HasErrors()` のnil安全な設計が活用されている
- templ テンプレートが構造体ベースのデータ受け渡しパターンに従っている
- 作業計画書の権限設計（移動元 CanUpdatePage / 移動先 CanCreatePage）が正確に実装されている

**指摘事項のサマリー**:

- 必須対応: 0件
- 推奨対応: 1件（整数パースのオーバーフロー対策）
- 要確認: 2件（CurrentPageName の意図確認、Create ハンドラーテストの追加検討）
- 設計との乖離: 0件
