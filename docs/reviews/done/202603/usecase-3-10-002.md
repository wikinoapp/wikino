# コードレビュー: usecase-3-10

## レビュー情報

| 項目                       | 内容                                                 |
| -------------------------- | ---------------------------------------------------- |
| レビュー日                 | 2026-03-30                                           |
| 対象ブランチ               | usecase-3-10                                         |
| ベースブランチ             | usecase-3-9                                          |
| 作業計画書（指定があれば） | docs/plans/1_doing/usecase-orchestration-refactor.md |
| 変更ファイル数             | 18 ファイル                                          |
| 変更行数（実装）           | +231 / -100 行                                       |
| 変更行数（テスト）         | +288 / -476 行                                       |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テストガイド
- [@go/docs/coding-guide.md](/workspace/go/docs/coding-guide.md) - コーディング規約
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/sign_in/create.go`
- [x] `go/internal/handler/sign_in/handler.go`
- [ ] `go/internal/usecase/create_user_session.go`
- [x] `go/internal/validator/sign_in.go`
- [x] `go/internal/templates/pages/sign_in/new.templ`
- [x] `go/internal/templates/pages/sign_in/new_templ.go`

### テストファイル

- [x] `go/internal/handler/sign_in/create_test.go`
- [x] `go/internal/handler/sign_in/new_test.go`
- [ ] `go/internal/usecase/create_user_session_test.go`
- [x] `go/internal/validator/sign_in_test.go`
- [x] `go/internal/handler/account/create_test.go`
- [x] `go/internal/handler/sign_in_two_factor/create_test.go`
- [x] `go/internal/handler/sign_in_two_factor/new_test.go`
- [x] `go/internal/handler/sign_in_two_factor_recovery/create_test.go`
- [x] `go/internal/handler/sign_in_two_factor_recovery/new_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/usecase-orchestration-refactor.md`

## ファイルごとのレビュー結果

### `go/internal/usecase/create_user_session.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - UseCase の設計

**問題点・改善提案**:

- **[@go/docs/architecture-guide.md#ユースケース]**: `CreateUserSessionInput` が「サインインフロー用」と「認証済みフロー用」の 2 つの異なるユースケースを 1 つの Input で表現しており、条件分岐で振る舞いが変わる設計になっている

  ```go
  // 現状: 1つの Input が2つの異なるフローを持つ
  type CreateUserSessionInput struct {
      // サインインフロー用（メールアドレス/パスワードによるバリデーション）
      Email    string
      Password string
      // 認証済みフロー用（2FA、アカウント作成後のセッション作成）
      UserID model.UserID
      // 共通フィールド
      IPAddress string
      UserAgent string
  }
  ```

  `UserID` が空か否かでフローが分岐する暗黙的な契約があり、呼び出し側が「Email/Password を渡す場合は UserID を空にする」「UserID を渡す場合は Email/Password は無視される」というルールを知っている必要がある。また、`signInValidator` が `nil` で渡される可能性がある（認証済みフロー専用の呼び出し元が `nil` を渡している: `account/create_test.go`, `sign_in_two_factor/create_test.go` 等）が、サインインフローが誤って呼ばれた場合にパニックする。

  **修正案**:

  作業計画書の設計方針（UseCase がオーケストレーターとなる）に照らすと、「サインインフロー（Email/Password → バリデーション → セッション作成）」と「認証済みフロー（UserID → セッション作成のみ）」は別のユースケースとして分離するのが自然。ただし、この PR のスコープ（タスク 3-10: create_user_session UseCase の移行）では段階的なリファクタリングを進めている最中であり、この統合は意図的な中間状態である可能性もある。

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [ ] 現状のまま（中間状態として許容、後のタスクで分離予定）
  - [x] サインインフロー専用の UseCase に分離する（例: `SignInUsecase`）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/usecase/create_user_session_test.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/testing-guide.md](/workspace/go/docs/testing-guide.md) - テスト戦略

**問題点・改善提案**:

- **[@go/docs/testing-guide.md#テストのベストプラクティス]**: 認証済みフロー（`UserID` を直接指定してセッションを作成する既存のフロー）のテストが削除されている

  変更前は `UserID` を指定してセッション作成するテスト（空 IP/UA のテスト、存在しない UserID のテスト）があったが、変更後はすべてサインインフロー（Email/Password）のテストに置き換わっている。UseCase が 2 つのフロー（サインインフロー・認証済みフロー）をサポートしている以上、認証済みフローのテストも必要。

  **修正案**:

  認証済みフロー（`UserID` を直接指定）でセッションを作成するテストケースを追加する:

  ```go
  t.Run("正常系: UserID指定でセッションを作成できる（認証済みフロー）", func(t *testing.T) {
      // UserIDを直接指定してExecuteを呼び出す
      output, err := uc.Execute(ctx, CreateUserSessionInput{
          UserID:    userID,
          IPAddress: "192.168.1.1",
          UserAgent: "Mozilla/5.0",
      })
      // ...
  })
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [ ] 認証済みフローのテストを追加する
  - [x] 前述の UseCase 分離と合わせて対応する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

### `go/internal/templates/pages/sign_in/new.templ`: `FormErrors` コンポーネントの再利用

**ステータス**: 要確認

**現状**:

既存の `components.FormErrors` コンポーネント（`session.FormErrors` を受け取る）を使わず、グローバルエラー表示のロジックをテンプレート内にインラインで展開している。

```templ
// new.templ（変更後）
if data.FormErrors != nil && len(data.FormErrors.Global) > 0 {
    for _, errorMsg := range data.FormErrors.Global {
        <div class="alert-destructive">
            <section>{ errorMsg }</section>
        </div>
    }
}
```

**提案**:

`components.FormErrors` コンポーネントを `model.ValidationError` を受け取るように更新し、テンプレート内でインライン展開せずにコンポーネントを再利用する。作業計画書に「`session.FormErrors` は廃止し、`model.ValidationError` に置き換える」とあるため、`FormErrors` コンポーネントの型を `*model.ValidationError` に変更するのが自然。

```templ
// components/form_errors.templ（変更案）
templ FormErrors(formErrors *model.ValidationError) {
    if formErrors != nil && len(formErrors.Global) > 0 {
        for _, errorMsg := range formErrors.Global {
            <div class="alert-destructive">
                <section>{ errorMsg }</section>
            </div>
        }
    }
}

// pages/sign_in/new.templ
@components.FormErrors(data.FormErrors)
```

**メリット**:

- グローバルエラー表示のロジックが一箇所に集約される
- 他のページテンプレートでも同じコンポーネントを再利用できる
- 表示スタイルの変更が 1 箇所で済む

**トレードオフ**:

- `components.FormErrors` を使用している他のテンプレートも同時に変更が必要（ただし作業計画書に沿った作業なので問題なし）
- 別タスクで `session.FormErrors` を全面的に `model.ValidationError` に置き換える際にまとめて対応する方が合理的かもしれない

**対応方針**:

<!-- 開発者が回答を記入してください -->

- [x] この PR で `components.FormErrors` を `model.ValidationError` 対応に変更する
- [ ] 別タスクでまとめて対応する（現状のインライン展開を維持）
- [ ] その他（下の回答欄に記入）

**回答**:

```
（ここに回答を記入）
```

## 総合評価

**評価**: Comment

**総評**:

作業計画書（タスク 3-10: create_user_session UseCase の移行）の方針に沿って、サインインのバリデーション処理を Handler から UseCase に移動するリファクタリングが適切に行われている。

**良かった点**:

- Handler が薄くなり、HTTP の入出力変換に専念する設計に近づいている
- `model.ValidationError` を使った `errors.As` パターンによるエラーハンドリングが作業計画書の設計通りに実装されている
- Validator の戻り値が `(Result, error)` パターン → `(*Output, error)` パターンに統一され、エラーは `error` インターフェースで返すようになりシンプルになった
- テストコードの `setupHandler` ヘルパー抽出で重複が大幅に削減された
- テストのメールアドレス・atname がユニークに変更され、並行テストの安全性が向上している

**確認が必要な点**:

- `CreateUserSessionInput` が 2 つの異なるフロー（サインイン / 認証済み）を 1 つの Input で持つ設計が中間状態として許容されるか、分離すべきかの判断
- 認証済みフロー（`UserID` 直接指定）のテストが削除されている点
