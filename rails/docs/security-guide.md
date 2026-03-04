# セキュリティガイドライン

このドキュメントは、Rails 版 Wikino でのセキュリティベストプラクティスを説明します。

## 基本方針

Web アプリケーションのセキュリティは**最優先事項**です。以下のガイドラインを必ず守ってください。

## CSRF 対策

- `protect_from_forgery` がデフォルトで有効
- フォームには `form_with` ヘルパーを使用（CSRF トークンが自動的に含まれる）

## XSS 対策

- ERB の自動エスケープを活用
- `raw`/`html_safe` は慎重に使用
- ユーザー入力を信頼しない

## SQL インジェクション対策

- ActiveRecord のプリペアドステートメントを使用
- プレースホルダーを使用

```ruby
# ❌ 悪い例：文字列補間でクエリを構築
UserRecord.where("email = '#{params[:email]}'")

# ✅ 良い例：プレースホルダーを使用
UserRecord.where(email: params[:email])
UserRecord.where("email = ?", params[:email])
```

## 認証

- bcrypt（`has_secure_password`）でパスワードを管理
- 平文パスワードをログに出力しない

## Strong Parameters

- すべてのコントローラーで Strong Parameters を使用
- 許可するパラメータを明示的に指定

```ruby
# ✅ 良い例
private def page_params
  params.require(:page).permit(:title, :body)
end
```

## セキュリティチェックリスト

新機能を実装する際は、以下を必ず確認してください：

- [ ] フォームに CSRF トークンが含まれているか（`form_with` を使用しているか）
- [ ] ユーザー入力をバリデーションしているか
- [ ] SQL クエリはプリペアドステートメントを使用しているか
- [ ] パスワードは bcrypt でハッシュ化されているか
- [ ] Strong Parameters を使用しているか
- [ ] 認証・認可チェックを行っているか
- [ ] エラーメッセージに詳細な情報を含めていないか
