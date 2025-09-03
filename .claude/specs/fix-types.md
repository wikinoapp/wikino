# 型定義ファイルの配置修正仕様書

## 背景と問題

### 現在の問題

型定義ファイルが`config/initializers/001_types.rb`に配置されているため、以下の問題が発生している：

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

型定義を`app/types/`配下に移動し、Railsのオートローディング機構（Zeitwerk）で管理する。これにより、コードリロード時も型定義とクラス定義の整合性が保たれる。

## 要件

### 機能要件

1. **型定義の適切な配置**
   - `T::Wikino`モジュール内の型定義を`app/types/`配下に配置
   - Zeitwerkによる自動読み込みに対応した構造にする

2. **既存コードとの互換性**
   - 現在`T::Wikino::DatabaseId`、`T::Wikino::SpacePolicyInstance`、`T::Wikino::TopicPolicyInstance`を使用している全てのコードが引き続き動作すること

3. **開発環境での安定性**
   - コード変更時のリロードでエラーが発生しないこと
   - 型チェック（`bin/srb tc`）が正常に動作すること

### 非機能要件

1. **保守性**
   - 型定義の追加・変更が容易であること
   - ファイル構造が直感的で理解しやすいこと

2. **パフォーマンス**
   - 本番環境での起動時間に影響を与えないこと

## タスクリスト

### 1. 準備作業

- [ ] 現在の型定義ファイル（`config/initializers/001_types.rb`）のバックアップ
- [ ] 影響を受けるファイルの確認（`T::Wikino`を使用している箇所の洗い出し）

### 2. 実装作業

- [ ] `app/types/`ディレクトリの作成
- [ ] `app/types/t/`ディレクトリの作成（名前空間に対応）
- [ ] `app/types/t/wikino.rb`ファイルの作成と型定義の移動
- [ ] `config/initializers/001_types.rb`の削除

### 3. ファイル構造

```
app/
  types/
    t/
      wikino.rb  # T::Wikino モジュールと型定義
```

### 4. 実装内容

`app/types/t/wikino.rb`:
```ruby
# typed: strict
# frozen_string_literal: true

module T
  module Wikino
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
end
```

### 5. 検証作業

- [ ] 型チェック（`bin/srb tc`）の実行と成功確認
- [ ] オートローディングチェック（`bin/rails zeitwerk:check`）の実行と成功確認
- [ ] テストスイート（`bin/rspec`）の実行と成功確認
- [ ] 開発環境での動作確認
  - [ ] サーバー起動確認
  - [ ] ページアクセス確認
  - [ ] コード変更後のリロード確認（エラーが発生しないこと）
- [ ] Linterの実行（`bin/standardrb`）

### 6. クリーンアップ

- [ ] 不要になった`config/initializers/001_types.rb`の削除確認
- [ ] コミットメッセージの作成

## 注意事項

### Zeitwerkの命名規則

- ファイル名とクラス/モジュール名の対応に注意
- `app/types/t/wikino.rb` → `T::Wikino`モジュール

### 型定義のスコープ

- Policyクラスを参照する際は`::`を使用してトップレベルから参照
- これにより、名前空間の曖昧さを回避

### テスト環境への影響

- CI/CDパイプラインでの動作確認も必要
- 本番環境へのデプロイ前に、staging環境での検証を推奨

## 期待される結果

1. **開発体験の向上**
   - コード変更時のエラーが解消される
   - 型定義の管理が容易になる

2. **コードの保守性向上**
   - 型定義がアプリケーションコードと同じ場所で管理される
   - Railsの標準的な構造に従うことで、理解しやすくなる

3. **エラーの解消**
   - `TypeError`が発生しなくなる
   - 型チェックが安定して動作する