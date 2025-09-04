# 型定義ファイルの配置修正仕様書

## 背景と問題

### 現在の問題

型定義ファイルが`config/initializers/`に配置されていたため、以下の問題が発生していた：

1. **開発環境でのコードリロード時のエラー**
   - コントローラーを編集後、ページをリロードすると`TypeError`が発生
   - エラー内容：`Expected type T.any(SpaceGuestPolicy, SpaceMemberPolicy, SpaceOwnerPolicy), got type SpaceOwnerPolicy`

2. **根本原因**
   - Railsの開発環境では、コード変更時にクラスが再読み込みされる
   - `config/initializers/`内の型定義は再読み込みされないが、参照先のPolicyクラスは再読み込みされる
   - この結果、型定義とクラス定義の不整合が発生する

### 再現手順

1. トピックページにアクセスする
2. `app/controllers/topics/show_controller.rb`を編集する
3. ページをリロードする
4. TypeErrorが発生

## 解決方針

型定義を`app/models/types.rb`に配置し、Railsのオートローディング機構（Zeitwerk）で管理する。これにより、コードリロード時も型定義とクラス定義の整合性が保たれる。

## 要件

### 機能要件

1. **型定義の適切な配置**
   - `Types`モジュール内の型定義を`app/models/types.rb`に配置
   - Zeitwerkによる自動読み込みに対応した構造にする

2. **既存コードとの互換性**
   - `T::Wikino::DatabaseId`、`T::Wikino::SpacePolicyInstance`、`T::Wikino::TopicPolicyInstance`を
   - `Types::DatabaseId`、`Types::SpacePolicyInstance`、`Types::TopicPolicyInstance`に移行

3. **開発環境での安定性**
   - コード変更時のリロードでエラーが発生しないこと
   - 型チェック（`bin/srb tc`）が正常に動作すること

### 非機能要件

1. **保守性**
   - 型定義の追加・変更が容易であること
   - ファイル構造が直感的で理解しやすいこと

2. **パフォーマンス**
   - 本番環境での起動時間に影響を与えないこと

## 実装結果

### 1. 実装作業

- [x] 型定義ファイルを`app/models/types.rb`に作成
- [x] 影響を受けるファイルの確認（`T::Wikino`を使用している箇所の洗い出し）
- [x] `T::Wikino::`を`Types::`に一括置換
- [x] 不要なファイルの削除

### 2. ファイル構造

```
app/
  models/
    types.rb  # Types モジュールと型定義
```

### 3. 実装内容

`app/models/types.rb`:

```ruby
# typed: strict
# frozen_string_literal: true

# Wikino全体で使用する型定義
module Types
  extend T::Sig

  # データベースIDの型エイリアス
  DatabaseId = T.type_alias { String }

  # Policy関連の型エイリアス
  # SpacePolicyFactoryが返す可能性のあるPolicyクラスの型
  SpacePolicyInstance = T.type_alias {
    T.any(::SpaceOwnerPolicy, ::SpaceMemberPolicy, ::SpaceGuestPolicy)
  }

  # Topic層のPolicyクラスの型
  TopicPolicyInstance = T.type_alias {
    T.any(::TopicOwnerPolicy, ::TopicAdminPolicy, ::TopicMemberPolicy, ::TopicGuestPolicy)
  }
end
```

### 4. 検証結果

- [x] 型チェック（`bin/srb tc`）の実行と成功確認 ✅
- [x] オートローディングチェック（`bin/rails zeitwerk:check`）の実行と成功確認 ✅
- [x] テストスイート（`bin/rspec spec/policies/space_policy_factory_spec.rb`）の実行と成功確認 ✅
- [x] Linterの実行（`bin/standardrb`） ✅

## 注意事項

### Zeitwerkの命名規則

- ファイル名とクラス/モジュール名の対応に注意
- `app/models/types.rb` → `Types`モジュール

### 型定義のスコープ

- Policyクラスを参照する際は`::`を使用してトップレベルから参照
- これにより、名前空間の曖昧さを回避

### テスト環境への影響

- CI/CDパイプラインでの動作確認も必要
- 本番環境へのデプロイ前に、staging環境での検証を推奨

## 変更サマリー

### 変更前

- 型定義：`config/initializers/001_types.rb`
- モジュール名：`T::Wikino`
- 使用例：`T::Wikino::DatabaseId`

### 変更後

- 型定義：`app/models/types.rb`
- モジュール名：`Types`
- 使用例：`Types::DatabaseId`

## 期待される結果

1. **開発体験の向上**
   - コード変更時のエラーが解消される
   - 型定義の管理が容易になる
   - シンプルな`Types`モジュール名で可読性向上

2. **コードの保守性向上**
   - 型定義がアプリケーションコードと同じ場所で管理される
   - Railsの標準的な構造に従うことで、理解しやすくなる
   - 1ファイルのシンプルな構成

3. **エラーの解消**
   - `TypeError`が発生しなくなる
   - 型チェックが安定して動作する
   - 開発環境でのコードリロードに対応
