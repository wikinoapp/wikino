-- migrate:up transaction:false
-- CONCURRENTLYはトランザクション外で単独実行し、通常の書き込みを止めない。
CREATE INDEX CONCURRENTLY index_email_confirmations_on_user_id ON email_confirmations (user_id);

-- migrate:down transaction:false
DROP INDEX CONCURRENTLY index_email_confirmations_on_user_id;
