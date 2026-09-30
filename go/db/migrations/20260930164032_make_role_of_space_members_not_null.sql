-- migrate:up
-- 既存行はロールで埋め終わっており、Go版は常にロールを保存するため、ロールの無いスペースメンバーを作れなくする。
ALTER TABLE space_members
    ALTER COLUMN role SET NOT NULL;

-- migrate:down
ALTER TABLE space_members
    ALTER COLUMN role DROP NOT NULL;
