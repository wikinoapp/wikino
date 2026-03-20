-- migrate:up
ALTER TABLE draft_pages ADD COLUMN featured_image_attachment_id UUID;

-- migrate:down
ALTER TABLE draft_pages DROP COLUMN featured_image_attachment_id;
