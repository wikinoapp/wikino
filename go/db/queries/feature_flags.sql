-- name: IsFeatureFlagEnabled :one
-- 指定ユーザーに対してフィーチャーフラグが有効かどうかを返す
SELECT EXISTS(
    SELECT 1 FROM feature_flags
    WHERE user_id = $1 AND name = $2
);

-- name: IsFeatureFlagEnabledForDevice :one
-- デバイストークンまたはセッショントークン経由のuser_idでフラグが有効かを判定する
SELECT EXISTS(
    SELECT 1 FROM feature_flags ff
    WHERE ff.name = $3
    AND (
        (ff.device_token IS NOT NULL AND ff.device_token = $1)
        OR (ff.user_id IS NOT NULL AND ff.user_id = (
            SELECT us.user_id FROM user_sessions us WHERE us.token = $2
        ))
    )
);
