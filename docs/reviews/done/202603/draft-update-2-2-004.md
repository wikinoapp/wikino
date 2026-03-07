# コードレビュー: draft-update-2-2 (4回目)

## レビュー情報

| 項目                       | 内容                                  |
| -------------------------- | ------------------------------------- |
| レビュー日                 | 2026-03-05                            |
| 対象ブランチ               | draft-update-2-2                      |
| ベースブランチ             | draft-update-2-1                      |
| 作業計画書（指定があれば） | docs/plans/1_doing/draft-update.md    |
| 変更ファイル数             | 16 ファイル（ドキュメント5, 実装11）  |
| 変更行数（実装）           | +264 / -92 行（自動生成ファイル除く） |
| 変更行数（テスト）         | +255 / -2 行                          |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/draft_page/create.go`
- [x] `go/internal/handler/draft_page/handler.go`
- [x] `go/internal/handler/page/edit.go`
- [x] `go/internal/handler/page/update.go`
- [x] `go/internal/templates/pages/page/edit.templ`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`

### テストファイル

- [x] `go/internal/handler/draft_page/create_test.go`
- [x] `go/internal/handler/draft_page/update_test.go`

### 自動生成ファイル

- [x] `go/internal/templates/pages/page/edit_templ.go`

### ドキュメント

- [x] `docs/plans/1_doing/draft-update.md`
- [x] `docs/plans/1_doing/edit-suggestion.md`
- [x] `docs/reviews/draft-update-2-2-001.md`
- [x] `docs/reviews/draft-update-2-2-002.md`
- [x] `docs/reviews/draft-update-2-2-003.md`

## ファイルごとのレビュー結果

問題のあるファイルのみ記載します。

### `go/internal/templates/pages/page/edit.templ`: ManualSaveURL の重複生成

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - テンプレートデータ構造体とViewModelの関係
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - CSRF対策

**問題点・改善提案**:

- **ManualSaveURL のハンドラー側での組み立てとテンプレート内での `PageDraftPagePath` 呼び出しが混在している**: `EditPageData.ManualSaveURL` はハンドラー側で `templates.PageDraftPagePath(...)` を使って組み立てているが、テンプレート内の自動保存関連でも `templates.PageDraftPagePath(data.Space.Identifier.String(), data.Page.Number)` が直接呼ばれている（117行目、169行目）。同じパスを2つの方法で生成しているため、片方を変更した際に不整合が生じるリスクがある。

  ただし、117行目・169行目は今回の変更対象外（以前から存在）であり、今回追加の `ManualSaveURL` は手動保存ボタン専用のフィールドとして正当。現状のままでも問題はないが、将来的にパス生成をデータ構造体に集約することを推奨。

  **対応方針**:
  - [x] 今回は対応不要（既存コードの問題であり、別タスクで整理する）
  - [ ] `ManualSaveURL` を削除してテンプレート内で直接パスを生成するように統一
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク2-2（編集画面に「下書き保存」ボタンを追加）の実装として、作業計画書の要件を適切に満たしています。

**良い点**:

- `create.go` のハンドラー実装が既存の `update.go` と一貫したパターンに従っている（認証→スペース→メンバー→ページ→ポリシーチェック→ユースケース実行）
- `edit.go` のリンクデータ取得ロジックが `fetchEditLinkData` メソッドに抽出され、`update.go` の `renderEditWithErrors` からも再利用されている
- テストが正常系・異常系（未ログイン、不正なページ番号、スペース不存在、下書き不存在）を網羅的にカバーしている
- i18n対応が日英両方で行われている
- CSRFトークンが手動保存ボタンのリクエストに含まれている
- 作業計画書通り「現時点ではその場に留まる動作」（`StatusNoContent`）として実装され、下書き一覧画面完了後にリダイレクト先を変更する予定（3-3）との整合も取れている

**軽微な指摘（対応任意）**:

- テンプレート内のパス生成方法が `ManualSaveURL`（ハンドラー経由）と `PageDraftPagePath` 直接呼び出しで混在しているが、今回の変更範囲外の既存コードの問題であり、本PRの品質には影響しない
