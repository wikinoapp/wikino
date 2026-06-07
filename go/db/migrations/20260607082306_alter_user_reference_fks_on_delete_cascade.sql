-- migrate:up

-- Recreate the user_id FKs on the Go-added tables with ON DELETE CASCADE so
-- that deleting users from the Rails side (which does not know about these
-- Go-added FKs) cannot fail with a foreign key violation. Both tables hold
-- pure dependent rows (reset tokens and per-user flags) that are meaningless
-- without the user. The FK columns are already indexed
-- (idx_password_reset_tokens_user_id / idx_feature_flags_user_id).
--
-- [Ja] Rails 側 (Go で追加されたこれらの FK を知らない) がユーザーを削除しても
-- 外部キー違反で失敗しないよう、Go 追加テーブルの user_id FK を
-- ON DELETE CASCADE 付きで作り直す。どちらのテーブルもユーザー無しでは
-- 意味を持たない純粋な従属データ (リセットトークンとユーザー単位のフラグ)。
-- FK カラムにはインデックスが既にある
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
