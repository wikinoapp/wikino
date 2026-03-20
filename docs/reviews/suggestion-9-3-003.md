# コードレビュー: suggestion-9-3

## レビュー情報

| 項目                       | 内容                                    |
| -------------------------- | --------------------------------------- |
| レビュー日                 | 2026-03-20                              |
| 対象ブランチ               | suggestion-9-3                          |
| ベースブランチ             | develop                                 |
| 作業計画書（指定があれば） | docs/plans/1_doing/suggestion.md        |
| 変更ファイル数             | 22 ファイル（実装+テスト+ドキュメント） |
| 変更行数（実装）           | +852 / -8 行                            |
| 変更行数（テスト）         | +922 / -7 行                            |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/handler/suggestion_page/handler.go`
- [x] `go/internal/handler/suggestion_page/update.go`
- [x] `go/internal/usecase/update_suggestion_page.go`
- [x] `go/internal/validator/suggestion_page.go`
- [x] `go/internal/templates/path.go`
- [x] `go/internal/templates/pages/page/edit.templ`
- [x] `go/internal/templates/pages/page/edit_templ.go`（自動生成）
- [x] `go/internal/handler/page/edit.go`
- [x] `go/cmd/server/main.go`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`

### テストファイル

- [x] `go/internal/handler/suggestion_page/main_test.go`
- [x] `go/internal/handler/suggestion_page/update_test.go`
- [x] `go/internal/usecase/update_suggestion_page_test.go`
- [x] `go/internal/validator/suggestion_page_test.go`
- [x] `go/internal/handler/page/edit_test.go`

### ドキュメント

- [x] `docs/plans/1_doing/suggestion.md`
- [x] `docs/plans/1_doing/write-usecase-refactoring.md`
- [x] `docs/reviews/suggestion-9-3-001.md`
- [x] `docs/reviews/suggestion-9-3-002.md`
- [x] `go/docs/architecture-guide.md`
- [x] `go/docs/testing-guide.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。すべてのファイルがガイドラインに準拠しています。

## 設計改善の提案

### `go/internal/templates/path.go`: `SuggestionPagePath` での `url.PathEscape` の一貫性

**ステータス**: 要確認

**現状**:

`SuggestionPagePath` は `suggestionPageID` をエスケープせずに使用している:

```go
func SuggestionPagePath(spaceIdentifier string, suggestionNumber int32, suggestionPageID string) Path {
    return Path(fmt.Sprintf("/s/%s/suggestions/%d/suggestion_pages/%s", spaceIdentifier, suggestionNumber, suggestionPageID))
}
```

一方、同じファイルの `SuggestionPageEditShowPath` は `url.PathEscape` を使用している:

```go
func SuggestionPageEditShowPath(spaceIdentifier string, suggestionNumber int32, suggestionPageID string) Path {
    return Path(fmt.Sprintf("/s/%s/suggestions/%d/page_edits/%s", spaceIdentifier, suggestionNumber, url.PathEscape(suggestionPageID)))
}
```

**提案**:

IDはULID（英数字のみ）であるため実害はないが、同じパラメータ（`suggestionPageID`）の扱いを統一する。

選択肢:

- A: `SuggestionPagePath` にも `url.PathEscape` を追加する（防御的プログラミング）
- B: `SuggestionPageEditShowPath` の `url.PathEscape` を削除する（他のpath関数と一貫させる。`spaceIdentifier` 等も一切エスケープしていない）

**メリット**:

- パス関数間でパラメータの扱いが統一される
- コードベースの一貫性が向上する

**トレードオフ**:

- ULIDは英数字のみなので実害はなく、どちらを選んでも機能上の問題はない
- 案Bの方がコードベース全体の慣習に合っている（他のpath関数は一切エスケープしていない）

**対応方針**:

<!-- 開発者が回答を記入してください -->

- [x] 案A: `SuggestionPagePath` に `url.PathEscape` を追加する
- [ ] 案B: `SuggestionPageEditShowPath` の `url.PathEscape` を削除する
- [ ] 現状のまま（理由を回答欄に記入）
- [ ] その他（下の回答欄に記入）

**回答**:

```
（ここに回答を記入）
```

## 総合評価

**評価**: Approve

**総評**:

タスク9-3（「編集提案を更新」アクション）の実装として、設計・コード品質ともに優れている。

**良かった点**:

- **アーキテクチャガイドへの準拠**: Handler → 読み取りUseCase → Validator → 書き込みUseCase の処理フローが正確に踏襲されている
- **セキュリティ**: 認証チェック、スペースメンバー認可、ステータスチェック、SuggestionPage所属検証、DraftPageリンク検証と多層的な防御が適切に実装されている
- **テストカバレッジ**: Handler・UseCase・Validatorの各レイヤーに対するテストが充実しており、正常系・異常系（未ログイン、非メンバー、反映済み、クローズ済み、DraftPage不存在、リンク不一致）が網羅されている
- **命名規則**: ファイル名・構造体名・メソッド名がすべてガイドラインに準拠している
- **書き込みUseCaseの責務分離**: データ取得・検証をHandler/Validatorに任せ、UseCase はトランザクション内の永続化に専念する設計が正しく実装されている
- **作業計画書との整合性**: タスク9-3の要件（下書き/オープンステータスのみ更新可、DraftPageの`suggestion_page_id`は更新時にクリアしない）がすべて正しく実装されている
