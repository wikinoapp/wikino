# Go移行済みエンドポイントのRails版コード削除 設計書

<!--
このテンプレートの使い方:
1. このファイルを `docs/designs/2_todo/` ディレクトリにコピー
   例: cp docs/designs/template.md docs/designs/2_todo/new-feature.md
2. [機能名] などのプレースホルダーを実際の内容に置き換え
3. 各セクションのガイドラインに従って記述
4. コメント（ `\<!-- ... --\>` ）はガイドラインとして残しておくことを推奨

**公開時の注意事項**:
- 開発用ドメイン名を記載する場合は `example.dev` を使用してください（実際のドメイン名は記載しない）
- 環境変数の値はサンプル値のみ記載し、実際の値は含めないでください
-->

## 実装ガイドラインの参照

### Rails版の実装の場合

以下のガイドラインに従って設計・実装を行ってください：

- [@rails/CLAUDE.md](/workspace/rails/CLAUDE.md) - 全体的なコーディング規約

## 概要

Go版に移行済みのエンドポイントに対応するRails版のコード（コントローラー、ビュー、フォーム、サービス、テスト等）を削除し、不要なコードを整理する。

**目的**:

- Go版に移行完了したエンドポイントのRails版コードを削除し、コードベースをクリーンに保つ
- 保守対象のコード量を削減し、開発効率を向上させる

**背景**:

- Go版のリバースプロキシミドルウェアにより、移行済みエンドポイントへのリクエストはすべてGo版で処理される
- Rails版の対応コードはリクエストが到達しないため、完全にデッドコードとなっている
- デッドコードの放置は、依存関係の理解を困難にし、リファクタリングの妨げとなる

## 要件

### 機能要件

- Go版で処理される19のルート定義をRails版のroutes.rbから削除する
- 対応するコントローラー、ビュー、フォーム、サービス、テストファイルを削除する
- 削除後もRails版の既存機能（設定画面、スペース管理等）が正常に動作する

### 非機能要件

- **安全性**: 他の機能で使用されている共有コード（モデル、レコード、一部サービス等）は削除しない
- **検証**: 各フェーズで `bin/check` を実行し、Zeitwerk・Sorbet・テストが通ることを確認する

## 設計

### Go版移行済みエンドポイント一覧

`go/cmd/server/main.go` で定義されている以下のエンドポイントが移行済み：

| HTTPメソッド | URLパス                            | 機能                            |
| ------------ | ---------------------------------- | ------------------------------- |
| GET          | `/`                                | ウェルカムページ                |
| GET          | `/health`                          | ヘルスチェック                  |
| GET          | `/manifest.json`                   | PWAマニフェスト                 |
| GET          | `/sign_in`                         | サインインフォーム              |
| POST         | `/sign_in`                         | サインイン処理                  |
| GET          | `/sign_in/two_factor/new`          | 2FAコード入力フォーム           |
| POST         | `/sign_in/two_factor`              | 2FAコード検証                   |
| GET          | `/sign_in/two_factor/recovery/new` | 2FAリカバリーコード入力フォーム |
| POST         | `/sign_in/two_factor/recovery`     | 2FAリカバリーコード検証         |
| GET          | `/sign_up`                         | サインアップフォーム            |
| POST         | `/email_confirmation`              | メール確認コード送信            |
| GET          | `/email_confirmation/edit`         | メール確認コード入力フォーム    |
| PATCH        | `/email_confirmation`              | メール確認コード検証            |
| GET          | `/accounts/new`                    | アカウント作成フォーム          |
| POST         | `/accounts`                        | アカウント作成処理              |
| GET          | `/password/reset`                  | パスワードリセット申請フォーム  |
| POST         | `/password/reset`                  | パスワードリセット申請処理      |
| GET          | `/password/edit`                   | パスワード変更フォーム          |
| PATCH        | `/password`                        | パスワード変更処理              |
| DELETE       | `/user_session`                    | サインアウト                    |

### 対応するRails側ルート（routes.rb）

削除対象のルート定義（計20行）：

| 行番号 | ルート                                 | コントローラー                               |
| ------ | -------------------------------------- | -------------------------------------------- |
| 17     | `POST /accounts`                       | `accounts/create#call`                       |
| 18     | `GET /accounts/new`                    | `accounts/new#call`                          |
| 21     | `PATCH /email_confirmation`            | `email_confirmations/update#call`            |
| 22     | `POST /email_confirmation`             | `email_confirmations/create#call`            |
| 23     | `GET /email_confirmation/edit`         | `email_confirmations/edit#call`              |
| 26     | `GET /manifest`                        | `manifests/show#call`                        |
| 27     | `GET /password_reset`                  | `password_resets/new#call`                   |
| 28     | `POST /password_reset`                 | `password_resets/create#call`                |
| 29     | `PATCH /password`                      | `passwords/update#call`                      |
| 30     | `GET /password/edit`                   | `passwords/edit#call`                        |
| 80     | `GET /sign_in`                         | `sign_in/show#call`                          |
| 81     | `GET /sign_in/two_factor/new`          | `sign_in/two_factors/new#call`               |
| 82     | `POST /sign_in/two_factor`             | `sign_in/two_factors/create#call`            |
| 83     | `GET /sign_in/two_factor/recovery/new` | `sign_in/two_factors/recoveries/new#call`    |
| 84     | `POST /sign_in/two_factor/recovery`    | `sign_in/two_factors/recoveries/create#call` |
| 85     | `GET /sign_up`                         | `sign_up/show#call`                          |
| 89     | `DELETE /user_session`                 | `user_sessions/destroy#call`                 |
| 90     | `POST /user_session`                   | `user_sessions/create#call`                  |
| 93     | `root`                                 | `welcome/show#call`                          |

### 削除対象ファイル一覧

#### コントローラー（19ファイル）

| ファイルパス                                                          | 備考 |
| --------------------------------------------------------------------- | ---- |
| `app/controllers/welcome/show_controller.rb`                          |      |
| `app/controllers/sign_in/show_controller.rb`                          |      |
| `app/controllers/sign_in/two_factors/new_controller.rb`               |      |
| `app/controllers/sign_in/two_factors/create_controller.rb`            |      |
| `app/controllers/sign_in/two_factors/recoveries/new_controller.rb`    |      |
| `app/controllers/sign_in/two_factors/recoveries/create_controller.rb` |      |
| `app/controllers/sign_up/show_controller.rb`                          |      |
| `app/controllers/user_sessions/create_controller.rb`                  |      |
| `app/controllers/user_sessions/destroy_controller.rb`                 |      |
| `app/controllers/email_confirmations/create_controller.rb`            |      |
| `app/controllers/email_confirmations/edit_controller.rb`              |      |
| `app/controllers/email_confirmations/update_controller.rb`            |      |
| `app/controllers/accounts/new_controller.rb`                          |      |
| `app/controllers/accounts/create_controller.rb`                       |      |
| `app/controllers/password_resets/new_controller.rb`                   |      |
| `app/controllers/password_resets/create_controller.rb`                |      |
| `app/controllers/passwords/edit_controller.rb`                        |      |
| `app/controllers/passwords/update_controller.rb`                      |      |
| `app/controllers/manifests/show_controller.rb`                        |      |

#### コントローラーコンサーン（1ファイル）

| ファイルパス                                                         | 備考                                  |
| -------------------------------------------------------------------- | ------------------------------------- |
| `app/controllers/controller_concerns/email_confirmation_findable.rb` | 使用元6コントローラーすべてが削除対象 |

#### ビュー（19ファイル）

| ファイルパス                                                 | 備考 |
| ------------------------------------------------------------ | ---- |
| `app/views/welcome/show_view.rb`                             |      |
| `app/views/welcome/show_view.html.erb`                       |      |
| `app/views/sign_in/show_view.rb`                             |      |
| `app/views/sign_in/show_view.html.erb`                       |      |
| `app/views/sign_in/two_factors/new_view.rb`                  |      |
| `app/views/sign_in/two_factors/new_view.html.erb`            |      |
| `app/views/sign_in/two_factors/recoveries/new_view.rb`       |      |
| `app/views/sign_in/two_factors/recoveries/new_view.html.erb` |      |
| `app/views/sign_up/show_view.rb`                             |      |
| `app/views/sign_up/show_view.html.erb`                       |      |
| `app/views/email_confirmations/edit_view.rb`                 |      |
| `app/views/email_confirmations/edit_view.html.erb`           |      |
| `app/views/accounts/new_view.rb`                             |      |
| `app/views/accounts/new_view.html.erb`                       |      |
| `app/views/password_resets/new_view.rb`                      |      |
| `app/views/password_resets/new_view.html.erb`                |      |
| `app/views/passwords/edit_view.rb`                           |      |
| `app/views/passwords/edit_view.html.erb`                     |      |
| `app/views/manifests/show/call.json.erb`                     |      |

#### フォーム（7ファイル）

| ファイルパス                                              | 備考 |
| --------------------------------------------------------- | ---- |
| `app/forms/user_sessions/creation_form.rb`                |      |
| `app/forms/user_sessions/two_factor_verification_form.rb` |      |
| `app/forms/user_sessions/two_factor_recovery_form.rb`     |      |
| `app/forms/email_confirmations/creation_form.rb`          |      |
| `app/forms/email_confirmations/check_form.rb`             |      |
| `app/forms/accounts/creation_form.rb`                     |      |
| `app/forms/password_resets/creation_form.rb`              |      |

#### サービス（5ファイル）

| ファイルパス                                                      | 備考 |
| ----------------------------------------------------------------- | ---- |
| `app/services/user_sessions/create_service.rb`                    |      |
| `app/services/user_sessions/create_with_recovery_code_service.rb` |      |
| `app/services/emails/confirm_service.rb`                          |      |
| `app/services/accounts/create_service.rb`                         |      |
| `app/services/passwords/update_service.rb`                        |      |

#### リポジトリ（1ファイル）

| ファイルパス                                  | 備考                     |
| --------------------------------------------- | ------------------------ |
| `app/repositories/user_session_repository.rb` | コードベース内で参照なし |

#### テストファイル（19ファイル）

| ファイルパス                                                  | 備考 |
| ------------------------------------------------------------- | ---- |
| `spec/requests/sign_in/show_spec.rb`                          |      |
| `spec/requests/sign_in/two_factors/new_spec.rb`               |      |
| `spec/requests/sign_in/two_factors/create_spec.rb`            |      |
| `spec/requests/sign_in/two_factors/recoveries/new_spec.rb`    |      |
| `spec/requests/sign_in/two_factors/recoveries/create_spec.rb` |      |
| `spec/requests/sign_up/show_spec.rb`                          |      |
| `spec/requests/user_sessions/create_spec.rb`                  |      |
| `spec/requests/user_sessions/destroy_spec.rb`                 |      |
| `spec/requests/email_confirmations/create_spec.rb`            |      |
| `spec/requests/email_confirmations/edit_spec.rb`              |      |
| `spec/requests/email_confirmations/update_spec.rb`            |      |
| `spec/requests/accounts/new_spec.rb`                          |      |
| `spec/requests/accounts/create_spec.rb`                       |      |
| `spec/requests/password_resets/new_spec.rb`                   |      |
| `spec/requests/password_resets/create_spec.rb`                |      |
| `spec/requests/passwords/edit_spec.rb`                        |      |
| `spec/requests/passwords/update_spec.rb`                      |      |
| `spec/requests/manifests/show_spec.rb`                        |      |
| `spec/forms/accounts/creation_form_spec.rb`                   |      |

#### Sorbet RBIファイル（削除後に `make sorbet-update` で自動削除されるもの）

| ファイルパス                                                            | 備考 |
| ----------------------------------------------------------------------- | ---- |
| `sorbet/rbi/dsl/user_sessions/creation_form.rbi`                        |      |
| `sorbet/rbi/dsl/user_sessions/two_factor_verification_form.rbi`         |      |
| `sorbet/rbi/dsl/user_sessions/two_factor_recovery_form.rbi`             |      |
| `sorbet/rbi/dsl/email_confirmations/creation_form.rbi`                  |      |
| `sorbet/rbi/dsl/email_confirmations/check_form.rbi`                     |      |
| `sorbet/rbi/dsl/accounts/creation_form.rbi`                             |      |
| `sorbet/rbi/dsl/password_resets/creation_form.rbi`                      |      |
| `sorbet/rbi/manual/controller_concerns/email_confirmation_findable.rbi` |      |

### 削除してはいけないファイル（共有依存）

以下のファイルは、削除対象外のRails機能から参照されているため、**削除してはいけない**。

#### レコード・モデル

| ファイルパス                               | 参照元（削除対象外）                                               |
| ------------------------------------------ | ------------------------------------------------------------------ |
| `app/records/user_session_record.rb`       | `authenticatable.rb`（認証コンサーン、全認証コントローラーで使用） |
| `app/records/email_confirmation_record.rb` | `settings/emails/update_controller.rb` が CreateService 経由で使用 |
| `app/records/user_record.rb`               | 全般的に使用                                                       |
| `app/models/user_session.rb`               | `authenticatable.rb` が `TOKENS_COOKIE_KEY` 定数を参照             |
| `app/models/email_confirmation_event.rb`   | `email_confirmation_record.rb` が使用                              |

#### サービス

| ファイルパス                                         | 参照元（削除対象外）                                                                                                |
| ---------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------- |
| `app/services/user_sessions/destroy_service.rb`      | `authenticatable.rb` の `sign_out` メソッドが呼び出し（`settings/account/deletions/create_controller.rb` 等で使用） |
| `app/services/email_confirmations/create_service.rb` | `settings/emails/update_controller.rb` が呼び出し（メールアドレス変更フロー）                                       |
| `app/services/accounts/destroy_service.rb`           | `destroy_account_job.rb` が呼び出し                                                                                 |
| `app/services/accounts/soft_destroy_service.rb`      | `settings/account/deletions/create_controller.rb` が呼び出し                                                        |

#### メーラー

| ファイルパス                                                         | 参照元（削除対象外）                                                                       |
| -------------------------------------------------------------------- | ------------------------------------------------------------------------------------------ |
| `app/mailers/email_confirmation_mailer.rb`                           | `email_confirmation_record.rb` の `send_mail!` メソッド経由で `CreateService` から呼び出し |
| `app/views/email_confirmation_mailer/email_confirmation.ja.html.erb` | 上記メーラーのテンプレート                                                                 |
| `app/views/email_confirmation_mailer/email_confirmation.en.html.erb` | 上記メーラーのテンプレート                                                                 |

#### フォームコンサーン

| ファイルパス                                          | 参照元（削除対象外）                       |
| ----------------------------------------------------- | ------------------------------------------ |
| `app/forms/form_concerns/password_validatable.rb`     | `TwoFactorAuths::CreationForm` 等で使用    |
| `app/forms/form_concerns/password_authenticatable.rb` | `TwoFactorAuths::DestructionForm` 等で使用 |

#### その他

| ファイルパス                                             | 参照元（削除対象外）                           |
| -------------------------------------------------------- | ---------------------------------------------- |
| `app/forms/accounts/destroy_confirmation_form.rb`        | アカウント削除設定画面で使用                   |
| `app/controllers/controller_concerns/authenticatable.rb` | 全認証コントローラーで使用                     |
| `app/controllers/home/show_controller.rb`                | `/home` は Go 未移行（ログイン後のホーム画面） |
| `app/views/home/show_view.rb`                            | 同上                                           |
| `app/views/home/show_view.html.erb`                      | 同上                                           |
| `spec/factories/email_confirmation_record.rb`            | 他のテストから使用される可能性                 |

### 設定メールアドレス変更フローに関する注意

`settings/emails/update_controller.rb` は以下のフローで動作する：

1. `PATCH /settings/email` → Rails が処理
2. `EmailConfirmations::CreateService` でメール確認レコードを作成し、メーラーで確認メール送信
3. `/email_confirmation/edit` にリダイレクト → **Go版が処理**（Go のリバースプロキシで処理される）
4. ユーザーが確認コードを入力し `PATCH /email_confirmation` → **Go版が処理**

Go版の `markEmailAsConfirmedUC` が、Rails版の `Emails::ConfirmService` と同等の処理（メール確認の成功マーク、ユーザーメールアドレスの更新）を行っていることを**確認済みであること**が前提。

## タスクリスト

<!--
本設計は純粋なコード削除（デッドコードの除去）であるため、テストコードの新規追加は不要。
削除対象のテストファイルは「テスト」列に計上する。
各フェーズの後に `bin/check` を実行してCI通過を確認する。
-->

### フェーズ 1: ルート定義の削除

- [x] **1-1**: [Rails] routes.rb から Go 移行済みルート定義を削除
  - `config/routes.rb` から20行のルート定義を削除
  - 削除後に `bin/check` を実行して確認
  - **想定ファイル数**: 約 1 ファイル（実装 1 + テスト 0）
  - **想定行数**: 約 20 行（実装 20 行 + テスト 0 行）

### フェーズ 2: サインイン・サインアップ関連の削除

- [x] **2-1**: [Rails] サインイン関連のコントローラー・ビューの削除
  - コントローラー削除（6ファイル）: `sign_in/show`, `sign_in/two_factors/new`, `sign_in/two_factors/create`, `sign_in/two_factors/recoveries/new`, `sign_in/two_factors/recoveries/create`, `sign_up/show`
  - ビュー削除（8ファイル）: `sign_in/show_view.{rb,html.erb}`, `sign_in/two_factors/new_view.{rb,html.erb}`, `sign_in/two_factors/recoveries/new_view.{rb,html.erb}`, `sign_up/show_view.{rb,html.erb}`
  - テスト削除（6ファイル）: `sign_in/show_spec`, `sign_in/two_factors/new_spec`, `sign_in/two_factors/create_spec`, `sign_in/two_factors/recoveries/new_spec`, `sign_in/two_factors/recoveries/create_spec`, `sign_up/show_spec`
  - **想定ファイル数**: 約 20 ファイル（実装 14 + テスト 6）
  - **想定行数**: 約 300 行（実装 200 行 + テスト 100 行）※すべて削除行

- [x] **2-2**: [Rails] サインイン関連のフォーム・サービスの削除
  - フォーム削除（3ファイル）: `user_sessions/creation_form`, `user_sessions/two_factor_verification_form`, `user_sessions/two_factor_recovery_form`
  - サービス削除（2ファイル）: `user_sessions/create_service`, `user_sessions/create_with_recovery_code_service`
  - RBI 削除（3ファイル）: `user_sessions/creation_form.rbi`, `user_sessions/two_factor_verification_form.rbi`, `user_sessions/two_factor_recovery_form.rbi`
  - `make sorbet-update` を実行
  - **想定ファイル数**: 約 8 ファイル（実装 8 + テスト 0）
  - **想定行数**: 約 250 行（実装 250 行 + テスト 0 行）※すべて削除行

### フェーズ 3: セッション・ウェルカム・マニフェスト関連の削除

- [ ] **3-1**: [Rails] ウェルカム・セッション・マニフェスト関連の削除
  - コントローラー削除（3ファイル）: `welcome/show`, `user_sessions/create`, `user_sessions/destroy`
  - コントローラー削除（1ファイル）: `manifests/show`
  - ビュー削除（3ファイル）: `welcome/show_view.{rb,html.erb}`, `manifests/show/call.json.erb`
  - リポジトリ削除（1ファイル）: `user_session_repository`
  - テスト削除（3ファイル）: `user_sessions/create_spec`, `user_sessions/destroy_spec`, `manifests/show_spec`
  - **想定ファイル数**: 約 11 ファイル（実装 8 + テスト 3）
  - **想定行数**: 約 200 行（実装 150 行 + テスト 50 行）※すべて削除行

### フェーズ 4: メール確認関連の削除

- [ ] **4-1**: [Rails] メール確認関連のコントローラー・ビュー・フォーム・サービスの削除
  - コントローラー削除（3ファイル）: `email_confirmations/create`, `email_confirmations/edit`, `email_confirmations/update`
  - コントローラーコンサーン削除（1ファイル）: `email_confirmation_findable`
  - ビュー削除（2ファイル）: `email_confirmations/edit_view.{rb,html.erb}`
  - フォーム削除（2ファイル）: `email_confirmations/creation_form`, `email_confirmations/check_form`
  - サービス削除（1ファイル）: `emails/confirm_service`
  - RBI 削除（3ファイル）: `email_confirmations/creation_form.rbi`, `email_confirmations/check_form.rbi`, `email_confirmation_findable.rbi`
  - テスト削除（3ファイル）: `email_confirmations/create_spec`, `edit_spec`, `update_spec`
  - `make sorbet-update` を実行
  - **想定ファイル数**: 約 15 ファイル（実装 12 + テスト 3）
  - **想定行数**: 約 300 行（実装 250 行 + テスト 50 行）※すべて削除行

### フェーズ 5: アカウント・パスワード関連の削除

- [ ] **5-1**: [Rails] アカウント作成関連の削除
  - コントローラー削除（2ファイル）: `accounts/new`, `accounts/create`
  - ビュー削除（2ファイル）: `accounts/new_view.{rb,html.erb}`
  - フォーム削除（1ファイル）: `accounts/creation_form`
  - サービス削除（1ファイル）: `accounts/create_service`
  - RBI 削除（1ファイル）: `accounts/creation_form.rbi`
  - テスト削除（3ファイル）: `accounts/new_spec`, `accounts/create_spec`, `accounts/creation_form_spec`
  - `make sorbet-update` を実行
  - **想定ファイル数**: 約 10 ファイル（実装 7 + テスト 3）
  - **想定行数**: 約 250 行（実装 200 行 + テスト 50 行）※すべて削除行

- [ ] **5-2**: [Rails] パスワードリセット・変更関連の削除
  - コントローラー削除（4ファイル）: `password_resets/new`, `password_resets/create`, `passwords/edit`, `passwords/update`
  - ビュー削除（4ファイル）: `password_resets/new_view.{rb,html.erb}`, `passwords/edit_view.{rb,html.erb}`
  - フォーム削除（1ファイル）: `password_resets/creation_form`
  - サービス削除（1ファイル）: `passwords/update_service`
  - RBI 削除（1ファイル）: `password_resets/creation_form.rbi`
  - テスト削除（4ファイル）: `password_resets/new_spec`, `password_resets/create_spec`, `passwords/edit_spec`, `passwords/update_spec`
  - `make sorbet-update` を実行
  - **想定ファイル数**: 約 15 ファイル（実装 11 + テスト 4）
  - **想定行数**: 約 300 行（実装 250 行 + テスト 50 行）※すべて削除行

### 実装しない機能（スコープ外）

以下の機能は今回の実装では**実装しません**：

- **I18n キーの削除**: 削除対象のビュー・フォームでのみ使用されている翻訳キーの特定と削除。影響範囲の特定が複雑なため、別タスクとして実施する
- **レコード・モデルのメソッド整理**: `UserRecord.run_after_email_confirmation_success!` 等、削除対象コントローラーからのみ呼ばれていたメソッドの整理。Go版の対応実装で代替されているが、レコードクラスの変更はリスクが高いため別タスクとする
- **空ディレクトリの削除**: ファイル削除後に空になるディレクトリの整理（Gitは空ディレクトリを追跡しないため、自動的に解決される）

## 参考資料

- [go/cmd/server/main.go](/workspace/go/cmd/server/main.go) - Go版のエントリポイント（移行済みエンドポイント一覧）
- [rails/config/routes.rb](/workspace/rails/config/routes.rb) - Rails版のルーティング定義
- [go/internal/middleware/reverse_proxy.go](/workspace/go/internal/middleware/reverse_proxy.go) - リバースプロキシミドルウェア
