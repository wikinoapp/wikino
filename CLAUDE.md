# Wikino開発ガイド

WikinoはWikiアプリケーションです。
ユーザーは「スペース」と呼ばれる場所にページを作成し、ページ間をリンクで繋げることができます。

## 技術スタック

### バックエンド

- Ruby 3.4.4
- Ruby on Rails 8.0.0
- PostgreSQL
- Sorbet（型検査）
- Active Job（Solid Queue）

### フロントエンド

- TypeScript
- Hotwire (Turbo + Stimulus)
- Tailwind CSS 4
- CodeMirror 6（ページエディタ）

### ツール・ライブラリ

- パッケージマネージャー: Bundler, pnpm
- テスティング: RSpec, FactoryBot
- Linter: Standard (Ruby), ERB Lint, ESLint, Prettier
- ViewComponent, html-pipeline, meta-tags

## プロジェクト構造

### app/ディレクトリの構成と責務

| ディレクトリ      | 責務                   | 説明                                              |
| ----------------- | ---------------------- | ------------------------------------------------- |
| **controllers/**  | HTTPリクエスト処理     | 1アクション1コントローラー、`#call`メソッドで実装 |
| **records/**      | DBテーブル操作         | ActiveRecord::Base継承、1テーブル1レコード        |
| **models/**       | ドメインロジック       | PORO、データベースアクセスなし                    |
| **repositories/** | データ変換             | RecordとModel間の変換                             |
| **services/**     | ビジネスロジック       | **データ永続化を伴う処理のみ**実装                |
| **forms/**        | フォーム処理           | バリデーションとデータ変換                        |
| **components/**   | UIコンポーネント       | ViewComponent、再利用可能なUI要素                 |
| **views/**        | ビュー                 | ViewComponent使用、DB直接アクセス禁止             |
| **policies/**     | 認可ルール             | 権限管理                                          |
| **validators/**   | カスタムバリデーション | ActiveModelバリデーター拡張                       |
| **jobs/**         | 非同期処理             | 最小限のロジック、主にService呼び出し             |
| **mailers/**      | メール送信             | Action Mailer                                     |

## Railsクラス設計と依存関係

### クラス間の依存関係ルール

| クラス     | 依存可能な先                                   |
| ---------- | ---------------------------------------------- |
| Component  | Component, Form, Model                         |
| Controller | Form, Model, Record, Repository, Service, View |
| Form       | Record, Validator                              |
| Job        | Service                                        |
| Mailer     | Model, Record, Repository, View                |
| Model      | Model                                          |
| Policy     | Record                                         |
| Record     | Record                                         |
| Repository | Model, Record                                  |
| Service    | Job, Mailer, Record                            |
| Validator  | Record                                         |
| View       | Component, Form, Model                         |

### ServiceとJobの依存関係について

ServiceとJobの間には相互依存が存在しますが、以下のルールで循環依存を回避します：

- **Service → Job**: `perform_later`メソッドによるキューへの追加のみ許可
- **Job → Service**: ジョブ実行時のService呼び出しは許可
- **重要**: ServiceからJobインスタンスの直接実行（`perform`メソッド）は禁止

```ruby
# ✅ 良い例：Serviceからジョブをキューに追加
class Users::CreateService
  def call
    user = UserRecord.create!(...)
    Users::SendWelcomeEmailJob.perform_later(user.id)  # キューに追加のみ
  end
end

# ❌ 悪い例：Serviceからジョブを直接実行
class Users::CreateService
  def call
    user = UserRecord.create!(...)
    Users::SendWelcomeEmailJob.new.perform(user.id)  # 直接実行は禁止
  end
end
```

### 命名規則

- Controller: `(ModelPlural)::(ActionName)Controller`
- Service: `(ModelPlural)::(Verb)Service`
- Form: `(ModelPlural)::(Noun)Form`
- Repository: `(Model)Repository`
- View: `(ModelPlural)::(ActionName)View`
- Component: `(UIComponentPlural)::(Noun)Component`

## コーディング規約（プロジェクト固有）

### Ruby

```ruby
# typed: strict
# frozen_string_literal: true

class Example
  # ✅ 文字列はダブルクオート
  name = "example"

  # ✅ ハッシュの省略記法
  {user:, name:}

  # ❌
  {user: user, name: name}

  # ✅ プライベートメソッドは private def
  private def process_value(value)
    value.upcase
  end

  # ✅ プロテクテッドメソッドは protected def
  protected def shared_method(value)
    value.downcase
  end

  # ❌ 後置ifは使用しない
  # return if value.nil? # 悪い例

  # ✅
  if value.nil?
    return
  end

  # ❌ attr_readerにprivateブロックを使用しない
  # private
  # attr_reader :user_record

  # ✅ attr_readerは個別にprivate指定
  attr_reader :user_record
  private :user_record

  # ✅ T.mustではなくnot_nil!を使用
  value.not_nil!
end
```

### ActiveRecord

```ruby
# ❌ includesは使用禁止
Model.includes(:association)

# ✅ 明示的にpreloadまたはeager_loadを使用
Model.preload(:association)   # 別クエリで取得（基本はこちら）
Model.eager_load(:association) # JOINで取得（関連テーブルでフィルタリング時）
```

### マイグレーション

```ruby
create_table :examples, id: false do |t|
  # ULIDを使用
  t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
end
```

### 型定義

```ruby
# データベースIDの型はTypes::DatabaseIdを使用
sig { params(space_record_id: Types::DatabaseId).returns(T::Boolean) }
def in_same_space?(space_record_id:)
  # ...
end

# ❌ 単純なStringではなく
sig { params(space_record_id: String).returns(T::Boolean) }

# ✅ Types::DatabaseIdを使用
sig { params(space_record_id: Types::DatabaseId).returns(T::Boolean) }
```

### RSpec

```ruby
# ❌ context, let, described_classは使用しない
context "when xxx" do
  let(:user) { create(:user) }
end

# ✅ itブロック内で変数定義
it "xxxのとき、somethingすること" do
  user = FactoryBot.create(:user)
  # テスト実装
end

# ✅ FactoryBotで作成したレコードの変数名には_recordサフィックスを付ける
user_record = FactoryBot.create(:user_record)
space_record = FactoryBot.create(:space_record)
space_member_record = FactoryBot.create(:space_member_record, user_record:, space_record:)

# ❌ サフィックスなしの変数名は避ける
user = FactoryBot.create(:user_record)
space = FactoryBot.create(:space_record)
```

### I18n

翻訳ファイルは用途別に分類し、日本語と英語の両方を更新：

- `forms.(ja,en).yml`: フォーム関連
- `messages.(ja,en).yml`: メッセージ・説明文
- `meta.(ja,en).yml`: メタデータ
- `nouns.(ja,en).yml`: 名詞・ラベル

### JavaScript/TypeScript

#### HTTPリクエスト

```typescript
// ❌ fetchを使用しない
const response = await fetch("/api/endpoint", {
  method: "POST",
  headers: {
    "Content-Type": "application/json",
    "X-CSRF-Token": csrfToken,
  },
  body: JSON.stringify(data),
});

// ✅ @rails/request.jsを使用
import { post } from "@rails/request.js";

const response = await post("/api/endpoint", {
  body: data,
  responseKind: "json",
});
```

**重要**: Railsアプリケーション内でのHTTPリクエストには、`fetch`ではなく`@rails/request.js`パッケージを使用すること。CSRFトークンの管理が自動化され、Railsとの統合がよりシームレスになります。

## サービスクラスのルール

### サービスクラスを使用する場合

- ✅ データベースへの永続化を伴う処理
- ✅ 複数のモデル/レコードにまたがる複雑なビジネスロジックで永続化を伴うもの
- ✅ トランザクション管理が必要な処理

### サービスクラスを使用しない場合

- ❌ データベースへの永続化を伴わない処理（URL生成、データ変換など）
- ❌ 単一のモデル/レコードに閉じた処理（モデルやレコードのメソッドとして定義）

### トランザクション処理

**重要**: Serviceクラスでトランザクションを張る場合は、必ず `#with_transaction` メソッドを使用すること

```ruby
# ✅ 良い例：with_transactionを使用
module Users
  class CreateService < ApplicationService
    def call
      with_transaction do
        user = UserRecord.create!(...)
        ProfileRecord.create!(user:, ...)
      end
    end
  end
end

# ❌ 悪い例：transactionを直接使用
module Users
  class CreateService < ApplicationService
    def call
      ApplicationRecord.transaction do
        # with_transactionを使うべき
      end
    end
  end
end
```

**重要**: Controller、Job、Rakeタスク内で永続化処理を書く場合は、必ずServiceクラスを定義すること

## 開発ワークフロー

### 環境セットアップ

```bash
docker compose up      # Docker環境起動
mise install          # 依存関係インストール
bin/setup            # 初期セットアップ
bin/dev              # 開発環境起動
bin/rails server     # サーバー起動
```

## 作業完了ガイドライン

### タスク実装フロー

#### 1. タスク理解

- 要件を理解
- このガイドの固有ルールを確認

#### 2. 実装前の準備

- 既存コードの調査
- 特に以下を意識：
  - プライベートメソッドは `private def`
  - `includes` ではなく `preload` / `eager_load`
  - `T.must` ではなく `not_nil!`

#### 3. 実装

- 規約に従ってコーディング
- 新規ファイルにはマジックコメント追加

#### 4. 完了前の検証

**重要**: 完了報告前に全ての作業が適切に検証されていることを確認すること

- **テスト作成**: テスト作成後は、必ず `bin/rspec` を実行してテストが通ることを確認する
- **コード実装**: コード記述後は、必ず以下を確認する:
  - 型チェックが成功すること（`bin/srb tc` or `pnpm tsc`）
  - Linterの実行が成功すること（`bin/standardrb` or `bin/erb_lint --lint-all` or `pnpm prettier . --write && pnpm eslint . --fix`）
  - 関連するテストが通ること（`bin/rspec path/to/xxx_spec.rb`）
  - 明らかなランタイムエラーがないこと
- **ドキュメント編集**: Markdownファイルを編集した後は、必ず以下を行う:
  - Prettierの実行 (`pnpm prettier . --write`)
- **リトライポリシー**: 問題発生時は自動で最大5回まで再試行し、それでも解消できない場合にのみユーザーへ連絡する（途中経過は報告しない）
  - Report to user: "同じエラーが5回続いています。別のアプローチが必要かもしれません。"
- **以下の状態では絶対に完了報告をしない**:
  - テストが失敗している（未実装機能のテストを意図的に作成している場合を除く）
  - コンパイルエラーがある
  - 前回の試行から未解決のエラーが残っている

### 検証コマンド

```bash
# Rubyのファイルを編集したとき実行する
bin/standardrb
bin/erb_lint --lint-all
bin/srb tc
bin/rails sorbet:update
bin/rails zeitwerk:check
bin/rspec

# JavaScript/TypeScriptを編集したとき実行する
pnpm prettier . --write
pnpm eslint . --fix
pnpm tsc

# 全ての検証を実行
bin/check
```

## デバッグ・トラブルシューティング

- Sorbetエラー: `bin/rails sorbet:update` で型定義更新
- オートローディングエラー: `bin/rails zeitwerk:check`
- フォーマットエラー: `pnpm prettier . --write`
- Lintエラー: 各種Lintコマンドで修正

## 重要な原則

- ネストしたトランザクションを避ける
- レコードのコールバックを避ける
- View/Componentでのデータベースアクセスを防ぐ
- 問題が解決されるなら、レイヤーを跨いだ依存も許可
- 説明的な命名規則
- コメントは日本語で記載
- 1行100文字以内
- セキュリティベストプラクティスに従う
