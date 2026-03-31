# コードレビュー: usecase-3a-1

## レビュー情報

| 項目                       | 内容                                                 |
| -------------------------- | ---------------------------------------------------- |
| レビュー日                 | 2026-03-30                                           |
| 対象ブランチ               | usecase-3a-1                                         |
| ベースブランチ             | usecase-3-12                                         |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md |
| 変更ファイル数             | 6 ファイル                                           |
| 変更行数（実装）           | +0 / -77 行（`form_errors.go` 削除）                 |
| 変更行数（ドキュメント）   | +57 / -58 行                                         |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/session/form_errors.go`（削除）

### ドキュメント

- [x] `docs/plans/1_doing/usecase-orchestration-refactor.md`
- [x] `go/docs/architecture-guide.md`
- [x] `go/docs/security-guide.md`
- [x] `go/docs/templ-guide.md`
- [x] `go/docs/validation-guide.md`

## ファイルごとのレビュー結果

### `go/docs/i18n-guide.md`（変更対象から漏れている）

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションの API 命名
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド

**問題点・改善提案**:

- **[設計との整合性]**: `go/docs/i18n-guide.md` に `AddFieldError`（旧 API メソッド名）が 4 箇所残存している。他のガイドラインドキュメント（security-guide, validation-guide, templ-guide, architecture-guide）はすべて `AddField` に更新されているが、i18n-guide が漏れている

  該当箇所（168 行目、234 行目、311 行目、314 行目）:

  ```go
  // 現在のコード
  errors.AddFieldError("email", i18n.T(ctx, "password_reset_email_required"))
  ```

  **修正案**:

  `model.ValidationError` の API に合わせて `AddFieldError` → `AddField` に更新し、変数名も他のガイドラインと統一する:

  ```go
  ve.AddField("email", i18n.T(ctx, "password_reset_email_required"))
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 修正案の通り `AddFieldError` → `AddField` に更新し、変数名も統一する
  - [ ] 変数名はそのままで `AddFieldError` → `AddField` のみ更新する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計との整合性チェック

### 作業計画書タスク 3a-1 との整合性

タスク 3a-1 の要件:

| 要件                                                       | 状態                                                         |
| ---------------------------------------------------------- | ------------------------------------------------------------ |
| `internal/templates/components/form_errors.templ` を変更   | ✅ （ベースブランチで対応済み）                              |
| 残存する全テンプレートの `session.FormErrors` 参照を除去   | ✅ （ベースブランチで対応済み、Go ソースコード内に残存なし） |
| `internal/session/form_errors.go` を削除                   | ✅                                                           |
| templ generate を実行し、生成ファイルを更新                | ✅ （ベースブランチで対応済み）                              |
| ガイドラインドキュメントの `session.FormErrors` 参照を更新 | ⚠️ `go/docs/i18n-guide.md` に旧 API メソッド名 4 箇所が残存  |

### 補足

- `docs/specs/page/move.md` に `session.FormErrors` への参照が 1 箇所残存しているが、仕様書の更新は全体リファクタリング完了時に行うものであり、本タスクの範囲外と判断する

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Request Changes

**総評**:

`session.FormErrors` の完全除去というタスクの趣旨は正しく実装されている。Go ソースコード（`go/internal/`）からの `session.FormErrors` 参照は完全に除去されており、`form_errors.go` ファイルも適切に削除されている。ガイドラインドキュメント（architecture-guide, security-guide, templ-guide, validation-guide）の更新も一貫して行われている。

唯一の問題点は `go/docs/i18n-guide.md` の旧 API メソッド名（`AddFieldError`）の残存であり、他のガイドラインと同様に `AddField` に更新する必要がある。軽微な修正で対応可能。
