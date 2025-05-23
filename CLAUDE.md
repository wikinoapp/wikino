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

- `app/assets`: 画像やCSSのファイル
- `app/components`: ViewComponentを使用したコンポーネント
- `app/controllers`: Railsのコントローラー
- `app/forms`: フォームオブジェクト
- `app/javascript`: Hotwireで実装されたフロントエンド
- `app/jobs`: Active Job
- `app/mailers`: Action Mailer
- `app/models`: POROや構造体など
- `app/policies`: 認可ルールが書かれたクラス
- `app/records`: `ActiveRecord::Base` を継承したデータベースのテーブルと1:1の関係となるクラス
- `app/repositories`: RecordをModelに変換するクラス
- `app/services`: サービスクラス
- `app/validators`: カスタムバリデーション
- `app/views`: ViewComponentを使用したビュー

## コマンドリファレンス

- `bin/rspec`: テストを実行する
- `bin/rails sorbet:update`: 型を更新する
- `yarn prettier . --write`: Prettierで整形する
- `bin/erb_lint --lint-all`: ERB Lintを実行する
- `bin/standardrb`: Standardを実行する
- `bin/srb tc`: 型チェックをする
- `bin/check`: 各種チェックをまとめて行う

## コーディング規約

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

| クラス     | 説明                               | 依存先                                                        |
| ---------- | ---------------------------------- | ------------------------------------------------------------- |
| Component  | 再利用可能なUI要素                 | `Component`, `Form`, `Model`                                  |
| Controller | HTTPリクエスト処理と応答の調整     | `Form`, `Model`, `Record`,<br>`Repository`, `Service`, `View` |
| Form       | フォームオブジェクト               | `Record`, `Validator`                                         |
| Job        | ジョブの定義                       | `Service`                                                     |
| Mailer     | メール送信                         | `Model`, `Record`, `Repository`, `View`                       |
| Model      | データ構造とドメインロジックを表現 | `Model`                                                       |
| Policy     | 認可ルール                         | `Record`                                                      |
| Record     | DBのテーブルから取得・保存する     | `Record`                                                      |
| Repository | ModelとRecordの変換                | `Model`, `Record`                                             |
| Service    | ビジネスロジックのカプセル化       | `Job`, `Mailer`, `Record`                                     |
| Validator  | カスタムバリデーション             | `Record`                                                      |
| View       | 表示処理                           | `Component`, `Form`, `Model`                                  |

#### Controller

- コントローラー1つにつき1つのアクションを定義してください

```rb
# typed: true
# frozen_string_literal: true

module Pages
  class ShowController < ApplicationController
    sig { returns(T.untyped) }
    def call
    end
  end
end
```

#### Form

- 共通で使うバリデーションは `form_concerns` 配下にモジュールとして定義してください

##### 命名規則

- クラス名は `(リソース名)Form::(名詞)` とします
  - 例: `SpaceForm::Creation`

```rb
# typed: strict
# frozen_string_literal: true

module SpaceForm
  class Creation < ApplicationForm
  end
end
```

#### Model

- `Model` では `T::Struct` や `T::Enum` を継承したクラスや、PORO (Plain Old Ruby Object) を定義します

```rb
# typed: strict
# frozen_string_literal: true

class User < T::Struct
  extend T::Sig

  include T::Struct::ActsAsComparable

  const :database_id, T::Wikino::DatabaseId
  # ...
end
```

#### Service

##### 命名規則

- クラス名は `(リソース名)Service::(動詞)` とします

  - 例: `PageService::Update`

- メソッド名は `#call` とします

```rb
# typed: strict
# frozen_string_literal: true

module PageService
  class Update
    sig { void }
    def call
      # ...
    end
  end
end
```

#### View

- コントローラーのアクションごとに定義します

```rb
# typed: strict
# frozen_string_literal: true

module Pages
  class ShowView < ApplicationView
    sig { params(page: Page).void }
    def initialize(page:)
      @page = page
    end

    sig { returns(Page) }
    attr_reader :page
    private :page
  end
end
```

### RSpec

- `context` ブロックは使用せず、`it` の中にケースを書いてください

```rb
# NG
RSpec.describe "GET /", type: :request do
  context "ログインしていないとき" do
    it "ログインページにリダイレクトすること" do
    end
  end
end

# OK
RSpec.describe "GET /", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
  end
end
```

### CSS

- Tailwind CSSを使用して書いてください
