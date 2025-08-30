# ページサムネイル画像とOGP画像の実装仕様

## 要件

ページ本文の最初に添付された画像ファイルを、以下の用途で利用できるようにする：

1. **ページ一覧のカード表示**
   - ページ一覧（トピック表示、スペース表示など）のカードコンポーネントにサムネイル画像を表示
   - Scrapboxのようなカード表示（テキストを左側に配置、画像を右側に配置）
   - 高解像度ディスプレイ対応のため、`:card`サイズ（400x400px）を使用
   - CSSで表示サイズは調整（実際の表示は100-150px程度）

2. **OGP画像（og:image）**
   - ページごとにog:imageメタタグを設定
   - 画像サイズは`:og`サイズ（最大1200x630px）を使用
   - ページに画像がない場合はデフォルトのOGP画像を使用

### 達成条件

- [ ] Markdown形式のページ本文1行目の画像を特定できる
- [ ] ページ一覧のカードに画像が表示される
- [ ] 各ページのog:imageメタタグが適切に設定される
- [ ] 画像がない場合の適切なフォールバック処理
- [ ] アニメーションGIFの特別処理（サムネイル生成せず、元画像を使用）
- [ ] pagesテーブルに最初の画像IDを保存してパフォーマンス向上
- [ ] 公開ページのOGP画像は認証なしでアクセス可能

## 修正方針

### 1. AttachmentRecordのバリアント定義更新

`AttachmentRecord#thumbnail_variant`メソッドを更新して、2種類のバリアントのみに絞る：

```ruby
variant_options = case size
when :card
  # ページ一覧のカード用（高解像度対応で400x400px）
  {resize_to_limit: [400, 400]}
when :og
  # OGP画像用（推奨: 1200x630）
  {resize_to_fit: [1200, 630]}
else
  {resize_to_limit: [400, 400]}
end
```

### 2. データベーススキーマの更新

`pages`テーブルにカラムを追加：

```ruby
# マイグレーション
add_column :pages, :first_image_attachment_id, :uuid, null: true
add_foreign_key :pages, :attachments, column: :first_image_attachment_id
add_index :pages, :first_image_attachment_id
```

### 3. PageRecordへのメソッド追加

`PageRecord`に以下のメソッドを追加：

```ruby
# belongs_to関連の追加
belongs_to :first_image_attachment_record,
  class_name: "AttachmentRecord",
  foreign_key: :first_image_attachment_id,
  optional: true

# Markdown本文の1行目から画像IDを抽出
def extract_first_line_image_id
  return nil if body.blank?
  
  # 1行目を取得
  first_line = body.lines.first&.strip
  return nil if first_line.blank?
  
  # Markdown画像形式をチェック: ![alt text](/attachments/attachment_id)
  match = first_line.match(/!\[[^\]]*\]\(\/attachments\/([^)]+)\)/)
  return nil unless match
  
  match[1]
end

# 最初の画像がGIFかどうか判定
def first_image_is_gif?
  attachment = first_image_attachment_record
  return false unless attachment
  
  filename = attachment.filename
  return false unless filename
  
  filename.downcase.end_with?('.gif')
end

# カード用画像URLを取得
def card_image_url(expires_in: 1.hour)
  attachment = first_image_attachment_record
  return nil unless attachment
  
  # GIFの場合はオリジナル画像のURLを返す
  if first_image_is_gif?
    attachment.generate_signed_url(space_member_record: nil, expires_in:)
  else
    attachment.thumbnail_url(size: :card, expires_in:)
  end
end

# OGP画像URLを取得
def og_image_url(expires_in: 1.hour)
  attachment = first_image_attachment_record
  return nil unless attachment
  
  # GIFの場合はnilを返す（デフォルトOGP画像を使用）
  return nil if first_image_is_gif?
  
  attachment.thumbnail_url(size: :og, expires_in:)
end
```

### 4. ページ作成・更新時の処理

`Pages::UpdateService`および`Pages::CreateService`で：

```ruby
def call
  # ... 既存の処理 ...
  
  # 1行目の画像IDを抽出
  first_image_id = page_record.extract_first_line_image_id
  
  if first_image_id
    # 同じスペースの添付ファイルか確認
    attachment = AttachmentRecord.find_by(
      id: first_image_id,
      space_id: page_record.space_id
    )
    
    if attachment
      # first_image_attachment_idを更新
      page_record.update!(first_image_attachment_id: attachment.id)
    else
      # 画像が見つからない場合はnullに設定
      page_record.update!(first_image_attachment_id: nil)
    end
  else
    # 1行目に画像がない場合はnullに設定
    page_record.update!(first_image_attachment_id: nil)
  end
end
```

### 5. ページ一覧カードの更新

`CardLinks::PageComponent`を更新：

1. PageRecordから`card_image_url`を取得
2. 画像がある場合はカード内に表示（テキストを左側、画像を右側に配置）
3. GIFの場合はアニメーションを維持したまま表示
4. 画像がない場合は既存のテキストのみ表示を維持
5. レスポンシブデザインに対応（モバイルでは画像サイズを調整）

### 6. OGP設定の更新

`Pages::ShowView`を更新：

1. PageRecordから`og_image_url`を取得
2. `set_meta_tags`のog:imageに設定
3. GIFの場合または画像がない場合はデフォルトのOGP画像を使用

### 7. 公開ページのOGP画像アクセス

`Attachments::ShowController`を更新：

1. 公開トピックのページに使用されている画像は認証なしでアクセス可能に
2. `all_referencing_pages_public?`メソッドを活用

## タスクリスト

### フェーズ1: データベーススキーマの更新

- [ ] マイグレーションファイルを作成（`first_image_attachment_id`カラム追加）
- [ ] マイグレーションを実行

### フェーズ2: バリアント定義の更新

- [ ] AttachmentRecordの`thumbnail_variant`メソッドを更新（`:card`と`:og`のみに）
- [ ] `generate_thumbnails`メソッドは削除または無効化（事前生成は行わない）
- [ ] テストを作成（spec/records/attachment_record_spec.rb）

### フェーズ3: データ取得ロジックの実装

- [ ] PageRecordに`first_image_attachment_record`関連を追加
- [ ] PageRecordに`extract_first_line_image_id`メソッドを追加
- [ ] PageRecordに`first_image_is_gif?`メソッドを追加
- [ ] PageRecordに`card_image_url`メソッドを追加
- [ ] PageRecordに`og_image_url`メソッドを追加
- [ ] テストを作成（spec/records/page_record_spec.rb）

### フェーズ4: ページ更新時の処理実装

- [ ] `Pages::CreateService`で1行目画像の検出と保存処理を追加
- [ ] `Pages::UpdateService`で1行目画像の検出と保存処理を追加
- [ ] テストを作成

### フェーズ5: UI表示の実装

- [ ] `CardLinks::PageComponent`にサムネイル表示を追加
- [ ] コンポーネントのHTMLテンプレートを更新
- [ ] CSSでレイアウト調整（画像とテキストのバランス）
- [ ] システムテストを作成

### フェーズ6: OGP設定の実装

- [ ] `Pages::ShowView`でog:imageメタタグを設定
- [ ] `ApplicationView`のdefault_meta_tagsメソッドを調整（必要に応じて）
- [ ] 公開ページの画像アクセス権限を調整
- [ ] テストを作成

### フェーズ7: パフォーマンス最適化

- [ ] N+1問題の確認と対策（first_image_attachment_recordのpreload）
- [ ] ページ一覧表示時のクエリ最適化
- [ ] キャッシュの検討（必要に応じて）

## 注意事項

### セキュリティ

- 非公開ページの画像は適切に保護する
- 公開ページのOGP画像のみ認証なしアクセスを許可
- URLの有効期限を適切に設定

### パフォーマンス

- Active Storageのvariant機能により初回表示時に自動生成
- 一度生成されたサムネイルはキャッシュされる
- pagesテーブルに画像IDを保存することでページ一覧表示を高速化
- Retinaディスプレイ対応（2x解像度を考慮）

### エラーハンドリング

- 画像処理エラー時の適切なフォールバック
- 画像がない場合のデフォルト表示
- GIFファイルの適切な処理（サムネイル生成をスキップ）
- サムネイル生成失敗時のリトライ処理

### 互換性

- 既存のページに影響を与えない
- 段階的な移行が可能な実装
- 画像のない既存ページも正常に動作

## テスト計画

### ユニットテスト

- PageRecordの新メソッドのテスト
- サムネイル生成ジョブのテスト
- 画像抽出ロジックのテスト

### 統合テスト

- ページ作成・更新時のサムネイル生成
- カードコンポーネントの画像表示
- OGPメタタグの設定

### システムテスト

- ページ一覧での画像表示
- アニメーションGIFの表示確認
- SNSシェア時のOGP画像表示（手動確認）
- 様々な画像フォーマットでの動作確認（JPEG、PNG、GIF等）

## 実装優先度

1. **高**: PageRecordへのメソッド追加（基盤となる機能）
2. **高**: ページ一覧カードの画像表示（ユーザー体験の向上）
3. **中**: OGP画像の設定（SNSシェア機能の向上）
4. **低**: パフォーマンス最適化（必要に応じて後から調整）
