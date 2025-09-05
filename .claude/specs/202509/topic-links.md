# トピックリンク表示機能仕様書

## 概要

スペースページ (`GET /s/:space_identifier`) に、そのスペースに参加しているトピックのリンクをカード形式で表示する機能を実装します。

## 要件

### 表示場所

- スペースページ (`GET /s/:space_identifier`)

### リンク先

- トピックページ (`GET /s/:space_identifier/topics/:topic_number`)

### カード表示内容

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

### 表示順

- `topic_members.last_page_modified_at` の降順

## 実装タスクリスト

### 1. データ取得層の実装

- [x] ユーザーが参加しているトピックを取得するRepositoryメソッドの実装
  - `TopicRepository#find_topics_by_space` メソッドを実装
  - `space_member_record` を引数として受け取る設計
  - ユーザーが参加している（`topic_members`テーブルにレコードがある）トピックのみを取得
- [x] `last_page_modified_at` でソートする処理の実装
  - `topic_members.last_page_modified_at` の降順でソート
  - NULL値は最後に配置

### 2. UIコンポーネントの実装

- [x] トピックカードコンポーネント (`CardLinks::TopicComponent`) の作成
  - [x] トピック名の表示
  - [x] トピックページへのリンク
  - [x] ページ作成リンク（権限がある場合のみ）
  - [x] 設定ページへのリンク（権限がある場合のみ）
- [x] トピックカードリストコンポーネント (`CardLists::TopicComponent`) の作成
  - [x] カードの一覧表示
  - [x] グリッドレイアウトの実装

### 3. 権限チェックの実装

- [ ] トピックへのページ作成権限チェックメソッドの実装
- [ ] Policyクラスでの権限判定ロジック

### 4. スペースページへの統合

- [ ] `Spaces::ShowView` にトピックカードリストを追加
- [ ] スペースコントローラーでトピックデータの取得処理を追加

### 5. スタイリング

- [ ] カードのデザイン実装（Tailwind CSS）
- [ ] レスポンシブデザインの対応
- [ ] ホバー効果やトランジションの追加

### 6. テスト

- [x] Repositoryメソッドのテスト
  - ユーザーが参加しているトピックのみを取得することを確認
  - `space_member_record`がnilの場合は空配列を返すことを確認
  - `last_page_modified_at`でソートされることを確認
  - 権限フラグが正しく設定されることを確認
- [ ] コンポーネントのテスト
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
- `app/components/topics/card_component.rb`
- `app/components/topics/card_list_component.rb`
- `app/views/spaces/show_view.rb`
- `app/controllers/spaces/show_controller.rb`
- `app/policies/topic_policy.rb`
- `spec/components/topics/card_component_spec.rb`
- `spec/components/topics/card_list_component_spec.rb`
- `spec/system/spaces/topics_display_spec.rb`
