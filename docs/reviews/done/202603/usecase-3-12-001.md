# コードレビュー: usecase-3-12

## レビュー情報

| 項目                       | 内容                                                         |
| -------------------------- | ------------------------------------------------------------ |
| レビュー日                 | 2026-03-30                                                   |
| 対象ブランチ               | usecase-3-12                                                 |
| ベースブランチ             | usecase-3-11                                                 |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md（3-12） |
| 変更ファイル数             | 18 ファイル                                                  |
| 変更行数（実装）           | +93 / -124 行                                                |
| 変更行数（テスト）         | +260 / -197 行                                               |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/password/handler.go`
- [x] `go/internal/handler/password/update.go`
- [x] `go/internal/handler/password_reset/create.go`
- [x] `go/internal/handler/password_reset/handler.go`
- [x] `go/internal/usecase/create_password_reset_token.go`
- [x] `go/internal/usecase/update_password_reset.go`
- [x] `go/internal/validator/password.go`
- [x] `go/internal/validator/password_reset.go`

### テストファイル

- [x] `go/internal/handler/password/edit_test.go`
- [x] `go/internal/handler/password/update_test.go`
- [x] `go/internal/handler/password_reset/create_test.go`
- [x] `go/internal/handler/password_reset/new_test.go`
- [x] `go/internal/usecase/create_password_reset_token_test.go`
- [x] `go/internal/usecase/update_password_reset_test.go`
- [x] `go/internal/validator/password_reset_test.go`
- [x] `go/internal/validator/password_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/usecase-orchestration-refactor.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。すべてのファイルがガイドラインに従っており、作業計画書の設計通りに実装されています。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 3-12 の実装は、作業計画書の設計に忠実かつ丁寧に行われている。

**良い点**:

- **Handler が薄い Adapter になっている**: `password/update.go` と `password_reset/create.go` の Handler は、フォームデータのパース → UseCase 呼び出し → エラーハンドリング（`model.AsValidationError` パターン）→ レスポンスという HTTP 入出力に徹した構成になっている
- **UseCase がオーケストレーターとして機能している**: `update_password_reset.go` は 1. バリデーション → 2. ハッシュ化（トランザクション前）→ 3. トランザクション（永続化のみ）という処理順序が明確で、「Execute 内にロジックを直接書かない」ルールに従い `updatePassword` メソッドに永続化を分離している
- **Validator の Result 型を廃止し `(data, error)` パターンに統一**: `PasswordUpdateValidator` は `(*PasswordUpdateValidateOutput, error)` を、`PasswordResetCreateValidator` は `error` のみを返す Go 慣習的なパターンに変更されている
- **センチネルエラーの除去**: `ErrTokenNotFound`, `ErrTokenUsed`, `ErrTokenExpired` を廃止し、`validateToken` 内で `model.ValidationError` に i18n メッセージを直接設定する方式に統一。Handler でのエラー判別が `errors.As` ではなく `model.AsValidationError` のみで済むようになった
- **テストが充実**: UseCase テスト（`update_password_reset_test.go`）にバリデーションエラーのテストケースが追加され、正常系・パスワード不一致・無効トークンの 3 パターンが `t.Run` でサブテスト化されている
- **不要なコードの削除**: `password_test.go` の `containsSubstring` ヘルパーを標準ライブラリ `strings.Contains` に置き換え、テストコードの可読性が向上した
- **DI 構成の整理**: `main.go` で Validator の生成と Handler への注入を適切に移動し、Handler → Validator の依存を排除している
