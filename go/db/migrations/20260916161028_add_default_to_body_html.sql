-- migrate:up

-- body_htmlへの書き込みを止める前にデフォルト値を付ける。デプロイ中に新旧の
-- コードが混在しても、body_htmlを渡さないINSERTがNOT NULL違反にならないようにする。
ALTER TABLE pages ALTER COLUMN body_html SET DEFAULT '';
ALTER TABLE page_revisions ALTER COLUMN body_html SET DEFAULT '';
ALTER TABLE draft_pages ALTER COLUMN body_html SET DEFAULT '';
ALTER TABLE draft_page_revisions ALTER COLUMN body_html SET DEFAULT '';

-- migrate:down

ALTER TABLE pages ALTER COLUMN body_html DROP DEFAULT;
ALTER TABLE page_revisions ALTER COLUMN body_html DROP DEFAULT;
ALTER TABLE draft_pages ALTER COLUMN body_html DROP DEFAULT;
ALTER TABLE draft_page_revisions ALTER COLUMN body_html DROP DEFAULT;
