# Wikino開発ガイド

WikinoはWikiアプリです。
ユーザーはスペースと呼ばれる場所にページを作成することができ、ページ間をリンクで繋げることができます。

## 技術スタック

主に以下のツールを使用しています。

- **バックエンド**: Ruby, Ruby on Rails
- **フロントエンド**: TypeScript, Hotwire, Tailwind CSS
- **パッケージマネージャー**: Bundler, Yarn
- **テスティングフレームワーク**: RSpec, FactoryBot
- **Linter**: Standard (Ruby), ERB Lint (ERB)
- **型検査**: Sorbet
- **データベース**: PostgreSQL

## リポジトリの構造

基本的にRailsプロジェクトのディレクトリ構成となっていますが、一部独自のディレクトリがあります。

- `app/assets`
  - 画像やCSSのファイルが格納されています
- `app/components`
  - [ViewComponent](https://viewcomponent.org) を使用したコンポーネントが定義されています
- `app/controllers`
  - Railsのコントローラー
  - 1つのアクションごとに1つのコントローラーを定義しています
- `app/javascript`
  - フロントエンドJavaScriptの実装が格納されています
  - [Hotwire](https://hotwired.dev)で実装されています
- `app/jobs`
  - Active Job
- `app/mailers`
  - Action Mailer
- `app/models`
  - `ActiveModel::Model` をincludeしたクラスが定義されています
- `app/records`
  - `ActiveRecord::Base` を継承したクラスが定義されている
  - データベースのテーブルと1:1の関係となる
- `app/repositories`
  - Recordを介してデータを取得したり保存し、Modelを返す
- `app/validators`
  - カスタムバリデーション
- `app/views`
  - ViewComponentを使用したビュー

## コマンドリファレンス

- `bin/rspec` # テストを実行する
- `bin/rails sorbet:update` # 型を更新する
- `yarn prettier . --write` # Prettierで整形する
- `bin/erb_lint --lint-all` # ERB Lintを実行する
- `bin/standardrb` # Standardを実行する
- `bin/srb tc` # 型チェックをする
- `bin/check` # 各種チェックをまとめて行う

## コーディング規約

### 共通

- 説明的な命名規則を採用してください
- コメントを適切に追加し、コードの可読性を高めてください
- セキュリティのベストプラクティスに従った実装をしてください
- コードベースの一貫性を保つように心がけてください

### Ruby

- 文字列はダブルクオートで書いてください
- 後置ifは使わないでください

```rb
# NG
puts "Hello" if cond?

# OK
if cond?
  puts "Hello"
end
```

- ハッシュのキーと変数名が同じ場合は、省略記法を使用してください

```rb
# NG
def method(user:)
  User.create(user: user)
end

# OK
def method(user:)
  User.create(user:)
end
```

- 新たにファイルを作成するときはマジックコメントを書いてください
  - Sorbetで使用する `typed: xxx` はなるべく `strict` を指定します

```rb
# typed: strict
# frozen_string_literal: true

# ...
```

### Rails

#### 依存関係

- 主要なクラスの依存関係から外れないように書いてください
  - Railsなど外部のライブラリが提供するクラスなどはどのクラスからでも呼び出して大丈夫です

| クラス          | 説明                               | 依存先                             |
| --------------- | ---------------------------------- | ---------------------------------- |
| Component       | 再利用可能なUI要素                 | `Component`, `Model`               |
| Controller      | HTTPリクエスト処理と応答の調整     | `Model`, `Repository`, `View`      |
| Job             | ジョブの定義                       | `Model`, `Repository`              |
| Mailer          | メール送信                         | `Model`, `View`                    |
| Model           | データ構造とドメインロジックを表現 | `Model`, `ModelValidator`          |
| ModelValidator  | Modelのカスタムバリデーション      | -                                  |
| Record          | DBのテーブルから取得・保存する     | `Model`, `Record`                  |
| RecordValidator | Recordのカスタムバリデーション     | `Record`                           |
| Repository      | ビジネスロジックのカプセル化       | `Job`, `Mailer`, `Model`, `Record` |
| View            | 表示処理                           | `Component`, `Model`               |

#### Model

- `Model` を定義するときは `ApplicationModel` を継承するようにしてください

```rb
# typed: strict
# frozen_string_literal: true

class User < ApplicationModel
end
```

### CSS

- Tailwind CSSを使用して書いてください

## セキュリティ

- `.env.local` や `.env.*.local` など、機密情報を含むファイルの読み取りはしないでください
