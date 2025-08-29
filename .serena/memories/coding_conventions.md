# Wikino コーディング規約

## Ruby

- マジックコメント: `# typed: strict` と `# frozen_string_literal: true`
- 文字列はダブルクオート使用
- ハッシュの省略記法を使用: `{user:, name:}`
- プライベートメソッドは `private def` で定義
- プロテクテッドメソッドは `protected def` で定義
- 後置ifは使用しない
- attr_readerは個別にprivate指定
- T.mustではなくnot_nil!を使用
- データベースIDの型は `T::Wikino::DatabaseId` を使用

## ActiveRecord

- includesは使用禁止
- 明示的にpreloadまたはeager_loadを使用
- マイグレーションではULIDを使用

## RSpec

- context, let, described_classは使用しない
- itブロック内で変数定義
- FactoryBotで作成したレコードの変数名には`_record`サフィックスを付ける

## JavaScript/TypeScript

- HTTPリクエストには@rails/request.jsを使用（fetchは使用しない）

## サービスクラス

- データベースへの永続化を伴う処理のみ実装
- トランザクションには必ず`with_transaction`メソッドを使用
- Controller、Job、Rakeタスク内で永続化処理を書く場合は必ずServiceクラスを定義

## 一般原則

- コメントは日本語で記載
- 1行100文字以内
- 説明的な命名規則を使用
