# コードレビュー: usecase-3-10

## レビュー情報

| 項目                       | 内容                                                 |
| -------------------------- | ---------------------------------------------------- |
| レビュー日                 | 2026-03-30                                           |
| 対象ブランチ               | usecase-3-10                                         |
| ベースブランチ             | usecase-3-9                                          |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md |
| 変更ファイル数             | 12 ファイル                                          |
| 変更行数（実装）           | +113 / -100 行（実装 6 ファイル）                    |
| 変更行数（テスト）         | +271 / -459 行（テスト 4 ファイル）                  |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/sign_in/create.go`
- [x] `go/internal/handler/sign_in/handler.go`
- [x] `go/internal/usecase/create_user_session.go`
- [x] `go/internal/validator/sign_in.go`
- [x] `go/internal/templates/pages/sign_in/new.templ`

### テストファイル

- [x] `go/internal/handler/sign_in/create_test.go`
- [x] `go/internal/handler/sign_in/new_test.go`
- [x] `go/internal/usecase/create_user_session_test.go`
- [x] `go/internal/validator/sign_in_test.go`

### 設定・その他

- [x] `go/internal/templates/pages/sign_in/new_templ.go`（自動生成）
- [x] `docs/plans/1_doing/usecase-orchestration-refactor.md`

## ファイルごとのレビュー結果

### `go/internal/usecase/create_user_session.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - UseCase の設計方針
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約

**問題点・改善提案**:

- **[@go/docs/architecture-guide.md#ユースケース]**: コンストラクタの `signInValidator` が variadic 引数（`...*validator.SignInCreateValidator`）になっている。これは `new_test.go` など既存の呼び出し元で `NewCreateUserSessionUsecase(userSessionRepo)` のように Validator なしで呼び出されているケースを壊さないための対応と思われるが、作業計画書の方針「UseCase がバリデーション・認可・ビジネスロジック・永続化を統括する」と矛盾する。Validator が nil の場合、`executeSignIn` で nil pointer dereference が発生する。

  ```go
  // 問題のあるコード
  func NewCreateUserSessionUsecase(
      userSessionRepo *repository.UserSessionRepository,
      signInValidator ...*validator.SignInCreateValidator,
  ) *CreateUserSessionUsecase {
      uc := &CreateUserSessionUsecase{
          userSessionRepo: userSessionRepo,
      }
      if len(signInValidator) > 0 {
          uc.signInValidator = signInValidator[0]
      }
      return uc
  }
  ```

  **修正案**:

  Validator を必須引数にし、`new_test.go` など既存の呼び出し元も Validator を渡すように修正する。または、現段階では移行途中であることを考慮し、variadic を許容するなら、`executeSignIn` 内で nil チェックを追加する。ただし、作業計画書の方針に従うなら必須引数にすべき。

  ```go
  // 案A: 必須引数にする（推奨）
  func NewCreateUserSessionUsecase(
      userSessionRepo *repository.UserSessionRepository,
      signInValidator *validator.SignInCreateValidator,
  ) *CreateUserSessionUsecase {
      return &CreateUserSessionUsecase{
          userSessionRepo: userSessionRepo,
          signInValidator: signInValidator,
      }
  }
  ```

  **対応方針**:
  - [x] 案Aの通り必須引数にし、`new_test.go` 等の呼び出し元も修正する
  - [ ] 移行途中のため variadic のまま維持し、nil チェックを追加する
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

作業計画書タスク 3-10「create_user_session UseCase の移行」の実装として、方向性は正しく、主要な変更（Handler からバリデーション・2FA チェックの責務を UseCase に移動、Validator の戻り値を `(output, error)` パターンに変更、`session.FormErrors` → `model.ValidationError` への移行）は作業計画書の設計に沿っています。

テストコードは `setupHandler` ヘルパーの導入により大幅に簡略化され、可読性が向上しています。UseCase テストにバリデーションの異常系テストが追加され、カバレッジも十分です。

指摘は 1 件のみで、コンストラクタの variadic 引数パターンに関するものです。これは段階的移行の都合上意図的に行ったものかもしれませんが、nil pointer dereference のリスクがあるため確認が必要です。
