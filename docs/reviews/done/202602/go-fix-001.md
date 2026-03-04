# コードレビュー: go-fix

## レビュー情報

| 項目                   | 内容                                        |
| ---------------------- | ------------------------------------------- |
| レビュー日             | 2026-02-08                                  |
| 対象ブランチ           | go-fix                                      |
| ベースブランチ         | go (origin/go)                              |
| 設計書（指定があれば） | なし                                        |
| 変更ファイル数         | 84 ファイル                                 |
| 変更行数（実装）       | +195 / -27 行 (Go) / +810 / -665 行 (templ) |
| 変更行数（テスト）     | +112 / -1 行                                |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド

## 変更ファイル一覧

### 実装ファイル (Go)

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/sign_in/create.go`
- [x] `go/internal/handler/sign_in/handler.go`
- [x] `go/internal/handler/sign_in/new.go`
- [ ] `go/internal/middleware/csrf.go`
- [x] `go/internal/middleware/reverse_proxy.go`
- [ ] `go/internal/templates/helper.go`
- [x] `go/internal/worker/client.go`
- [x] `go/internal/worker/send_password_reset.go`
- [x] `go/internal/templates/emails/password_reset/data.go`

### 実装ファイル (templ)

- [x] `go/internal/templates/components/flash.templ`
- [x] `go/internal/templates/components/footer.templ`
- [x] `go/internal/templates/components/form_errors.templ`
- [ ] `go/internal/templates/components/head.templ`
- [x] `go/internal/templates/layouts/plain.templ`
- [x] `go/internal/templates/pages/account/new.templ`
- [x] `go/internal/templates/pages/email_confirmation/edit.templ`
- [x] `go/internal/templates/pages/password/edit.templ`
- [x] `go/internal/templates/pages/password/reset.templ`
- [x] `go/internal/templates/pages/password/reset_sent.templ`
- [x] `go/internal/templates/pages/sign_in/new.templ`
- [ ] `go/internal/templates/pages/sign_in_two_factor/new.templ`
- [x] `go/internal/templates/pages/sign_in_two_factor/recovery_new.templ`
- [x] `go/internal/templates/pages/sign_up/new.templ`
- [x] `go/internal/templates/pages/welcome/show.templ`
- [x] `go/internal/templates/emails/password_reset/en_html.templ`
- [x] `go/internal/templates/emails/password_reset/en_text.templ`
- [x] `go/internal/templates/emails/password_reset/ja_html.templ`
- [x] `go/internal/templates/emails/password_reset/ja_text.templ`

### テストファイル

- [x] `go/internal/handler/sign_in/create_test.go`
- [x] `go/internal/handler/sign_in/new_test.go`
- [x] `go/internal/handler/welcome/show_test.go`
- [x] `go/internal/middleware/csrf_test.go`

### I18n

- [ ] `go/internal/i18n/locales/ja.toml`
- [ ] `go/internal/i18n/locales/en.toml`

### 設定・インフラ

- [x] `Dockerfile.dev`
- [x] `docker-compose.yml`
- [x] `mise.toml`
- [x] `go/mise.toml`
- [x] `go/package.json`
- [x] `go/pnpm-lock.yaml`
- [x] `go/db/schema.sql`
- [x] `go/web/style.css`

### ドキュメント

- [x] `CLAUDE.md`
- [x] `docs/reviews/template.md`
- [x] `docs/designs/3_done/202602/unified-dev-container.md`

### Rails 関連

- [x] `rails/CLAUDE.md`
- [x] `rails/Dockerfile.dev` (削除)
- [x] `go/Dockerfile.dev` (削除)
- [x] `rails/app/controllers/controller_concerns/authenticatable.rb`
- [x] `rails/config/locales/messages.ja.yml`
- [x] `rails/db/queue_structure.sql`
- [x] `rails/spec/requests/welcome/show_spec.rb` (削除)
- [x] `rails/spec/support/capybara.rb`
- [x] `rails/spec/system/welcome/show_spec.rb` (削除)

### 自動生成ファイル（レビュー対象外）

- `go/internal/templates/components/flash_templ.go`
- `go/internal/templates/components/footer_templ.go`
- `go/internal/templates/components/form_errors_templ.go`
- `go/internal/templates/components/head_templ.go`
- `go/internal/templates/layouts/plain_templ.go`
- `go/internal/templates/pages/account/new_templ.go`
- `go/internal/templates/pages/email_confirmation/edit_templ.go`
- `go/internal/templates/pages/password/edit_templ.go`
- `go/internal/templates/pages/password/reset_templ.go`
- `go/internal/templates/pages/password/reset_sent_templ.go`
- `go/internal/templates/pages/sign_in/new_templ.go`
- `go/internal/templates/pages/sign_in_two_factor/new_templ.go`
- `go/internal/templates/pages/sign_in_two_factor/recovery_new_templ.go`
- `go/internal/templates/pages/sign_up/new_templ.go`
- `go/internal/templates/pages/welcome/show_templ.go`
- `go/internal/templates/emails/password_reset/en_html_templ.go`
- `go/internal/templates/emails/password_reset/en_text_templ.go`
- `go/internal/templates/emails/password_reset/ja_html_templ.go`
- `go/internal/templates/emails/password_reset/ja_text_templ.go`
- `go/static/images/welcome/*.png`

## ファイルごとのレビュー結果

### `go/internal/templates/helper.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - コーディング規約
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

**問題点・改善提案**:

- **バグ: `Icon()` のフォールバック処理が壊れている**: `icons` マップのキーが `"info"` から `"info-regular"` にリネームされたが、60-61行目のフォールバック処理は `icons["info"]` を参照したまま。存在しないキーを参照するため、フォールバック時にSVGが空文字列になる。

  ```go
  // 問題のあるコード（61行目）
  svg = icons["info"]
  ```

  **修正案**:

  ```go
  svg = icons["info-regular"]
  ```

### `go/internal/middleware/csrf.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - CSRF対策
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - セキュリティガイドライン

**問題点・改善提案**:

- **セキュリティ: CSRFスキップパスのプレフィックスマッチが過度に広い**: `strings.HasPrefix(r.URL.Path, "/user_session")` を使用しているため、`/user_session_attack`、`/user_sessions` など意図しないパスもCSRF検証をスキップしてしまう。`DELETE /user_session` は単一のパスなので、完全一致を使用すべき。

  ```go
  // 問題のあるコード（58-63行目）
  for _, path := range csrfSkipPaths {
      if strings.HasPrefix(r.URL.Path, path) {
          next.ServeHTTP(w, r)
          return
      }
  }
  ```

  **修正案**:

  ```go
  for _, path := range csrfSkipPaths {
      if r.URL.Path == path {
          next.ServeHTTP(w, r)
          return
      }
  }
  ```

### `go/internal/templates/components/head.templ`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@CLAUDE.md](/workspace/CLAUDE.md) - コメントのガイドライン

**問題点・改善提案**:

- **コメントガイドライン違反: コメントアウトされたコードが残っている**: 43-51行目のダークモード検出スクリプトが全てコメントアウトされているが、そのまま残されている。空の `<script>` タグがHTML出力に含まれる。コメントガイドラインによると、実装の変遷に関するコメント（削除されたコードなど）はgit履歴で確認できるため、コードから削除すべき。

  ```templ
  // 問題のあるコード（42-51行目）
  // ダークモード検出スクリプト
  <script>
      // (() => {
      // 	try {
      // 		if (matchMedia("(prefers-color-scheme: dark)").matches) {
      // 			document.documentElement.classList.add("dark");
      // 		}
      // 	} catch (_) {}
      // })();
  </script>
  ```

  **修正案**: コメントアウトされたコードと空の `<script>` タグを完全に削除する。

### `go/internal/templates/pages/sign_in_two_factor/new.templ`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド

**問題点・改善提案**:

- **インデントの不整合**: 34行目の `</div>` の閉じタグが、対応する26行目の `<div class="flex flex-col gap-2">` の開始タグよりインデントが浅い（タブ1つ少ない）。templ はコンパイル可能だが、HTML構造の可読性が低下し、ネスト構造の誤りを見落としやすくなる。

  ```templ
  // 問題のあるコード（26行目と34行目）
  		<div class="flex flex-col gap-2">   // タブ2つ
  			...
  	</div>                                  // タブ1つ（タブ2つであるべき）
  ```

  **修正案**: 閉じタグのインデントを開始タグと合わせる。

### `go/internal/i18n/locales/ja.toml` / `go/internal/i18n/locales/en.toml`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド（descriptionを必ず記述）

**問題点・改善提案**:

- **I18nガイドライン違反: 新規追加された翻訳キーに `description` フィールドがない**: `sign_up_email_instruction` と `password_reset_email_subject` の2つの新規キーに `description` が欠落している。I18nガイドでは「descriptionを必ず記述」とされている。

  ```toml
  # 問題のあるコード
  [sign_up_email_instruction]
  other = "確認コードをお送りします"

  [password_reset_email_subject]
  other = "[Wikino] パスワードリセット"
  ```

  **修正案**:

  ```toml
  [sign_up_email_instruction]
  description = "サインアップ時のメール入力フィールドの説明"
  other = "確認コードをお送りします"

  [password_reset_email_subject]
  description = "パスワードリセットメールの件名"
  other = "[Wikino] パスワードリセット"
  ```

- **未使用の翻訳キーが残っている可能性**: `sign_in_two_factor_recovery_enter_code` キーが `recovery_new.templ` から削除されたが、`ja.toml` と `en.toml` にはキーが残っている。コードベース内で他に使用箇所がなければ削除すべき。

## 総合評価

**評価**: Request Changes

**総評**:

このブランチは、開発コンテナの統合、UI改善、2FA対応、パスワードリセットメール送信ワーカーの実装、ミドルウェア順序の修正など、多岐にわたる変更を含んでいる。全体的にアーキテクチャガイドライン、命名規則、テスト戦略に従っており、コードの品質は高い。

**良い点**:

- ミドルウェア順序の修正（リバースプロキシの前にMethod Overrideを配置しない）は正しいバグ修正
- 2FA対応のハンドラー実装は、認証フロー的に適切な位置でチェックを実施
- CSRFスキップパスの追加はRails/Go共存のために必要な対応
- templテンプレートは構造体ベースの引数パターンに統一されている
- テストが新規機能（CSRFスキップパス）に対して追加されている
- UI改善はcard コンポーネントで統一的に適用されている

**要修正点**:

1. **バグ**: `helper.go` の `Icon()` フォールバックが壊れている（`icons["info"]` → `icons["info-regular"]`）
2. **セキュリティ**: CSRFスキップパスのプレフィックスマッチを完全一致に変更すべき
3. **I18n**: 新規翻訳キーに `description` フィールドが欠落している

---

## 質問と回答

### Q1: CSRFスキップパスの完全一致への変更

**種別**: 必須

**背景**:

現在 `strings.HasPrefix(r.URL.Path, "/user_session")` を使用しており、`/user_session` で始まるすべてのパスがCSRF検証をスキップします。意図しないパスがスキップされるリスクがあります。

**選択肢**:

- [x] 完全一致（`r.URL.Path == path`）に変更する
- [ ] パスの末尾に `/` またはクエリパラメータのみ許容するパターンに変更する
- [ ] その他（下の回答欄に記入）

**回答**:

```
（ここに回答を記入）
```

### Q2: コメントアウトされたダークモード検出スクリプトの扱い

**種別**: 推奨

**背景**:

`head.templ` のダークモード検出スクリプトが全てコメントアウトされています。コメントガイドラインに従い削除すべきですが、今後ダークモード対応の予定がある場合は一時的に残す理由があるかもしれません。

**選択肢**:

- [ ] 削除する（git履歴で復元可能）
- [ ] 残す（ダークモード対応の予定がある）
- [x] その他（下の回答欄に記入）

**回答**:

```
今後コメント化を解除する予定なので、その旨をコメントとして残しつつコメントアウトは残してください。
```

### Q3: 未使用の翻訳キー `sign_in_two_factor_recovery_enter_code` の削除

**種別**: 推奨

**背景**:

`recovery_new.templ` からこの翻訳キーの参照が削除されましたが、`ja.toml` と `en.toml` にはキーが残っています。Grepで確認したところ、Go コード内の他の箇所では使用されていません。

**選択肢**:

- [x] 削除する
- [ ] 残す（他で使用予定がある）

**回答**:

```
（ここに回答を記入）
```
