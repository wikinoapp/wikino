# トピックリンク表示機能仕様書

## 概要

スペースページ (`GET /s/:space_identifier`) に、トピックのリンクをカード形式で表示する機能を実装します。

- **スペースメンバー**: ユーザーが参加しているトピックのみ表示
- **ゲスト**: スペース内の公開トピックを表示

## 要件

### 表示場所

- スペースページ (`GET /s/:space_identifier`)

### リンク先

- トピックページ (`GET /s/:space_identifier/topics/:topic_number`)

### カード表示内容

#### スペースメンバー向け表示

1. トピック名
2. フッター
   1. トピックにページを作成するリンク
      - リンク先: `GET /s/:space_identifier/topics/:topic_number/pages/new`
      - アイコン: `pencil-simple-line`
      - 権限チェック: トピックにページを作成する権限がない場合はリンクを非アクティブにする
   2. 設定ページへのリンク
      - リンク先: `/s/:space_identifier/topics/:topic_number/settings`
      - アイコン: `gear-regular`
      - 権限チェック: トピックの設定を変更する権限がない場合はリンクを非アクティブにする

#### ゲスト向け表示

1. トピック名
2. トピックの説明（description）
3. トピックページへのリンク

### 表示順

#### スペースメンバー向け

- `topic_members.last_page_modified_at` の降順
- トピック番号（`topics.number`）の降順

#### ゲスト向け

- トピックの作成日時（`topics.created_at`）の降順
- トピック番号（`topics.number`）の降順

## 実装タスクリスト

### 1. データ取得層の実装

- [x] スペースメンバー向け：ユーザーが参加しているトピックを取得するRepositoryメソッドの実装
  - `TopicRepository#find_topics_by_space` メソッドを実装
  - `space_member_record` を引数として受け取る設計
  - ユーザーが参加している（`topic_members`テーブルにレコードがある）トピックのみを取得
- [x] `last_page_modified_at` でソートする処理の実装
  - `topic_members.last_page_modified_at` の降順でソート
  - NULL値は最後に配置
- [x] ゲスト向け：公開トピックを取得するRepositoryメソッドの実装
  - `TopicRepository#find_public_topics_by_space` メソッドを実装
  - 公開トピック（visibility: public）のみを取得
  - トピックの作成日時でソート

### 2. UIコンポーネントの実装

- [x] スペースメンバー向けトピックカードコンポーネント (`CardLinks::TopicComponent`) の作成
  - [x] トピック名の表示
  - [x] トピックページへのリンク
  - [x] ページ作成リンク（権限がある場合のみ）
  - [x] 設定ページへのリンク（権限がある場合のみ）
- [x] ゲスト向けトピックカードコンポーネント (`CardLinks::PublicTopicComponent`) の作成
  - [x] トピック名の表示
  - [x] トピックの説明（description）の表示
  - [x] トピックページへのリンク
- [x] トピックカードリストコンポーネント (`CardLists::TopicComponent`) の作成
  - [x] カードの一覧表示
  - [x] グリッドレイアウトの実装
  - [x] ゲスト/メンバーに応じたカードコンポーネントの使い分け

### 3. 権限チェックの実装

- [ ] トピックへのページ作成権限チェックメソッドの実装
- [ ] Policyクラスでの権限判定ロジック

### 4. スペースページへの統合

- [ ] `Spaces::ShowView` にトピックカードリストを追加
- [ ] スペースコントローラーでトピックデータの取得処理を追加
  - [ ] スペースメンバーの場合は参加トピックを取得
  - [ ] ゲストの場合は公開トピックを取得

### 5. スタイリング

- [ ] カードのデザイン実装（Tailwind CSS）
- [ ] レスポンシブデザインの対応
- [ ] ホバー効果やトランジションの追加

### 6. テスト

- [x] Repositoryメソッドのテスト（スペースメンバー向け）
  - ユーザーが参加しているトピックのみを取得することを確認
  - `space_member_record`がnilの場合は空配列を返すことを確認
  - `last_page_modified_at`でソートされることを確認
  - 権限フラグが正しく設定されることを確認
- [ ] Repositoryメソッドのテスト（ゲスト向け）
  - 公開トピックのみを取得することを確認
  - 作成日時でソートされることを確認
- [ ] コンポーネントのテスト
  - [ ] スペースメンバー向けカードコンポーネント
  - [ ] ゲスト向けカードコンポーネント
- [ ] 権限チェックのテスト
- [ ] Request spec

## 技術的考慮事項

### パフォーマンス

- N+1問題を避けるため、必要な関連データを `preload` または `eager_load` で取得
- トピック数が多い場合のページネーション検討
  - 参加するトピックはそこまで多くならないと思うので一旦検討しない

### アクセシビリティ

- 適切なARIA属性の設定
- キーボードナビゲーション対応
- スクリーンリーダー対応

### セキュリティ

- 権限チェックの確実な実施
- XSS対策（Railsのデフォルト機能を活用）

## 関連ファイル（想定）

- `app/repositories/topic_repository.rb`
- `app/components/card_links/topic_component.rb` （スペースメンバー向け）
- `app/components/card_links/public_topic_component.rb` （ゲスト向け）
- `app/components/card_lists/topic_component.rb`
- `app/views/spaces/show_view.rb`
- `app/controllers/spaces/show_controller.rb`
- `app/policies/topic_policy.rb`
- `spec/repositories/topic_repository_spec.rb`
- `spec/components/card_links/topic_component_spec.rb`
- `spec/components/card_links/public_topic_component_spec.rb`
- `spec/components/card_lists/topic_component_spec.rb`
- `spec/system/spaces/topics_display_spec.rb`
