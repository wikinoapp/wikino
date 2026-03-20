# コードレビュー: suggestion-9-1

## レビュー情報

| 項目                       | 内容                                     |
| -------------------------- | ---------------------------------------- |
| レビュー日                 | 2026-03-19                               |
| 対象ブランチ               | suggestion-9-1                           |
| ベースブランチ             | suggestion-8a-1                          |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md         |
| 変更ファイル数             | 25 ファイル（自動生成・レビューdoc含む） |
| 変更行数（実装）           | 約 +700 行                               |
| 変更行数（テスト）         | 約 +640 行                               |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/suggestion_change/index.go`
- [x] `go/internal/handler/suggestion_page_edit/handler.go`
- [ ] `go/internal/handler/suggestion_page_edit/new.go`
- [x] `go/internal/handler/suggestion_page_edit/create.go`
- [x] `go/internal/usecase/start_suggestion_page_edit.go`
- [x] `go/internal/viewmodel/suggestion.go`
- [x] `go/internal/templates/page_name.go`
- [x] `go/internal/templates/path.go`
- [x] `go/internal/templates/pages/suggestion_change/index.templ`
- [x] `go/internal/templates/pages/suggestion_page_edit/new.templ`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`

### テストファイル

- [x] `go/internal/handler/suggestion_page_edit/create_test.go`
- [x] `go/internal/handler/suggestion_page_edit/main_test.go`
- [x] `go/internal/usecase/start_suggestion_page_edit_test.go`

### 設定・その他

- [x] `go/docs/security-guide.md`
- [x] `go/docs/testing-guide.md`
- [x] `docs/plans/1_doing/suggestion.md`
- [x] `go/internal/templates/pages/suggestion_change/index_templ.go`（自動生成）
- [x] `go/internal/templates/pages/suggestion_page_edit/new_templ.go`（自動生成）
- [x] `docs/reviews/done/202603/suggestion-9-1-001.md`
- [x] `docs/reviews/done/202603/suggestion-9-1-002.md`
- [x] `docs/reviews/done/202603/suggestion-9-1-003.md`
- [x] `docs/reviews/done/202603/suggestion-9-1-004.md`

## ファイルごとのレビュー結果

### `go/internal/handler/suggestion_page_edit/new.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - メソッド命名規則
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - テンプレート関数の引数パターン

**問題点・改善提案**:

- **[@go/docs/handler-guide.md#メソッド命名規則]**: `New` ハンドラーのコメントに記載されているURLパターン `GET /s/{space_identifier}/suggestions/{suggestion_number}/page_edits/new` は正確だが、このハンドラーが表示するのは「新規作成フォーム」ではなく「確認画面」である。`New` メソッドの標準的な責務は「新規作成フォーム表示」だが、ここでは「既存の下書きがある場合の確認画面」として使用されている。URLパターンとしては `/new` が適切（POSTの前のGETフォーム）なので問題ないが、メソッドの実態と標準的な `New` の責務に若干のギャップがある点は認識しておくべき。ハンドラーガイドの「8種類のファイル名のみ」の原則に従った結果であり、これ自体は正しい判断。

  **修正案**: 特に修正不要。現在の実装はハンドラーガイドに準拠している。

  **対応方針**:
  - [ ] 現状のまま（問題なし）
  - [ ] コメントに「確認画面」である旨を追記する
  - [x] その他（下の回答欄に記入）

  **回答**:

  ```
  これは確かにちょっと違和感がありました。流れ的に create -> new となっている点も気になります。
  (普通は new -> create という流れのはず)
  new を使わずにshowで確認画面を表示すると良いかなと思ったのですが、良いリソース名はありますか？
  ```

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク9-1（編集提案ページ編集開始のUseCase・ハンドラー）の実装として、作業計画書の仕様を正確に実装している。

**良い点**:

- **アーキテクチャ準拠**: 3層アーキテクチャの依存関係ルールに忠実に従っている。Handler → UseCase → Repository の依存方向が正しく、Handler から Repository への直接依存がない
- **セキュリティ**: 認証チェック、スペースメンバー認可、CSRF保護、スペースIDスコープがすべて適切に実装されている
- **UseCase設計**: `StartSuggestionPageEditUsecase` が4つのケース（リンク済み下書きあり、通常下書きあり+非Force、通常下書きあり+Force、下書きなし）を明確にハンドリングしている。トランザクション管理も WithTx パターンに従っている
- **テストカバレッジ**: UseCase の4ケースすべてにテストがあり、ハンドラーテストも認証/認可/正常系/コンフリクトの代表ケースをカバーしている
- **I18n**: すべてのユーザー向けメッセージが国際化されており、ja/en両方が追加されている
- **既存コードとの一貫性**: 変更差分画面への「編集する」ボタン追加が、既存のテンプレートパターンに沿っている
- **ドキュメント更新**: セキュリティガイドとテストガイドの改善が含まれている
