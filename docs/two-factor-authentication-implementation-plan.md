# 二要素認証（2FA）実装方針

## 概要

Wikinoアプリケーションに二要素認証（2FA）機能を追加し、ユーザーアカウントのセキュリティを強化します。この実装では、TOTP（Time-based One-Time Password）方式を採用します。

## 現在の認証システム分析

### 主要コンポーネント

1. **認証フロー**

   - `UserSessionsController` - ログイン処理
   - `UserSessionForm::Creation` - ログインフォームバリデーション
   - `UserSessionService::Create` - セッション作成ロジック
   - `ControllerConcerns::Authenticatable` - 認証関連の共通処理

2. **データモデル**

   - `users` テーブル - ユーザー基本情報
   - `user_passwords` テーブル - パスワードハッシュ管理
   - `user_sessions` テーブル - セッション管理

3. **セッション管理**
   - Cookieベースのトークン認証
   - 永続的なセッション保存

## 実装方針

### 1. データベーススキーマ変更

#### 新規テーブル: `user_two_factor_auths`

```sql
CREATE TABLE user_two_factor_auths (
    id uuid DEFAULT generate_ulid() NOT NULL PRIMARY KEY,
    user_id uuid NOT NULL,
    secret character varying NOT NULL,
    enabled boolean DEFAULT false NOT NULL,
    enabled_at timestamp without time zone,
    recovery_codes text[] NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL,
    CONSTRAINT fk_user_two_factor_auths_user_id FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE UNIQUE INDEX index_user_two_factor_auths_on_user_id ON user_two_factor_auths(user_id);
```

### 2. 新規モデル・レコードクラス

#### `UserTwoFactorAuthRecord` (app/records/)

- Activeレコードクラス
- TOTPシークレットとリカバリーコードの管理

#### `UserTwoFactorAuth` (app/models/)

- ビジネスロジックを含むモデル
- TOTPコードの検証メソッド

### 3. 実装する機能

#### 3.1 2FA設定機能

**新規コントローラー:**

- `Settings::TwoFactorAuths::NewController` - 2FA設定開始
- `Settings::TwoFactorAuths::CreateController` - 2FA有効化
- `Settings::TwoFactorAuths::DestroyController` - 2FA無効化

**新規フォーム:**

- `TwoFactorAuthForm::Creation` - QRコード表示と確認コード入力
- `TwoFactorAuthForm::Destruction` - 2FA無効化時のパスワード確認

**新規サービス:**

- `TwoFactorAuthService::Setup` - シークレット生成とQRコード作成
- `TwoFactorAuthService::Enable` - 2FA有効化とリカバリーコード生成
- `TwoFactorAuthService::Disable` - 2FA無効化

#### 3.2 ログイン時の2FA検証

**既存コントローラーの拡張:**

- `UserSessions::CreateController` - 2FAが有効な場合の処理追加

**新規コントローラー:**

- `UserSessions::TwoFactorAuths::NewController` - 2FAコード入力画面
- `UserSessions::TwoFactorAuths::CreateController` - 2FAコード検証

**新規フォーム:**

- `UserSessionForm::TwoFactorVerification` - 2FAコード検証

#### 3.3 リカバリーコード管理

**新規コントローラー:**

- `Settings::TwoFactorAuths::RecoveryCodes::ShowController` - リカバリーコード表示
- `Settings::TwoFactorAuths::RecoveryCodes::CreateController` - リカバリーコード再生成

### 4. UI/UX設計

#### 4.1 設定画面

- アカウント設定内に「二要素認証」セクション追加
- QRコード表示とセットアップ手順の説明
- リカバリーコードの表示とダウンロード機能

#### 4.2 ログインフロー

1. メールアドレスとパスワード入力
2. 2FAが有効な場合、認証コード入力画面へ遷移
3. 6桁のTOTPコードまたはリカバリーコード入力
4. 検証成功後、通常のセッション作成

### 5. セキュリティ考慮事項

1. **シークレット管理**

   - TOTPシークレットは暗号化して保存
   - Rails.application.credentialsを使用

2. **レート制限**

   - 2FA認証試行回数の制限（5回失敗で一時的にロック）
   - IPアドレスベースの制限

3. **セッション管理**

   - 2FA検証前の一時セッション管理
   - 検証成功後に本セッションへ昇格

4. **監査ログ**
   - 2FA有効化/無効化の記録
   - 失敗した認証試行の記録

### 6. 実装順序

1. データベーススキーマとモデル作成
2. 2FA設定機能の実装
3. ログイン時の2FA検証機能
4. リカバリーコード管理機能
5. UI/UXの実装とテスト
6. セキュリティ機能の追加

### 7. 必要な依存関係

- `rotp` gem - TOTP実装
- `rqrcode` gem - QRコード生成

### 8. テスト戦略

- 単体テスト: モデル、フォーム、サービスクラス
- 統合テスト: 2FA設定フロー、ログインフロー
- システムテスト: E2Eでの2FA動作確認

## 今後の拡張可能性

- WebAuthn（FIDO2）対応
- SMS認証の追加
- バックアップコードの印刷機能
- 信頼できるデバイスの記憶機能
