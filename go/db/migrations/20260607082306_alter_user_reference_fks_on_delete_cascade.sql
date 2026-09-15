-- migrate:up

-- Rails側 (Goで追加されたこれらのFKを知らない) がユーザーを削除しても
-- 外部キー違反で失敗しないよう、Go追加テーブルのuser_id FKを
-- ON DELETE CASCADE付きで作り直す。どちらのテーブルもユーザー無しでは
-- 意味を持たない純粋な従属データ (リセットトークンとユーザー単位のフラグ)。
-- FKカラムにはインデックスが既にある
-- (idx_password_reset_tokens_user_id / idx_feature_flags_user_id)。
ALTER TABLE password_reset_tokens
    DROP CONSTRAINT password_reset_tokens_user_id_fkey,
    ADD CONSTRAINT password_reset_tokens_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE feature_flags
    DROP CONSTRAINT feature_flags_user_id_fkey,
    ADD CONSTRAINT feature_flags_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- migrate:down

ALTER TABLE feature_flags
    DROP CONSTRAINT feature_flags_user_id_fkey,
    ADD CONSTRAINT feature_flags_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users(id);

ALTER TABLE password_reset_tokens
    DROP CONSTRAINT password_reset_tokens_user_id_fkey,
    ADD CONSTRAINT password_reset_tokens_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users(id);
