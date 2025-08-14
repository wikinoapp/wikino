# プロジェクト構造

## 主要ディレクトリ

### app/

Railsアプリケーションの主要なコード:

- **assets/**: 画像やCSSファイル
- **components/**: ViewComponentを使用した再利用可能なUIコンポーネント
- **controllers/**: HTTPリクエスト処理（Railsコントローラー）
- **forms/**: フォームオブジェクト（バリデーションとデータ変換）
- **javascript/**: Hotwireで実装されたフロントエンドコード
- **jobs/**: Active Job（非同期処理）
- **mailers/**: Action Mailer（メール送信）
- **models/**: PORO（Plain Old Ruby Object）や構造体、ドメインロジック
- **policies/**: 認可ルール（権限管理）
- **records/**: ActiveRecord::Baseを継承したDBモデル
- **repositories/**: RecordとModel間の変換ロジック
- **services/**: ビジネスロジックのカプセル化
- **validators/**: カスタムバリデーション
- **views/**: ViewComponentを使用したビュー

### その他の重要ディレクトリ

- **bin/**: 実行可能スクリプト（rails, rspec, check等）
- **config/**: 設定ファイル
  - **locales/**: I18n翻訳ファイル
- **db/**: データベース関連（マイグレーション、スキーマ）
- **docs/**: ドキュメント
  - **claude/**: Claude AI用のドキュメント
- **spec/**: RSpecテスト
  - **requests/**: Request spec（コントローラー/アクション別）
  - **factories/**: FactoryBot定義
- **sorbet/**: Sorbet型定義

## アーキテクチャの特徴

1. **レイヤー分離**: Record（DB層）とModel（ドメイン層）を明確に分離
2. **Repository パターン**: データアクセスロジックの抽象化
3. **Form オブジェクト**: 複雑なフォーム処理の分離
4. **Service オブジェクト**: ビジネスロジックの整理
5. **ViewComponent**: コンポーネントベースのビュー構築
6. **型安全性**: Sorbetによる静的型付け
