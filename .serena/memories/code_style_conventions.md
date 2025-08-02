# コーディング規約

## Ruby
- 文字列はダブルクオートを使用
- 最終行に改行を入れる（RuboCop `Layout/TrailingEmptyLines`）
- 後置ifは使用しない
- ハッシュのキーと変数名が同じ場合は省略記法を使用（`user: user` → `user:`）
- 新規ファイルには以下のマジックコメントを記載:
  ```ruby
  # typed: strict
  # frozen_string_literal: true
  ```
- プライベートメソッドは `private def` で定義
- `T.must` の代わりに `#not_nil!` を使用

## Rails特有の規約

### ディレクトリ構造と責務
- `app/records`: ActiveRecord::Baseを継承（DBテーブルと1:1）
- `app/models`: POROや構造体（ドメインロジック）
- `app/repositories`: RecordとModelの変換
- `app/forms`: フォームオブジェクト
- `app/services`: ビジネスロジック
- `app/components`: ViewComponent
- `app/policies`: 認可ルール

### クラス間の依存関係ルール
各クラスは決められた依存先のみ参照可能（詳細は docs/claude/base/coding-conventions/rails.md 参照）

### マイグレーション
- IDには `generate_ulid()` を使用:
  ```ruby
  create_table :examples, id: false do |t|
    t.uuid :id, default: "generate_ulid()", null: false, primary_key: true
  end
  ```

### I18n
- 翻訳ファイルは用途別に分類:
  - `forms.(ja,en).yml`: フォーム関連
  - `messages.(ja,en).yml`: メッセージ・説明文
  - `meta.(ja,en).yml`: メタデータ
  - `nouns.(ja,en).yml`: 名詞・ラベル
- 日本語と英語の両方を更新

## CSS
- Tailwind CSSを使用

## JavaScript/TypeScript
- Hotwireパターンに従う
- Stimulus controllerの命名規則に従う

## RSpec
- `context`ブロックは使用しない
- `let`, `let!`は使わず、`it`内で変数定義
- `described_class`は使用せず、明示的にクラス名を記載
- Request specのパス: `spec/requests/<controller>/<action>_spec.rb`

## 一般
- 説明的な命名規則
- コメントは日本語で記載
- 1行90文字以内
- セキュリティベストプラクティスに従う