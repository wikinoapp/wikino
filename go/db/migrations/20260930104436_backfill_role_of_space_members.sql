-- migrate:up
-- ロールを保存しない旧版の停止後に、ロールの無いスペースメンバーを埋める。
-- ロールの無いメンバーは全員space:adminを持つため、管理者のロールにする。
-- 保存済みのロールは上書きしない。
UPDATE space_members SET role = 'admin' WHERE role IS NULL;

-- migrate:down
-- 埋めた行と元からロールを保存していた行は区別できないため、戻さない。
