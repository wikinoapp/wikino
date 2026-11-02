-- migrate:up
SET LOCAL lock_timeout = '5s';

-- 下書きを書き始めた時点のタイトルと本文 (起点)。最古のリビジョンの差分の比較対象にする。
-- base_bodyがNULLの行は起点が分からない下書きで、base_titleがNULLでbase_bodyがNULLでない行は
-- タイトルの無いページから書き始めた下書きである。
-- 既存の行は書き始めた時点の内容が分からないため、NULLのままにする。
ALTER TABLE draft_pages
    ADD COLUMN base_title citext,
    ADD COLUMN base_body citext;

-- migrate:down
SET LOCAL lock_timeout = '5s';

ALTER TABLE draft_pages
    DROP COLUMN base_body,
    DROP COLUMN base_title;
