-- name: IsFeatureFlagEnabled :one
-- 指定ユーザーに対してフィーチャーフラグが有効かどうかを返す
SELECT EXISTS(
    SELECT 1 FROM feature_flags
    WHERE user_id = $1 AND name = $2
);
