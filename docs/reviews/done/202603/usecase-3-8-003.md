# コードレビュー: usecase-3-8

## レビュー情報

| 項目                       | 内容                                                 |
| -------------------------- | ---------------------------------------------------- |
| レビュー日                 | 2026-03-30                                           |
| 対象ブランチ               | usecase-3-8                                          |
| ベースブランチ             | usecase-3-7                                          |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md |
| 変更ファイル数             | 17 ファイル（docs 除く）                             |
| 変更行数（実装）           | +356 / -132 行                                       |
| 変更行数（テスト）         | +363 / -81 行                                        |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/suggestion_close/create.go`
- [x] `go/internal/handler/suggestion_close/handler.go`
- [x] `go/internal/handler/suggestion_page_edit/create.go`
- [x] `go/internal/policy/topic.go`
- [x] `go/internal/policy/topic_admin.go`
- [x] `go/internal/policy/topic_guest.go`
- [x] `go/internal/policy/topic_member.go`
- [x] `go/internal/policy/topic_owner.go`
- [x] `go/internal/usecase/close_suggestion.go`
- [x] `go/internal/usecase/start_suggestion_page_edit.go`

### テストファイル

- [x] `go/internal/handler/suggestion_close/create_test.go`
- [x] `go/internal/handler/suggestion_page_edit/create_test.go`
- [x] `go/internal/usecase/close_suggestion_test.go`
- [x] `go/internal/usecase/start_suggestion_page_edit_test.go`

### 設定・その他

- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`

## ファイルごとのレビュー結果

問題のあるファイルのみ記載。問題がないファイルは上記チェックボックスにチェック済み。

（全ファイル問題なし）

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 3-8（close_suggestion, start_suggestion_page_edit UseCase の移行）が作業計画書の設計方針に正確に沿って実装されている。

**良い点**:

- **パターンの一貫性**: フェーズ 3 の他タスク（3-1 ~ 3-7）で確立された `fetchData → authorize → checkStatus → 永続化` のパターンに正確に従っている
- **Handler の薄さ**: Handler から Policy・読み取り UseCase の依存が完全に除去され、HTTP の入出力変換に徹している。`handleCreateError` メソッドによる `errors.As` パターンも既存ハンドラーと統一されている
- **べき等性の保持**: `close_suggestion` のクローズ済み提案に対するべき等な成功レスポンスが UseCase 内の `checkStatusForClose` で適切に処理されている
- **振る舞いの保持**: 旧 Handler で行われていたステータスチェック・権限チェック・リダイレクト先がすべて新 UseCase + Handler の `handleCreateError` で同一の振る舞いとして再現されている
- **テストカバレッジ**: UseCase テストに正常系（クローズ成功、べき等成功）と異常系（存在しないスペース、権限なし、ステータス不正）が網羅されている。Handler テストも書き込みテストで `GetTestDB()` / `BuilderDB` パターンに正しく切り替えられている
- **Policy の追加**: `CanEditSuggestionPage` が全ロール（Owner, Admin, Member, Guest）に追加され、旧 Handler の「スペースメンバーであれば許可」の振る舞いが正確に再現されている
- **国際化**: 新メッセージ `suggestion_page_edit_error_not_open` が ja/en 両方に追加されている
