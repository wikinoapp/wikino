-- name: IsFeatureFlagEnabled :one
-- 指定ユーザーに対してフィーチャーフラグが有効かどうかを返す
SELECT EXISTS(
    SELECT 1 FROM feature_flags
    WHERE user_id = $1 AND name = $2
);

-- name: IsFeatureFlagEnabledBySessionToken :one
-- セッショントークンからユーザーを特定し、フィーチャーフラグが有効かどうかを返す
SELECT EXISTS(
    SELECT 1 FROM feature_flags ff
    INNER JOIN user_sessions us ON ff.user_id = us.user_id
    WHERE us.token = $1 AND ff.name = $2
);
