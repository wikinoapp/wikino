# コードレビュー: cleanup-fix

## レビュー情報

| 項目                   | 内容                                                        |
| ---------------------- | ----------------------------------------------------------- |
| レビュー日             | 2026-02-09                                                  |
| 対象ブランチ           | cleanup-fix                                                 |
| ベースブランチ         | go                                                          |
| 設計書（指定があれば） | docs/designs/1_doing/rails-cleanup-go-migrated-endpoints.md |
| 変更ファイル数         | 104 ファイル                                                |
| 変更行数（実装）       | +43 / -2167 行                                              |
| 変更行数（テスト）     | +11 / -1243 行                                              |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@rails/CLAUDE.md](/workspace/rails/CLAUDE.md) - Rails版の開発ガイド
- [@CLAUDE.md#コメントのガイドライン](/workspace/CLAUDE.md) - コメントのガイドライン
- [@rails/CLAUDE.md#セキュリティガイドライン](/workspace/rails/CLAUDE.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル（修正）

- [x] `rails/config/routes.rb`
- [x] `rails/app/controllers/controller_concerns/authenticatable.rb`
- [x] `rails/app/controllers/settings/emails/update_controller.rb`
- [x] `rails/app/controllers/settings/account/deletions/create_controller.rb`
- [x] `rails/app/controllers/spaces/settings/deletions/create_controller.rb`
- [x] `rails/app/components/footers/global_component.html.erb`
- [x] `rails/app/components/links/brand_icon_component.html.erb`
- [x] `rails/app/components/navbars/bottom_component.html.erb`
- [x] `rails/app/components/sidebar/content_component.html.erb`
- [x] `rails/app/views/settings/show_view.html.erb`

### 実装ファイル（新規追加）

- [x] `rails/app/controllers/test/sign_in/create_controller.rb`

### 実装ファイル（削除）

- [x] `rails/app/controllers/controller_concerns/email_confirmation_findable.rb`
- [x] `rails/app/controllers/email_confirmations/create_controller.rb`
- [x] `rails/app/controllers/email_confirmations/edit_controller.rb`
- [x] `rails/app/controllers/email_confirmations/update_controller.rb`
- [x] `rails/app/controllers/accounts/new_controller.rb`
- [x] `rails/app/controllers/accounts/create_controller.rb`
- [x] `rails/app/controllers/password_resets/new_controller.rb`
- [x] `rails/app/controllers/password_resets/create_controller.rb`
- [x] `rails/app/controllers/passwords/edit_controller.rb`
- [x] `rails/app/controllers/passwords/update_controller.rb`
- [x] `rails/app/controllers/manifests/show_controller.rb`
- [x] `rails/app/controllers/sign_up/show_controller.rb`
- [x] `rails/app/controllers/sign_in/two_factors/new_controller.rb`
- [x] `rails/app/controllers/sign_in/two_factors/recoveries/new_controller.rb`
- [x] `rails/app/controllers/sign_in/two_factors/recoveries/create_controller.rb`
- [x] `rails/app/controllers/welcome/show_controller.rb`
- [x] `rails/app/controllers/user_sessions/create_controller.rb`
- [x] `rails/app/controllers/user_sessions/destroy_controller.rb`
- [x] `rails/app/views/welcome/show_view.rb`
- [x] `rails/app/views/welcome/show_view.html.erb`
- [x] `rails/app/views/sign_in/two_factors/new_view.rb`
- [x] `rails/app/views/sign_in/two_factors/new_view.html.erb`
- [x] `rails/app/views/sign_in/two_factors/recoveries/new_view.rb`
- [x] `rails/app/views/sign_in/two_factors/recoveries/new_view.html.erb`
- [x] `rails/app/views/sign_up/show_view.rb`
- [x] `rails/app/views/sign_up/show_view.html.erb`
- [x] `rails/app/views/email_confirmations/edit_view.rb`
- [x] `rails/app/views/email_confirmations/edit_view.html.erb`
- [x] `rails/app/views/accounts/new_view.rb`
- [x] `rails/app/views/accounts/new_view.html.erb`
- [x] `rails/app/views/password_resets/new_view.rb`
- [x] `rails/app/views/password_resets/new_view.html.erb`
- [x] `rails/app/views/passwords/edit_view.rb`
- [x] `rails/app/views/passwords/edit_view.html.erb`
- [x] `rails/app/views/manifests/show/call.json.erb`
- [x] `rails/app/forms/user_sessions/creation_form.rb`
- [x] `rails/app/forms/user_sessions/two_factor_verification_form.rb`
- [x] `rails/app/forms/user_sessions/two_factor_recovery_form.rb`
- [x] `rails/app/forms/email_confirmations/creation_form.rb`
- [x] `rails/app/forms/email_confirmations/check_form.rb`
- [x] `rails/app/forms/accounts/creation_form.rb`
- [x] `rails/app/forms/password_resets/creation_form.rb`
- [x] `rails/app/services/user_sessions/create_service.rb`
- [x] `rails/app/services/user_sessions/create_with_recovery_code_service.rb`
- [x] `rails/app/services/emails/confirm_service.rb`
- [x] `rails/app/services/accounts/create_service.rb`
- [x] `rails/app/services/passwords/update_service.rb`
- [x] `rails/app/repositories/user_session_repository.rb`

### テストファイル（修正）

- [x] `rails/spec/requests/attachments/presigns/create_spec.rb`
- [ ] `rails/spec/requests/search/show_spec.rb`
- [x] `rails/spec/requests/settings/account/deletions/create_spec.rb`
- [x] `rails/spec/support/request_spec_config.rb`
- [x] `rails/spec/support/system_spec_config.rb`
- [x] `rails/spec/system/global_hotkey_spec.rb`

### テストファイル（削除）

- [x] `rails/spec/requests/sign_in/show_spec.rb`
- [x] `rails/spec/requests/sign_in/two_factors/new_spec.rb`
- [x] `rails/spec/requests/sign_in/two_factors/create_spec.rb`
- [x] `rails/spec/requests/sign_in/two_factors/recoveries/new_spec.rb`
- [x] `rails/spec/requests/sign_in/two_factors/recoveries/create_spec.rb`
- [x] `rails/spec/requests/sign_up/show_spec.rb`
- [x] `rails/spec/requests/user_sessions/create_spec.rb`
- [x] `rails/spec/requests/user_sessions/destroy_spec.rb`
- [x] `rails/spec/requests/email_confirmations/create_spec.rb`
- [x] `rails/spec/requests/email_confirmations/edit_spec.rb`
- [x] `rails/spec/requests/email_confirmations/update_spec.rb`
- [x] `rails/spec/requests/accounts/new_spec.rb`
- [x] `rails/spec/requests/accounts/create_spec.rb`
- [x] `rails/spec/requests/password_resets/new_spec.rb`
- [x] `rails/spec/requests/password_resets/create_spec.rb`
- [x] `rails/spec/requests/passwords/edit_spec.rb`
- [x] `rails/spec/requests/passwords/update_spec.rb`
- [x] `rails/spec/requests/manifests/show_spec.rb`
- [x] `rails/spec/forms/accounts/creation_form_spec.rb`

### 自動生成・設定ファイル

- [x] `rails/sorbet/rbi/dsl/generated_path_helpers_module.rbi`
- [x] `rails/sorbet/rbi/dsl/generated_url_helpers_module.rbi`
- [ ] `rails/sorbet/rbi/todo.rbi`
- [x] `rails/sorbet/rbi/dsl/accounts/creation_form.rbi`
- [x] `rails/sorbet/rbi/dsl/email_confirmations/check_form.rbi`
- [x] `rails/sorbet/rbi/dsl/email_confirmations/creation_form.rbi`
- [x] `rails/sorbet/rbi/dsl/password_resets/creation_form.rbi`
- [x] `rails/sorbet/rbi/dsl/user_sessions/creation_form.rbi`
- [x] `rails/sorbet/rbi/dsl/user_sessions/two_factor_recovery_form.rbi`
- [x] `rails/sorbet/rbi/dsl/user_sessions/two_factor_verification_form.rbi`
- [x] `rails/sorbet/rbi/manual/controller_concerns/email_confirmation_findable.rbi`

### ドキュメント

- [x] `docs/designs/1_doing/rails-cleanup-go-migrated-endpoints.md`
- [x] `docs/designs/1_doing/sync-guideline-20260208.md`
- [x] `docs/designs/1_doing/sync-guideline-template.md`
- [x] `docs/reviews/done/202602/cleanup-1-1-202602080838.md`
- [x] `docs/reviews/template.md`

## ファイルごとのレビュー結果

### `rails/config/routes.rb`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@rails/CLAUDE.md](/workspace/rails/CLAUDE.md) - ルーティング
- 設計書: rails-cleanup-go-migrated-endpoints.md

**問題点・改善提案**:

1. **テスト環境用ルートが削除済みコントローラーを参照している**: `if Rails.env.test?` ブロック内で以下のルートが定義されているが、対応するコントローラーはすでに削除済み。テスト実行時にこれらのルートにアクセスすると `ActionController::RoutingError` が発生する。

   ```ruby
   # 問題のあるコード（21-28行目）
   if Rails.env.test?
     match "/manifest",        via: :get,    to: "manifests/show#call"       # コントローラー削除済み
     match "/sign_in",         via: :get,    to: "sign_in/show#call"         # 存在するがhead :okのスタブ
     match "/sign_in/two_factor", via: :post, to: "sign_in/two_factors/create#call" # 存在するが依存クラス削除済み
     match "/user_session",    via: :delete, to: "user_sessions/destroy#call" # コントローラー削除済み
     match "/user_session",    via: :post,   to: "user_sessions/create#call"  # コントローラー削除済み
     root to: "welcome/show#call"                                             # コントローラー削除済み
   end
   ```

   特に `spec/requests/search/show_spec.rb:19` で `delete "/user_session"` を呼び出しており、このテストは実行時にエラーになる可能性が高い。

   **修正案**: テスト環境用ルートブロックから、対応するコントローラーが削除済みのルートを削除する。`sign_in` GETルートはスタブコントローラーが存在するので残してもよいが、テストで使用されていないなら削除が望ましい。

   **対応方針**:

   - [x] テスト環境用ルートブロック全体を削除する
   - [ ] 削除済みコントローラーのルートのみ削除し、スタブコントローラーのルートは残す
   - [ ] その他（下の回答欄に記入）

### `rails/spec/requests/search/show_spec.rb`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@rails/CLAUDE.md#RSpec](/workspace/rails/CLAUDE.md) - テスト規約

**問題点・改善提案**:

1. **削除済みエンドポイントへのリクエスト**: 19行目で `delete "/user_session"` を呼び出しているが、`user_sessions/destroy_controller.rb` は削除済み。テスト環境用ルートは `routes.rb` に残っているが、コントローラーが存在しないためランタイムエラーになる。

   ```ruby
   # 問題のあるコード（18-21行目）
   it "ログインしていない場合、ログインページにリダイレクトされること" do
     delete "/user_session"
     get search_path
     expect(response).to redirect_to("/sign_in")
   end
   ```

   **修正案**: `sign_in` ヘルパーを呼ばずにリクエストすれば未ログイン状態をテストできるため、`delete "/user_session"` の行を削除する。

   ```ruby
   it "ログインしていない場合、ログインページにリダイレクトされること" do
     get search_path
     expect(response).to redirect_to("/sign_in")
   end
   ```

   **対応方針**: `delete "/user_session"` の行を削除してください

### `rails/sorbet/rbi/todo.rbi`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@rails/CLAUDE.md](/workspace/rails/CLAUDE.md) - Sorbet型チェック

**問題点・改善提案**:

1. **削除済みクラスのスタブが存在する**: `todo.rbi` に以下のスタブが追加されているが、これは `sign_in/two_factors/create_controller.rb` が削除済みの `UserSessions::CreateService` と `UserSessions::TwoFactorVerificationForm` を参照しているため。

   ```ruby
   module UserSessions::CreateService; end
   module UserSessions::TwoFactorVerificationForm; end
   ```

   根本原因は `sign_in/two_factors/create_controller.rb` が削除されていないことにある。このコントローラーを削除すれば `todo.rbi` のスタブも不要になる。

### 設計との整合性: 削除漏れファイル

**ステータス**: 要修正

**チェックしたガイドライン**:

- 設計書: rails-cleanup-go-migrated-endpoints.md（削除対象ファイル一覧）

**問題点・改善提案**:

1. **設計書で削除対象とされているが、まだ存在するファイル**:

   | ファイル | 設計書での削除フェーズ | 現在の状態 |
   | --- | --- | --- |
   | `app/controllers/sign_in/show_controller.rb` | 2-1 | スタブ（`head :ok`）として残存 |
   | `app/controllers/sign_in/two_factors/create_controller.rb` | 2-1 | 完全な実装が残存（削除済みクラスを参照） |
   | `app/views/sign_in/show_view.rb` | 2-1 | 残存（デッドコード） |
   | `app/views/sign_in/show_view.html.erb` | 2-1 | 残存（デッドコード） |

   `sign_in/show_controller.rb` は `head :ok` を返すスタブになっているが、テスト環境用ルートからのみ参照される。`sign_in/two_factors/create_controller.rb` は削除済みの `UserSessions::TwoFactorVerificationForm` と `UserSessions::CreateService` を参照しており、実行時にエラーになる。ビューファイルは完全にデッドコード。

   **修正案**: これら4ファイルを削除し、対応するテスト環境用ルートも `routes.rb` から削除する。`sorbet/rbi/todo.rbi` のスタブも不要になるため削除する。

   **対応方針**: 4ファイルを削除してください

## 設計との整合性チェック

### 設計書の要件と実装の比較

| 要件 | 状態 | 備考 |
| --- | --- | --- |
| routes.rbからGo移行済みルート定義を削除 | ✅ | 本番/開発環境からは全削除済み |
| コントローラー19ファイルの削除 | ⚠️ | 17/19削除。`sign_in/show`と`sign_in/two_factors/create`が残存 |
| コントローラーコンサーン1ファイルの削除 | ✅ | `email_confirmation_findable.rb` 削除済み |
| ビュー19ファイルの削除 | ⚠️ | 17/19削除。`sign_in/show_view.{rb,html.erb}`が残存 |
| フォーム7ファイルの削除 | ✅ | 全て削除済み |
| サービス5ファイルの削除 | ✅ | 全て削除済み |
| リポジトリ1ファイルの削除 | ✅ | 削除済み |
| テスト19ファイルの削除 | ✅ | 全て削除済み |
| Sorbet RBI 8ファイルの削除 | ✅ | 全て削除済み |
| 削除してはいけないファイルの保護 | ✅ | 共有依存ファイルは全て保持されている |
| 各フェーズで `bin/check` 実行 | ⚠️ | テスト環境用ルートが削除済みコントローラーを参照する問題あり |
| テスト用sign_inヘルパーの更新 | ✅ | DB直接操作に変更済み。テスト用エンドポイントも追加 |

### パスヘルパーの文字列リテラル置き換え

設計書には明記されていないが、ルート削除に伴うnamed routeヘルパーの参照更新は全て実施済み。`root_path` → `"/"`、`sign_in_path` → `"/sign_in"`、`user_session_path` → `"/user_session"` 等、app/とspec/の全箇所で一貫して文字列リテラルに置き換えられている。

### テストヘルパーの変更

`request_spec_config.rb` の `sign_in` ヘルパーをHTTPリクエスト経由からDB直接操作に変更した判断は適切。Go版にサインインエンドポイントが移行済みのため、Rails側でHTTPリクエストを使ったサインインテストは不可能になっている。

`system_spec_config.rb` のシステムテスト用 `sign_in` ヘルパーもテスト用エンドポイント `/_test/sign_in` 経由に変更されており、Go版のUIに依存しない設計になっている。

## 総合評価

**評価**: Request Changes

**総評**:

Go版に移行済みの19エンドポイントに対応するRails版コードの大規模削除を実施しており、全体的な方針は設計書に沿っている。コントローラー、ビュー、フォーム、サービス、リポジトリ、テスト、RBIファイルの大部分が正しく削除されている。パスヘルパーの文字列リテラル置き換えも一貫して実施済み。テストヘルパーの更新も適切。

ただし、以下の問題点が存在するため修正が必要：

1. **削除漏れ（4ファイル）**: `sign_in/show_controller.rb`（スタブ化のみ）、`sign_in/two_factors/create_controller.rb`（削除済みクラスを参照）、`sign_in/show_view.rb`、`sign_in/show_view.html.erb` が設計書の削除対象にもかかわらず残存
2. **テスト環境用ルートの不整合**: `routes.rb` のテスト環境用ルートブロックが削除済みコントローラー（`manifests/show`、`user_sessions/destroy`、`user_sessions/create`、`welcome/show`）を参照しており、テスト実行時にエラーになる可能性がある
3. **`search/show_spec.rb` の壊れたテスト**: `delete "/user_session"` が削除済みコントローラーを呼び出す
4. **`sorbet/rbi/todo.rbi` の不要スタブ**: 削除漏れファイルの依存解決のために追加されたスタブ。根本原因（ファイルの削除漏れ）を解決すれば不要
