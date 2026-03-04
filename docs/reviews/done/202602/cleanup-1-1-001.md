# コードレビュー: cleanup-1-1

## レビュー情報

| 項目                   | 内容                                                        |
| ---------------------- | ----------------------------------------------------------- |
| レビュー日             | 2026-02-08                                                  |
| 対象ブランチ           | cleanup-1-1                                                 |
| ベースブランチ         | cleanup                                                     |
| 設計書（指定があれば） | docs/designs/1_doing/rails-cleanup-go-migrated-endpoints.md |
| 変更ファイル数         | 39 ファイル                                                 |
| 変更行数（実装）       | +233 / -331 行                                              |
| 変更行数（テスト）     | +19 / -19 行                                                |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@rails/CLAUDE.md](/workspace/rails/CLAUDE.md) - Rails版の開発ガイド
- [@CLAUDE.md#コメントのガイドライン](/workspace/CLAUDE.md) - コメントのガイドライン
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `rails/config/routes.rb`
- [x] `rails/app/controllers/controller_concerns/authenticatable.rb`
- [x] `rails/app/controllers/controller_concerns/email_confirmation_findable.rb`
- [x] `rails/app/controllers/email_confirmations/create_controller.rb`
- [x] `rails/app/controllers/email_confirmations/update_controller.rb`
- [x] `rails/app/controllers/password_resets/create_controller.rb`
- [x] `rails/app/controllers/passwords/update_controller.rb`
- [x] `rails/app/controllers/settings/account/deletions/create_controller.rb`
- [x] `rails/app/controllers/settings/emails/update_controller.rb`
- [x] `rails/app/controllers/sign_in/two_factors/create_controller.rb`
- [x] `rails/app/controllers/sign_in/two_factors/new_controller.rb`
- [x] `rails/app/controllers/sign_in/two_factors/recoveries/create_controller.rb`
- [x] `rails/app/controllers/sign_in/two_factors/recoveries/new_controller.rb`
- [x] `rails/app/controllers/spaces/settings/deletions/create_controller.rb`
- [x] `rails/app/controllers/user_sessions/create_controller.rb`
- [x] `rails/app/controllers/user_sessions/destroy_controller.rb`
- [x] `rails/app/components/footers/global_component.html.erb`
- [x] `rails/app/components/links/brand_icon_component.html.erb`
- [x] `rails/app/components/navbars/bottom_component.html.erb`
- [x] `rails/app/components/sidebar/content_component.html.erb`
- [x] `rails/app/views/accounts/new_view.html.erb`
- [ ] `rails/app/views/email_confirmations/edit_view.html.erb`
- [x] `rails/app/views/password_resets/new_view.html.erb`
- [x] `rails/app/views/passwords/edit_view.html.erb`
- [x] `rails/app/views/settings/show_view.html.erb`
- [x] `rails/app/views/sign_in/show_view.html.erb`
- [x] `rails/app/views/sign_in/two_factors/new_view.html.erb`
- [ ] `rails/app/views/sign_in/two_factors/recoveries/new_view.html.erb`
- [x] `rails/app/views/sign_up/show_view.html.erb`
- [x] `rails/app/views/welcome/show_view.html.erb`

### 自動生成ファイル

- [x] `rails/sorbet/rbi/dsl/generated_path_helpers_module.rbi`
- [x] `rails/sorbet/rbi/dsl/generated_url_helpers_module.rbi`

### テストファイル

- [x] `rails/spec/requests/attachments/presigns/create_spec.rb`
- [x] `rails/spec/requests/password_resets/create_spec.rb`
- [x] `rails/spec/requests/search/show_spec.rb`
- [x] `rails/spec/requests/settings/account/deletions/create_spec.rb`
- [x] `rails/spec/support/request_spec_config.rb`
- [x] `rails/spec/system/global_hotkey_spec.rb`

### ドキュメント

- [x] `docs/designs/1_doing/rails-cleanup-go-migrated-endpoints.md`

## ファイルごとのレビュー結果

### `rails/app/views/sign_in/two_factors/recoveries/new_view.html.erb`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@rails/CLAUDE.md](/workspace/rails/CLAUDE.md) - コーディング規約

**問題点・改善提案**:

- **パスヘルパーの置き換え漏れ**: 27行目で `sign_in_two_factor_recovery_path` がまだ使用されているが、このnamed routeは `routes.rb` から削除されている（テスト環境用ルートにも `as:` が付いていないため未定義）。本番・開発環境で `NoMethodError` が発生する。

  ```erb
  <!-- 問題のあるコード（27行目） -->
  url: sign_in_two_factor_recovery_path
  ```

  **修正案**:

  ```erb
  url: "/sign_in/two_factor/recovery"
  ```

### `rails/app/views/email_confirmations/edit_view.html.erb`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - XSS対策
- [@rails/CLAUDE.md#セキュリティガイドライン](/workspace/rails/CLAUDE.md) - セキュリティガイドライン

**問題点・改善提案**:

- **XSSリスク**: 25行目で `params[:after]` が文字列補間によりURLに直接埋め込まれている。ユーザー入力がエスケープされずにHTMLに出力されるため、XSS攻撃の可能性がある。

  ```erb
  <!-- 問題のあるコード（25行目） -->
  url: "/email_confirmation#{params[:after] ? "?after=#{params[:after]}" : ""}"
  ```

  **修正案**:

  ```erb
  url: "/email_confirmation#{params[:after] ? "?after=#{ERB::Util.url_encode(params[:after])}" : ""}"
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 上記の修正案を適用する
  - [ ] 元の `email_confirmation_path(after: params[:after])` は Rails が自動でエスケープしていたので、修正前も同等のXSSリスクはなかった。ただし文字列補間に変えた以上エスケープは必要
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計との整合性チェック

### 設計書の要件と実装の比較

| 要件                                 | 状態 | 備考                                                                            |
| ------------------------------------ | ---- | ------------------------------------------------------------------------------- |
| routes.rbから20行のルート定義を削除  | ✅   | 本番/開発環境からは削除済み。テスト環境用に `if Rails.env.test?` ブロックで維持 |
| 削除後に `bin/check` を実行して確認  | ⚠️   | レビューでは確認不可。コミットメッセージから推測すると実行済みと思われる        |
| フェーズ1のスコープ（routes.rbのみ） | ⚠️   | routes.rb以外にも多数の変更あり（下記参照）                                     |

### 設計との乖離

設計書のタスク1-1は「routes.rb から Go 移行済みルート定義を削除」とあり、**想定ファイル数は約1ファイル**と記載されている。しかし実際の変更は39ファイルに及び、以下の追加作業が含まれている：

1. **パスヘルパーの文字列リテラル置き換え**: `root_path` → `"/"`、`sign_in_path` → `"/sign_in"` など、コントローラー・ビュー・コンポーネント・テスト全般で named route ヘルパーを文字列リテラルに置き換え
2. **Sorbet RBIファイルの更新**: 削除されたnamed routeに対応するRBI定義の削除
3. **テストコードの更新**: テストヘルパーやスペックでのパスヘルパー使用箇所の置き換え

これは設計の想定を超えているが、**ルート定義を削除した以上、named routeヘルパーの参照箇所も更新しなければビルドが壊れるため、妥当な対応**と判断できる。ただし、設計書にはこの影響範囲について事前の記載がなかった。

### テスト環境用ルートの維持について

設計書には「テスト環境でのみルートを維持する」という方針は記載されていないが、コメントで「コントローラー・テストの削除（フェーズ2〜5）に合わせて段階的に削除する」と説明されており、段階的移行の方針と整合している。

## 総合評価

**評価**: Request Changes

**総評**:

フェーズ1のルート定義削除に伴い、named routeヘルパーの文字列リテラル置き換えを全面的に実施しており、方針としては妥当。テスト環境用にルートを維持するアプローチも段階的移行として合理的。

ただし、以下の2点で修正が必要：

1. **`sign_in_two_factor_recovery_path` の置き換え漏れ**（`recoveries/new_view.html.erb:27`）: named routeが削除されているため、本番環境で `NoMethodError` が発生する致命的なバグ
2. **`params[:after]` のXSSリスク**（`email_confirmations/edit_view.html.erb:25`）: 元の `email_confirmation_path(after: ...)` ヘルパーはRailsが自動でURLエンコードしていたが、文字列補間に変更したことでエスケープが失われている
