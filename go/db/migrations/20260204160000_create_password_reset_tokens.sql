-- migrate:up

-- password_reset_tokensテーブル: パスワードリセット用のトークンを保存する
-- トークンはSHA256でハッシュ化して保存し、1回のみ使用可能
CREATE TABLE password_reset_tokens (
    id uuid NOT NULL PRIMARY KEY DEFAULT generate_ulid(),
    user_id uuid NOT NULL REFERENCES users(id),
    token_digest VARCHAR NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- token_digestのユニークインデックス（トークン検索の高速化）
CREATE UNIQUE INDEX idx_password_reset_tokens_token_digest ON password_reset_tokens(token_digest);

-- user_idのインデックス（ユーザーのトークン検索・削除の高速化）
CREATE INDEX idx_password_reset_tokens_user_id ON password_reset_tokens(user_id);

-- migrate:down

DROP INDEX IF EXISTS idx_password_reset_tokens_user_id;
DROP INDEX IF EXISTS idx_password_reset_tokens_token_digest;
DROP TABLE IF EXISTS password_reset_tokens;
