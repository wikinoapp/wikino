-- migrate:up

-- ページ本文のHTMLは表示時にレンダリングするため、保存していたbody_htmlを削除する。
-- Go版の列参照は先行リリースで停止済み。Rails版は列削除後に再起動し、
-- プリペアドステートメントとスキーマキャッシュを更新する。
-- ロック待ちが長引いてテーブルへのアクセスを止め続けないよう、取得できなければ失敗させる。
SET LOCAL lock_timeout = '5s';

ALTER TABLE pages DROP COLUMN body_html;
ALTER TABLE page_revisions DROP COLUMN body_html;
ALTER TABLE draft_pages DROP COLUMN body_html;
ALTER TABLE draft_page_revisions DROP COLUMN body_html;
ALTER TABLE suggestion_pages DROP COLUMN body_html;
ALTER TABLE suggestion_page_revisions DROP COLUMN body_html;

-- migrate:down

SET LOCAL lock_timeout = '5s';

ALTER TABLE pages ADD COLUMN body_html TEXT NOT NULL DEFAULT '';
ALTER TABLE page_revisions ADD COLUMN body_html TEXT NOT NULL DEFAULT '';
ALTER TABLE draft_pages ADD COLUMN body_html TEXT NOT NULL DEFAULT '';
ALTER TABLE draft_page_revisions ADD COLUMN body_html VARCHAR NOT NULL DEFAULT '';
ALTER TABLE suggestion_pages ADD COLUMN body_html VARCHAR NOT NULL DEFAULT '';
ALTER TABLE suggestion_page_revisions ADD COLUMN body_html VARCHAR NOT NULL DEFAULT '';
