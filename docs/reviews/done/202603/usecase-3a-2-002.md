# コードレビュー: usecase-3a-2

## レビュー情報

| 項目                       | 内容                                                 |
| -------------------------- | ---------------------------------------------------- |
| レビュー日                 | 2026-03-30                                           |
| 対象ブランチ               | usecase-3a-2                                         |
| ベースブランチ             | usecase-3a-1                                         |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md |
| 変更ファイル数             | 38 ファイル                                          |
| 変更行数（実装）           | +447 / -310 行                                       |
| 変更行数（テスト）         | +455 / -231 行                                       |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - 3層アーキテクチャ、依存関係ルール
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - ハンドラーガイドライン
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/.golangci.yml`
- [x] `go/cmd/server/main.go`
- [x] `go/internal/model/errors.go`
- [x] `go/internal/repository/user_session_repository.go`
- [x] `go/internal/repository/user_two_factor_auth_repository.go`
- [x] `go/internal/handler/sign_in_two_factor/handler.go`
- [x] `go/internal/handler/sign_in_two_factor/create.go`
- [x] `go/internal/handler/sign_in_two_factor_recovery/handler.go`
- [x] `go/internal/handler/sign_in_two_factor_recovery/create.go`
- [x] `go/internal/handler/draft_page/show.go`
- [x] `go/internal/handler/page/edit.go`
- [x] `go/internal/handler/page_backlink_list/show.go`
- [x] `go/internal/handler/page_backlinks/show.go`
- [x] `go/internal/handler/page_link_list/show.go`
- [x] `go/internal/handler/suggestion/edit.go`
- [x] `go/internal/handler/suggestion/show.go`
- [x] `go/internal/handler/suggestion_comment_edit/edit.go`
- [x] `go/internal/usecase/create_two_factor_session.go`
- [x] `go/internal/usecase/create_recovery_code_session.go`
- [x] `go/internal/usecase/recovery_code.go`
- [x] `go/internal/usecase/consume_recovery_code.go`（削除）
- [x] `go/internal/usecase/get_backlink_list.go`
- [x] `go/internal/usecase/get_link_list.go`
- [x] `go/internal/usecase/get_page_backlinks.go`
- [x] `go/internal/usecase/get_page_detail.go`
- [x] `go/internal/usecase/get_suggestion_detail.go`
- [x] `go/internal/validator/sign_in_two_factor.go`
- [x] `go/internal/validator/sign_in_two_factor_recovery.go`

### テストファイル

- [x] `go/internal/handler/sign_in_two_factor/create_test.go`
- [x] `go/internal/handler/sign_in_two_factor/new_test.go`
- [x] `go/internal/handler/sign_in_two_factor_recovery/create_test.go`
- [x] `go/internal/handler/sign_in_two_factor_recovery/new_test.go`
- [x] `go/internal/usecase/consume_recovery_code_test.go`（削除）
- [x] `go/internal/usecase/create_two_factor_session_test.go`
- [x] `go/internal/usecase/create_recovery_code_session_test.go`
- [x] `go/internal/validator/sign_in_two_factor_test.go`
- [x] `go/internal/validator/sign_in_two_factor_recovery_test.go`

### 設定・その他

- [x] `go/internal/testutil/user_builder.go`
- [x] `docs/plans/1_doing/usecase-orchestration-refactor.md`
- [x] `docs/reviews/usecase-3a-2-001.md`

## ファイルごとのレビュー結果

問題なし。すべてのファイルがガイドラインに従っています。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

このPRは作業計画書のフェーズ3a-2（depguard禁止ルール追加）に加え、以下の3つの大きな変更を含んでいます：

1. **depguard ルール追加**: Handler → Policy, Validator の依存を禁止するルールが追加され、アーキテクチャの強制が実現されている
2. **2FA関連UseCaseの統合**: `sign_in_two_factor` と `sign_in_two_factor_recovery` のHandlerからValidator・UseCaseの直接呼び出しを排除し、新しい統合UseCase（`CreateTwoFactorSessionUsecase`, `CreateRecoveryCodeSessionUsecase`）に集約している
3. **読み取りUseCaseへの認可チェック移動**: `get_page_detail`, `get_backlink_list`, `get_link_list`, `get_page_backlinks`, `get_suggestion_detail` の5つの読み取りUseCaseにPolicy呼び出しを統合し、Handlerから`policy`パッケージの依存を除去している

**良い点**:

- **作業計画書の方針に忠実**: UseCase がオーケストレーターとして機能し、Handler は HTTP の入出力変換に専念する設計が一貫して適用されている
- **エラーハンドリングの一貫性**: `model.AsValidationError` / `model.AsAppError` パターンが全Handlerで統一されている
- **テストカバレッジ**: 新しいUseCaseに対して正常系・異常系の両方が網羅されている。特に `CreateRecoveryCodeSessionUsecase` でリカバリーコードの消費確認やトランザクション整合性の検証が行われている
- **nil安全性**: 全ての読み取りUseCaseで `spaceMember == nil` の早期リターンが存在し（`get_suggestion_detail` は `if spaceMember != nil` ガード）、`policy.NewTopicPolicy` のnilパニックリスクがない
- **UseCase設計ルールの遵守**: `CreateRecoveryCodeSessionUsecase` でトランザクション開始前にバリデーション、トランザクション内は永続化のみというルールが守られている
- **不要コードの削除**: `ConsumeRecoveryCodeUsecase` が `CreateRecoveryCodeSessionUsecase` に統合され、単機能UseCaseの過剰分割が解消されている
