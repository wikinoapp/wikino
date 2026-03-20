-- migrate:up
ALTER TABLE suggestion_pages ADD COLUMN linked_page_ids VARCHAR[] NOT NULL DEFAULT '{}';
ALTER TABLE suggestion_pages ADD COLUMN featured_image_attachment_id UUID;

-- migrate:down
ALTER TABLE suggestion_pages DROP COLUMN featured_image_attachment_id;
ALTER TABLE suggestion_pages DROP COLUMN linked_page_ids;
