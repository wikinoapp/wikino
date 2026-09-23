-- このSQLはsqlcの生成時だけ適用する。実DBの列削除は後続リリースで行う。
ALTER TABLE pages DROP COLUMN body_html;
ALTER TABLE page_revisions DROP COLUMN body_html;
ALTER TABLE draft_pages DROP COLUMN body_html;
ALTER TABLE draft_page_revisions DROP COLUMN body_html;
ALTER TABLE suggestion_pages DROP COLUMN body_html;
ALTER TABLE suggestion_page_revisions DROP COLUMN body_html;
