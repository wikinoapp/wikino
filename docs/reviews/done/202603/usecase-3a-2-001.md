# コードレビュー: usecase-3a-2

## レビュー情報

| 項目                       | 内容                                                 |
| -------------------------- | ---------------------------------------------------- |
| レビュー日                 | 2026-03-30                                           |
| 対象ブランチ               | usecase-3a-2                                         |
| ベースブランチ             | usecase-3a-1                                         |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md |
| 変更ファイル数             | 31 ファイル                                          |
| 変更行数（実装）           | +374 / -261 行                                       |
| 変更行数（テスト）         | +85 / -101 行                                        |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/.golangci.yml`
- [x] `go/cmd/server/main.go`
- [x] `go/internal/model/errors.go`
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
- [x] `go/internal/validator/sign_in_two_factor_test.go`
- [x] `go/internal/validator/sign_in_two_factor_recovery_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/usecase-orchestration-refactor.md`

## ファイルごとのレビュー結果

### `go/internal/usecase/create_two_factor_session.go` / `go/internal/usecase/create_recovery_code_session.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - UseCase のルール
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - レイヤーごとのテストカバレッジ

**問題点・改善提案**:

- **[@go/docs/testing-guide.md#レイヤーごとのテストカバレッジ]**: 新規 UseCase ファイルに対応するテストファイルがない

  テストガイドでは「新しい実装ファイルを追加した場合は、対応するテストファイルも必ず作成する」とされており、UseCase のテストは「必須」レベルです。`create_two_factor_session.go` と `create_recovery_code_session.go` の 2 つの新規 UseCase に対してテストが存在しません。

  Handler テストが間接的にカバーしている可能性はありますが、テストガイドに「Handler テストだけでは UseCase のロジックを十分にテストできない」と明記されています。特に `CreateRecoveryCodeSessionUsecase` はリカバリーコード消費 + セッション作成という複合操作があり、UseCase 単体のテストが望ましいです。

  **修正案**:

  `go/internal/usecase/create_two_factor_session_test.go` と `go/internal/usecase/create_recovery_code_session_test.go` を作成し、正常系・異常系（バリデーションエラー、AppError、システムエラー）のテストを追加する。

  **対応方針**:
  - [x] テストファイルを追加する
  - [ ] Handler テストで十分カバーされているため追加しない（理由を回答欄に記入）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/usecase/create_recovery_code_session.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - 書き込み UseCase のルール

**問題点・改善提案**:

- **[@go/docs/architecture-guide.md#書き込みUseCaseのルール]**: `consumeRecoveryCode` と `createSession` がトランザクションで保護されていない

  `CreateRecoveryCodeSessionUsecase.Execute` はリカバリーコードの消費（ステップ 2）とセッション作成（ステップ 3）の 2 つの書き込み操作を行っています。ステップ 2 が成功しステップ 3 が失敗した場合、リカバリーコードが消費されたのにセッションが作成されない不整合が発生する可能性があります。

  作業計画書の設計（検討事項 4）では「トランザクション開始後はデータの取得や計算処理を行わない（永続化のみ）」を維持するとされており、現在の実装はこれに沿っています（バリデーション → 永続化の順）。ただし、2 つの永続化操作がアトミックでない点が気になります。

  **修正案**:

  `db *sql.DB` を依存に追加し、ステップ 2 と 3 をトランザクションで囲む。あるいは、セッション作成が失敗するケースが実質的にありえないと判断できるのであれば、現状のままでも可。

  **対応方針**:
  - [x] トランザクションを追加する
  - [ ] セッション作成失敗は極めて稀であり、リカバリーコード消費がロールバックされなくてもセキュリティ上の問題は軽微なため現状のまま
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/cmd/server/main.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - UseCase

**問題点・改善提案**:

- **未使用コードの残存の可能性**: `consumeRecoveryCodeUC` の生成行が削除されていますが、`ConsumeRecoveryCodeUsecase` 自体がまだ `consume_recovery_code.go` に存在しています。この UseCase を直接使用する箇所がなくなった場合（`CreateRecoveryCodeSessionUsecase` が内部で `removeRecoveryCode` 関数を直接呼び出しており、`ConsumeRecoveryCodeUsecase` を経由していない）、`consume_recovery_code.go` が未使用コードとなっている可能性があります。

  **修正案**:

  `ConsumeRecoveryCodeUsecase` が他の呼び出し元から使用されていなければ、`consume_recovery_code.go`（と対応するテスト `consume_recovery_code_test.go`）を削除する。`removeRecoveryCode` ヘルパー関数は別ファイルに残す。

  **対応方針**:
  - [x] `ConsumeRecoveryCodeUsecase` を削除し、`removeRecoveryCode` のみ残す
  - [ ] 将来的に使う可能性があるため残す（理由を回答欄に記入）
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

タスク 3a-2 の主目的である depguard ルールの追加（Handler → Policy, Validator の禁止）は正しく実装されています。加えて、実際に depguard 違反となる Handler ファイルの修正が行われており、以下の 2 つの方針で対処されています：

1. **読み取り UseCase（GET ハンドラー用）**: Policy チェックを UseCase 内に移動し、結果を `CanUpdatePage` 等のブール値フィールドとして Output に含める
2. **書き込み UseCase（POST ハンドラー用）**: Validator・Policy の呼び出しを UseCase に統合し、Handler は `errors.As` パターンでエラーハンドリングのみ行う

いずれも作業計画書の設計方針に沿っており、Handler から `policy` / `validator` パッケージの import が完全に除去されています。Validator の Result 型廃止・`(data, error)` 返しへの移行も作業計画書通りです。

指摘事項は 3 件（いずれも軽微〜確認レベル）で、必須対応はありません：

- 新規 UseCase のテストファイル不足
- `CreateRecoveryCodeSessionUsecase` のトランザクション保護の検討
- 未使用となった `ConsumeRecoveryCodeUsecase` の残存
