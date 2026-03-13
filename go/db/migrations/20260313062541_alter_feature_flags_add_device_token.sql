-- migrate:up

-- device_tokenカラムを追加
ALTER TABLE feature_flags ADD COLUMN device_token VARCHAR;

-- user_idをnullableに変更
ALTER TABLE feature_flags ALTER COLUMN user_id DROP NOT NULL;

-- device_tokenとuser_idの少なくとも一方がNOT NULLであることを保証
ALTER TABLE feature_flags ADD CONSTRAINT chk_feature_flags_identifier
    CHECK (device_token IS NOT NULL OR user_id IS NOT NULL);

-- 既存のユニーク制約 (user_id, name) を削除して再作成
ALTER TABLE feature_flags DROP CONSTRAINT feature_flags_user_id_name_key;
ALTER TABLE feature_flags ADD CONSTRAINT feature_flags_user_id_name_key UNIQUE (user_id, name);

-- device_tokenとnameのユニーク制約を追加
ALTER TABLE feature_flags ADD CONSTRAINT feature_flags_device_token_name_key UNIQUE (device_token, name);

-- device_token用のインデックスを追加
CREATE INDEX idx_feature_flags_device_token ON feature_flags(device_token);

-- migrate:down

DROP INDEX IF EXISTS idx_feature_flags_device_token;
ALTER TABLE feature_flags DROP CONSTRAINT IF EXISTS feature_flags_device_token_name_key;
ALTER TABLE feature_flags DROP CONSTRAINT IF EXISTS feature_flags_user_id_name_key;
ALTER TABLE feature_flags ADD CONSTRAINT feature_flags_user_id_name_key UNIQUE (user_id, name);
ALTER TABLE feature_flags DROP CONSTRAINT IF EXISTS chk_feature_flags_identifier;
ALTER TABLE feature_flags ALTER COLUMN user_id SET NOT NULL;
ALTER TABLE feature_flags DROP COLUMN IF EXISTS device_token;
