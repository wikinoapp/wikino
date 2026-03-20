# コードレビュー: suggestion-9-3

## レビュー情報

| 項目                       | 内容                                               |
| -------------------------- | -------------------------------------------------- |
| レビュー日                 | 2026-03-20                                         |
| 対象ブランチ               | suggestion-9-3                                     |
| ベースブランチ             | develop                                            |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md                   |
| 変更ファイル数             | 21 ファイル（実装 11 / テスト 5 / ドキュメント 5） |
| 変更行数（実装）           | +342 / -14 行                                      |
| 変更行数（テスト）         | +773 / -7 行                                       |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - 3層アーキテクチャ
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/page/edit.go`
- [x] `go/internal/handler/suggestion_page/handler.go`
- [x] `go/internal/handler/suggestion_page/update.go`
- [x] `go/internal/templates/pages/page/edit.templ`
- [x] `go/internal/templates/pages/page/edit_templ.go`（自動生成）
- [x] `go/internal/templates/path.go`
- [x] `go/internal/usecase/update_suggestion_page.go`
- [x] `go/internal/validator/suggestion_page.go`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`

### テストファイル

- [x] `go/internal/handler/page/edit_test.go`
- [x] `go/internal/handler/suggestion_page/main_test.go`
- [x] `go/internal/handler/suggestion_page/update_test.go`
- [x] `go/internal/usecase/update_suggestion_page_test.go`
- [x] `go/internal/validator/suggestion_page_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/suggestion.md`
- [x] `docs/plans/1_doing/write-usecase-refactoring.md`
- [x] `docs/reviews/suggestion-9-3-001.md`
- [x] `go/docs/architecture-guide.md`
- [x] `go/docs/testing-guide.md`

## ファイルごとのレビュー結果

### `go/internal/validator/suggestion_page_test.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約

**問題点・改善提案**:

- **不要な変数代入**: 71行目の `_ = db` と18行目の `db := testutil.GetTestDB()` が不要。このテストでは `db` を使用しておらず、`queries` のみ使用している

  ```go
  // 問題のあるコード（18行目、71行目）
  db := testutil.GetTestDB()
  // ...
  _ = db
  ```

  **修正案**:

  ```go
  // 18行目のdb取得を削除し、71行目の _ = db も削除する
  ```

  **対応方針**:
  - [x] 修正案の通り不要な `db` 変数を削除する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

### `go/internal/handler/suggestion_page/update.go`: ステータスチェックで下書きステータスも許可すべきか

**ステータス**: 要確認

**現状**:

71行目で編集提案のステータスがオープンの場合のみ更新を許可している:

```go
if detailOutput.Suggestion.Status != model.SuggestionStatusOpen {
    suggestionPath := string(templates.SuggestionShowPath(string(spaceIdentifier), int32(suggestionNumber)))
    http.Redirect(w, r, suggestionPath, http.StatusSeeOther)
    return
}
```

**提案**:

作業計画書のステータス定義によると、編集提案には「下書き」「オープン」「反映済み」「クローズ」の4つのステータスがある。現在の実装ではオープンのみ更新可能だが、下書きステータスの編集提案も更新可能にすべきではないか。

```go
// 提案コード
if detailOutput.Suggestion.Status != model.SuggestionStatusOpen &&
    detailOutput.Suggestion.Status != model.SuggestionStatusDraft {
    // ...
}
```

**メリット**:

- 下書きステータスの編集提案のページも編集・更新できるようになる
- 将来的に「下書きで提案を作成 → ページを編集して完成させる → オープンにする」というフローが実現可能

**トレードオフ**:

- 現時点で下書きステータスの編集提案を作成するUIが存在しない可能性がある（作成時にOpenで作られる場合はこの変更は不要）
- 初期リリースの範囲として意図的にOpenのみに制限している可能性がある

**対応方針**:

- [ ] 提案通り下書きステータスも許可する
- [ ] 現状のまま（理由を回答欄に記入）
- [x] その他（下の回答欄に記入）

**回答**:

```
確かに下書きステータスも許可すべきですね。作業計画書の要件や仕様なども併せて修正をお願いします
```

## 総合評価

**評価**: Approve

**総評**:

タスク9-3「編集提案を更新」アクションの実装として、適切な品質で実装されている。

**良い点**:

- **アーキテクチャの遵守**: 読み取りUseCase → Validator → 書き込みUseCaseの処理フローがガイドラインに正確に従っている
- **セキュリティ**: 認証チェック、スペースメンバー確認、ステータス確認、SuggestionPageの所属確認、DraftPageのリンク検証と多層の防御が実装されている
- **テストカバレッジ**: Handler/UseCase/Validatorの各レイヤーに独立したテストがあり、正常系・異常系を網羅している
- **既存パターンとの一貫性**: suggestion_apply/suggestion_closeハンドラーと同じパターン（URL解析、認証、データ取得、検証、リダイレクト）に従っている
- **edit.templの修正**: `_method=PATCH`の条件分岐を削除し、Method Overrideを一貫して使用するように修正した点は正しい
- **Validatorの設計**: Validatorが検証の過程で取得したDraftPageをResultに含めて返すパターンは、アーキテクチャガイドの推奨パターンに従っている

**指摘事項**:

- 必須修正1件（テストコード内の不要な変数代入）
- 設計確認1件（下書きステータスの編集提案の更新可否）
