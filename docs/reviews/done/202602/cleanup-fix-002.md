# コードレビュー: cleanup-fix

## レビュー情報

| 項目                   | 内容                                                        |
| ---------------------- | ----------------------------------------------------------- |
| レビュー日             | 2026-02-09                                                  |
| 対象ブランチ           | cleanup-fix                                                 |
| ベースブランチ         | go                                                          |
| 設計書（指定があれば） | docs/designs/1_doing/rails-cleanup-go-migrated-endpoints.md |
| 変更ファイル数         | 104 ファイル                                                |
| 変更行数（実装）       | +1544 / -3587 行                                            |
| 変更行数（テスト）     | 含む（主に削除）                                            |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@rails/CLAUDE.md](/workspace/rails/CLAUDE.md) - Rails版の開発ガイド

## 変更ファイル一覧

### 削除ファイル: コントローラー（20ファイル）

- [x] `rails/app/controllers/welcome/show_controller.rb`
- [x] `rails/app/controllers/sign_in/show_controller.rb`
- [x] `rails/app/controllers/sign_in/two_factors/new_controller.rb`
- [x] `rails/app/controllers/sign_in/two_factors/create_controller.rb`
- [x] `rails/app/controllers/sign_in/two_factors/recoveries/new_controller.rb`
- [x] `rails/app/controllers/sign_in/two_factors/recoveries/create_controller.rb`
- [x] `rails/app/controllers/sign_up/show_controller.rb`
- [x] `rails/app/controllers/user_sessions/create_controller.rb`
- [x] `rails/app/controllers/user_sessions/destroy_controller.rb`
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
- [x] `rails/app/controllers/controller_concerns/email_confirmation_findable.rb`

### 削除ファイル: ビュー（19ファイル）

- [x] `rails/app/views/welcome/show_view.rb`
- [x] `rails/app/views/welcome/show_view.html.erb`
- [x] `rails/app/views/sign_in/show_view.rb`
- [x] `rails/app/views/sign_in/show_view.html.erb`
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

### 削除ファイル: フォーム（7ファイル）

- [x] `rails/app/forms/user_sessions/creation_form.rb`
- [x] `rails/app/forms/user_sessions/two_factor_verification_form.rb`
- [x] `rails/app/forms/user_sessions/two_factor_recovery_form.rb`
- [x] `rails/app/forms/email_confirmations/creation_form.rb`
- [x] `rails/app/forms/email_confirmations/check_form.rb`
- [x] `rails/app/forms/accounts/creation_form.rb`
- [x] `rails/app/forms/password_resets/creation_form.rb`

### 削除ファイル: サービス（5ファイル）

- [x] `rails/app/services/user_sessions/create_service.rb`
- [x] `rails/app/services/user_sessions/create_with_recovery_code_service.rb`
- [x] `rails/app/services/emails/confirm_service.rb`
- [x] `rails/app/services/accounts/create_service.rb`
- [x] `rails/app/services/passwords/update_service.rb`

### 削除ファイル: リポジトリ（1ファイル）

- [x] `rails/app/repositories/user_session_repository.rb`

### 削除ファイル: テスト（19ファイル）

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

### 削除ファイル: Sorbet RBI（8ファイル）

- [x] `rails/sorbet/rbi/dsl/accounts/creation_form.rbi`
- [x] `rails/sorbet/rbi/dsl/email_confirmations/check_form.rbi`
- [x] `rails/sorbet/rbi/dsl/email_confirmations/creation_form.rbi`
- [x] `rails/sorbet/rbi/dsl/password_resets/creation_form.rbi`
- [x] `rails/sorbet/rbi/dsl/user_sessions/creation_form.rbi`
- [x] `rails/sorbet/rbi/dsl/user_sessions/two_factor_recovery_form.rbi`
- [x] `rails/sorbet/rbi/dsl/user_sessions/two_factor_verification_form.rbi`
- [x] `rails/sorbet/rbi/manual/controller_concerns/email_confirmation_findable.rbi`

### 修正ファイル: 実装

- [x] `rails/config/routes.rb`
- [x] `rails/app/controllers/controller_concerns/authenticatable.rb`
- [x] `rails/app/controllers/settings/emails/update_controller.rb`
- [x] `rails/app/controllers/settings/account/deletions/create_controller.rb`
- [x] `rails/app/controllers/spaces/settings/deletions/create_controller.rb`
- [x] `rails/app/views/settings/show_view.html.erb`
- [x] `rails/app/components/footers/global_component.html.erb`
- [x] `rails/app/components/links/brand_icon_component.html.erb`
- [x] `rails/app/components/navbars/bottom_component.html.erb`
- [x] `rails/app/components/sidebar/content_component.html.erb`
- [x] `rails/sorbet/rbi/dsl/generated_path_helpers_module.rbi`
- [x] `rails/sorbet/rbi/dsl/generated_url_helpers_module.rbi`

### 追加ファイル: 実装

- [x] `rails/app/controllers/test/sign_in/create_controller.rb`

### 修正ファイル: テスト

- [x] `rails/spec/support/request_spec_config.rb`
- [x] `rails/spec/support/system_spec_config.rb`
- [x] `rails/spec/system/global_hotkey_spec.rb`
- [x] `rails/spec/requests/attachments/presigns/create_spec.rb`
- [x] `rails/spec/requests/search/show_spec.rb`
- [x] `rails/spec/requests/settings/account/deletions/create_spec.rb`

### 追加ファイル: ドキュメント

- [x] `docs/designs/1_doing/rails-cleanup-go-migrated-endpoints.md`
- [x] `docs/designs/1_doing/sync-guideline-20260208.md`
- [x] `docs/designs/1_doing/sync-guideline-template.md`
- [x] `docs/reviews/done/202602/cleanup-1-1-202602080838.md`
- [x] `docs/reviews/done/202602/cleanup-fix-202602090458.md`
- [x] `docs/reviews/template.md`

## ファイルごとのレビュー結果

全ファイルについて問題なし。以下、修正ファイルごとの確認結果を記載する。

### 修正ファイルの確認結果

**`rails/config/routes.rb`**: 設計書の19エンドポイント（20ルート定義）すべてが正しく削除されている。`root` ルートも削除済み。テスト用エンドポイント `/_test/sign_in` が `if Rails.env.test?` ガード付きで適切に追加されている。

**`rails/app/controllers/controller_concerns/authenticatable.rb`**: `sign_in_path` → `"/sign_in"` への変更。ルートヘルパーが削除されたため文字列リテラルに置換。正しい。

**`rails/app/controllers/settings/account/deletions/create_controller.rb`**: `root_path` → `"/"` への変更。正しい。

**`rails/app/controllers/settings/emails/update_controller.rb`**: `edit_email_confirmation_path` → `"/email_confirmation/edit"` への変更。正しい。

**`rails/app/controllers/spaces/settings/deletions/create_controller.rb`**: `root_path` → `"/"` への変更。正しい。

**`rails/app/views/settings/show_view.html.erb`**: `user_session_path` → `"/user_session"` への変更。`button_to` の `method: :delete` と組み合わせて正しく動作する。

**`rails/app/components/footers/global_component.html.erb`**: `root_path` → `"/"` への変更。正しい。

**`rails/app/components/links/brand_icon_component.html.erb`**: `root_path` → `"/"` への変更。正しい。

**`rails/app/components/navbars/bottom_component.html.erb`**: `root_path` → `"/"` および `sign_in_path` → `"/sign_in"` への変更。正しい。

**`rails/app/components/sidebar/content_component.html.erb`**: `root_path` → `"/"` および `sign_in_path` → `"/sign_in"` への変更。正しい。

**`rails/app/controllers/test/sign_in/create_controller.rb`**: 新規追加。テスト環境専用のサインインコントローラー。`if Rails.env.test?` でルートが保護されている。`Authenticatable` concern の `sign_in` メソッドを使用してセッションを作成し、`/home` にリダイレクト。システムテストの高速化に寄与する適切な実装。

**`rails/spec/support/request_spec_config.rb`**: リクエストスペックの `sign_in` ヘルパーを、削除された `POST /user_session` エンドポイント経由から、直接 `user_session_records.start!` + Cookie 設定に変更。`sign_in_with_2fa` は `sign_in` に委譲するのみ（2FA検証もGo版で処理されるため）。正しい。

**`rails/spec/support/system_spec_config.rb`**: システムスペックの `sign_in` ヘルパーを、フォーム入力方式から `/_test/sign_in?user_id=` エンドポイント経由に変更。`expect(page).to have_current_path("/home")` で遷移を確認。正しい。

**`rails/spec/system/global_hotkey_spec.rb`**: `root_path` → `"/home"` への変更。サインイン済みユーザーのホームページは `/home` であるため正しい。

**`rails/spec/requests/attachments/presigns/create_spec.rb`**: `sign_in_path` → `"/sign_in"` への変更。正しい。

**`rails/spec/requests/search/show_spec.rb`**: `sign_in_path` → `"/sign_in"` への変更、および `delete user_session_path` 行の削除（サインアウトしなくても未ログイン状態をテストできる）。正しい。

**`rails/spec/requests/settings/account/deletions/create_spec.rb`**: `root_path` → `"/"` への変更。正しい。

## 設計との整合性チェック

### 削除対象ファイルの網羅性

設計書の「削除対象ファイル一覧」に記載されたすべてのファイルが正しく削除されていることを確認した。

| カテゴリ                 | 設計書の想定 | 実際の削除数 | 状態 |
| ------------------------ | ------------ | ------------ | ---- |
| コントローラー           | 19           | 19           | OK   |
| コントローラーコンサーン | 1            | 1            | OK   |
| ビュー                   | 19           | 19           | OK   |
| フォーム                 | 7            | 7            | OK   |
| サービス                 | 5            | 5            | OK   |
| リポジトリ               | 1            | 1            | OK   |
| テスト                   | 19           | 19           | OK   |
| Sorbet RBI               | 8            | 8            | OK   |

### 削除してはいけないファイルの保全

設計書の「削除してはいけないファイル（共有依存）」に記載されたファイルがすべて保全されていることを確認した。

- `app/records/user_session_record.rb`: 保全 OK
- `app/records/email_confirmation_record.rb`: 保全 OK
- `app/records/user_record.rb`: 保全 OK
- `app/models/user_session.rb`: 保全 OK
- `app/models/email_confirmation_event.rb`: 保全 OK
- `app/services/user_sessions/destroy_service.rb`: 保全 OK
- `app/services/email_confirmations/create_service.rb`: 保全 OK
- `app/services/accounts/destroy_service.rb`: 保全 OK
- `app/services/accounts/soft_destroy_service.rb`: 保全 OK
- `app/mailers/email_confirmation_mailer.rb`: 保全 OK
- `app/forms/form_concerns/password_validatable.rb`: 保全 OK
- `app/forms/form_concerns/password_authenticatable.rb`: 保全 OK
- `app/controllers/controller_concerns/authenticatable.rb`: 保全 OK（修正のみ）
- `app/controllers/home/show_controller.rb`: 保全 OK
- `spec/factories/email_confirmation_record.rb`: 保全 OK

### ルートヘルパーの置換

削除されたルート定義に対応するルートヘルパー（`sign_in_path`, `root_path`, `user_session_path`, `edit_email_confirmation_path`）が、`app/` および `spec/` 内で文字列リテラルに正しく置換されていることを確認した。`grep` で残存参照がないことを検証済み。

### `email_confirmations/creation_form.rb` の削除タイミング

設計書のフェーズ4-1では「`creation_form` はフェーズ5で削除」と記載されているが、実際にはフェーズ4で削除されている。ただし、このフォームの参照元はすべて削除済みコントローラーのみであり、`app/` 内に残存参照がないことを確認した。全フェーズを一括実装しているため、フェーズ順序の差異は問題ない。

## 総合評価

**評価**: Approve

**総評**:

設計書に記載された全要件を正確に実装している。

- **設計との整合性**: 削除対象ファイル79件すべてが正しく削除され、保全すべき共有依存ファイルはすべて保全されている
- **ルートヘルパーの置換**: 削除されたルートに対応するヘルパーが文字列リテラルに適切に置換され、残存参照なし
- **テストヘルパーの更新**: リクエストスペック・システムスペックの `sign_in` ヘルパーが、削除されたエンドポイントに依存しない方式に適切に変更されている
- **テスト用エンドポイントの追加**: `/_test/sign_in` コントローラーが `Rails.env.test?` ガード付きで適切に追加されている
- **セキュリティ**: テスト用エンドポイントは本番環境で有効にならない設計。既存の認証・認可ロジック（`authenticatable.rb`）は保全されている
