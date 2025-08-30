# ページサムネイル画像とOGP画像の実装仕様

## 要件

ページ本文の最初に添付された画像ファイルを、以下の用途で利用できるようにする：

1. **ページ一覧のカード表示**
   - ページ一覧（トピック表示、スペース表示など）のカードコンポーネントにサムネイル画像を表示
   - Scrapboxのようなカード表示（画像を左側に配置、テキストを右側に配置）
   - 高解像度ディスプレイ対応のため、`:medium`サイズ（400x400px）を使用
   - CSSで表示サイズは調整（実際の表示は100-150px程度）

2. **OGP画像（og:image）**
   - ページごとにog:imageメタタグを設定
   - 画像サイズは`:og`サイズ（最大1200x630px）を使用
   - ページに画像がない場合はデフォルトのOGP画像を使用

### 達成条件

- [ ] ページ本文の最初の画像を特定できる
- [ ] ページ一覧のカードに画像が表示される
- [ ] 各ページのog:imageメタタグが適切に設定される
- [ ] 画像がない場合の適切なフォールバック処理
- [ ] パフォーマンスを考慮したサムネイル事前生成
- [ ] 公開ページのOGP画像は認証なしでアクセス可能

## 修正方針

### 1. PageRecordへのメソッド追加

`PageRecord`に以下のメソッドを追加：

```ruby
# ページ本文の最初の添付画像を取得
def first_image_attachment
  # body_htmlから最初のimg要素のattachment-idを抽出
  # page_attachment_reference_recordsと結合して取得
end

# カード用サムネイルURLを取得
def card_thumbnail_url(expires_in: 1.hour)
  first_image_attachment&.thumbnail_url(size: :medium, expires_in:)
end

# OGP画像URLを取得
def og_image_url(expires_in: 1.hour)
  first_image_attachment&.thumbnail_url(size: :og, expires_in:)
end
```

### 2. ページ作成・更新時の処理

`Pages::UpdateService`および`Pages::CreateService`で：

1. ページ保存後に最初の画像を検出
2. 画像が見つかった場合、サムネイル事前生成ジョブをキューに追加
3. `:medium`（カード表示用）と`:og`（OGP用）サイズのサムネイルを生成

### 3. ページ一覧カードの更新

`CardLinks::PageComponent`を更新：

1. PageRecordから`card_thumbnail_url`を取得
2. 画像がある場合はカード内に表示（左側に画像、右側にテキスト）
3. 画像がない場合は既存のテキストのみ表示を維持
4. レスポンシブデザインに対応（モバイルでは画像サイズを調整）

### 4. OGP設定の更新

`Pages::ShowView`を更新：

1. PageRecordから`og_image_url`を取得
2. `set_meta_tags`のog:imageに設定
3. 画像がない場合はデフォルトのOGP画像を使用

### 5. 公開ページのOGP画像アクセス

`Attachments::ShowController`を更新：

1. 公開トピックのページに使用されている画像は認証なしでアクセス可能に
2. `all_referencing_pages_public?`メソッドを活用

## タスクリスト

### フェーズ1: データ取得ロジックの実装

- [ ] PageRecordに`first_image_attachment`メソッドを追加
- [ ] PageRecordに`card_thumbnail_url`メソッドを追加
- [ ] PageRecordに`og_image_url`メソッドを追加
- [ ] テストを作成（spec/records/page_record_spec.rb）

### フェーズ2: サムネイル生成処理の実装

- [ ] `Attachments::GenerateThumbnailsJob`を作成（既存のものがあれば活用）
- [ ] `Pages::CreateService`でサムネイル生成ジョブを追加
- [ ] `Pages::UpdateService`でサムネイル生成ジョブを追加
- [ ] テストを作成

### フェーズ3: UI表示の実装

- [ ] `CardLinks::PageComponent`にサムネイル表示を追加
- [ ] コンポーネントのHTMLテンプレートを更新
- [ ] CSSでレイアウト調整（画像とテキストのバランス）
- [ ] システムテストを作成

### フェーズ4: OGP設定の実装

- [ ] `Pages::ShowView`でog:imageメタタグを設定
- [ ] `ApplicationView`のdefault_meta_tagsメソッドを調整（必要に応じて）
- [ ] 公開ページの画像アクセス権限を調整
- [ ] テストを作成

### フェーズ5: パフォーマンス最適化

- [ ] サムネイル生成の非同期処理を確認
- [ ] N+1問題の確認と対策
- [ ] キャッシュの検討（必要に応じて）

## 注意事項

### セキュリティ

- 非公開ページの画像は適切に保護する
- 公開ページのOGP画像のみ認証なしアクセスを許可
- URLの有効期限を適切に設定

### パフォーマンス

- サムネイル生成は非同期で実行
- 頻繁に使用されるサイズ（:medium）は事前生成
- OGP用（:og）は必要時に生成
- Retinaディスプレイ対応（2x解像度を考慮）

### エラーハンドリング

- 画像処理エラー時の適切なフォールバック
- 画像がない場合のデフォルト表示
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
- SNSシェア時のOGP画像表示（手動確認）
- 様々な画像フォーマットでの動作確認

## 実装優先度

1. **高**: PageRecordへのメソッド追加（基盤となる機能）
2. **高**: ページ一覧カードの画像表示（ユーザー体験の向上）
3. **中**: OGP画像の設定（SNSシェア機能の向上）
4. **低**: パフォーマンス最適化（必要に応じて後から調整）