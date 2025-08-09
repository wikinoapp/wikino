# WikinoのRailsクラス設計ガイドライン

## 設計思想

- 関心の分離を厳格に実施
- レイヤー間の依存関係を最小化
- Sorbetを使用した型チェック
- 1つの責務に対して1つのクラス

## 各クラスの実装指針

### 1. Controller

- **責務**: HTTPリクエスト処理と応答の調整
- **実装方針**:
  - 1アクションにつき1コントローラー
  - レコードの取得
  - Repositoryを介したレコードからモデルへの変換
  - フォームバリデーションの処理
  - サービスの実行とエラーハンドリング
- **命名規則**: `(ModelPlural)::(ActionName)Controller`

### 2. View

- **責務**: 表示処理
- **実装方針**:
  - ViewComponentを使用
  - データベースへの直接アクセス禁止
- **命名規則**: `(ModelPlural)::(ActionName)View`

### 3. Component

- **責務**: 再利用可能なUI要素
- **実装方針**:
  - ViewComponentを使用
  - データベース依存なし
  - プライベートインスタンスメソッドを優先
- **命名規則**: `(UIComponentPlural)::(Noun)Component`

### 4. Form

- **責務**: フォームオブジェクト、ビジネスロジックのバリデーション
- **実装方針**:
  - 共通バリデーションモジュールの使用
  - ビジネスロジックに関するバリデーション
- **命名規則**: `(ModelPlural)::(Noun)Form`

### 5. Validator

- **責務**: カスタムバリデーション
- **実装方針**:
  - ActiveModelのバリデーターを拡張
  - カスタムバリデーションロジックの定義

### 6. Job

- **責務**: 非同期処理
- **実装方針**:
  - 最小限のロジック
  - 主にServiceの呼び出し
  - 複雑な処理は避ける

### 7. Service

- **責務**: ビジネスロジックのカプセル化、データ永続化
- **実装方針**:
  - データの永続化処理
  - データベーストランザクションの管理
  - 複雑なビジネスロジックは置かない
  - 例外をビジネスエラーに変換
- **命名規則**: `(ModelPlural)::(Verb)Service`

### 8. Mailer

- **責務**: メール送信
- **実装方針**:
  - 標準的なAction Mailerの使用

### 9. Policy

- **責務**: 認可ルール
- **実装方針**:
  - 認可の管理
  - Plain Old Ruby Objects (PORO)を使用

### 10. Model

- **責務**: データ構造とドメインロジックを表現
- **実装方針**:
  - 軽量なデータ構造
  - データベースアクセスなし
  - nullable参照の使用

### 11. Repository

- **責務**: ModelとRecordの変換
- **実装方針**:
  - RecordからModelへの変換
  - ModelとRecord間の直接依存を防ぐ
- **命名規則**: `(Model)Repository`

### 12. Record

- **責務**: DBのテーブルから取得・保存
- **実装方針**:
  - データベースとのやり取り
  - 1テーブルにつき1レコード

## 重要な原則

- ネストしたトランザクションを最小化
- レコードのコールバックを避ける
- View/Componentでのデータベースアクセスを防ぐ
- 問題が解決されるなら、レイヤーを跨いだ依存も許可

## 依存関係表

| クラス     | 依存先                                                    |
| ---------- | --------------------------------------------------------- |
| Component  | `Component`, `Form`, `Model`                              |
| Controller | `Form`, `Model`, `Record`,`Repository`, `Service`, `View` |
| Form       | `Record`, `Validator`                                     |
| Job        | `Service`                                                 |
| Mailer     | `Model`, `Record`, `Repository`, `View`                   |
| Model      | `Model`                                                   |
| Policy     | `Record`                                                  |
| Record     | `Record`                                                  |
| Repository | `Model`, `Record`                                         |
| Service    | `Job`, `Mailer`, `Record`                                 |
| Validator  | `Record`                                                  |
| View       | `Component`, `Form`, `Model`                              |
