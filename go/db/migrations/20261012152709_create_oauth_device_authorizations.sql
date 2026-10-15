-- migrate:up

-- デバイス認可 (RFC 8628) の要求。ブラウザを開けない環境のクライアントが要求を作り、利用者が
-- 別の端末でユーザーコードを入力して許可すると、クライアントはデバイスコードでトークンを受け取る。
-- 連携先のスペースは要求の時点で決まっている (アプリのスペースか、公式クライアントでは `resource`)。
--
-- デバイスコードとユーザーコードの値そのものは保存せず、SHA-256のダイジェストだけを持つ。
-- ユーザーコードは区切りを除き大文字にそろえた形のダイジェストである。
-- scopesは要求されたスコープで、許可したときに利用者が選んだ範囲に置き換える。
-- 認可コードのscopesと同じく、交換で発行するトークンのスコープになる。
--
-- statusは0が許可待ち、1が許可済み、2が拒否、3がトークンと交換済みである。
-- 許可すると許可 (oauth_grants) を作ってoauth_grant_idに持たせ、交換はstatusを1から3へ
-- 変えることで1回限りにする。束縛先のスペースを取り違えないよう、許可は (oauth_grant_id, space_id)
-- の複合外部キーで参照する。
--
-- interval_secondsはクライアントがポーリングする間隔で、短すぎるポーリング (slow_down) の
-- たびに延ばす。last_polled_atは間隔を確かめるための最後のポーリングの日時である。
CREATE TABLE oauth_device_authorizations (
    id UUID NOT NULL DEFAULT generate_ulid() PRIMARY KEY,
    oauth_application_id UUID NOT NULL REFERENCES oauth_applications(id) ON DELETE CASCADE,
    space_id UUID NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
    oauth_grant_id UUID,
    device_code_digest VARCHAR NOT NULL,
    user_code_digest VARCHAR NOT NULL,
    scopes TEXT[] NOT NULL DEFAULT '{}',
    status INTEGER NOT NULL DEFAULT 0,
    interval_seconds INTEGER NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    last_polled_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT oauth_device_authorizations_oauth_grant_id_space_id_fkey
        FOREIGN KEY (oauth_grant_id, space_id)
        REFERENCES oauth_grants(id, space_id) ON DELETE CASCADE,
    CONSTRAINT chk_oauth_device_authorizations_status CHECK (status IN (0, 1, 2, 3)),
    -- 許可済みと交換済みの要求だけが許可を持つ
    CONSTRAINT chk_oauth_device_authorizations_oauth_grant_id
        CHECK ((status IN (1, 3)) = (oauth_grant_id IS NOT NULL)),
    CONSTRAINT chk_oauth_device_authorizations_interval_seconds CHECK (interval_seconds > 0)
);

-- ユーザーコードは8文字で衝突しうるため、一意制約に違反したら作り直す。
-- 照合で引く行が1つに決まるよう、期限切れの要求も含めて一意にする
CREATE UNIQUE INDEX idx_oauth_device_authorizations_device_code_digest ON oauth_device_authorizations(device_code_digest);
CREATE UNIQUE INDEX idx_oauth_device_authorizations_user_code_digest ON oauth_device_authorizations(user_code_digest);
CREATE INDEX idx_oauth_device_authorizations_oauth_application_id ON oauth_device_authorizations(oauth_application_id);
CREATE INDEX idx_oauth_device_authorizations_space_id ON oauth_device_authorizations(space_id);
CREATE INDEX idx_oauth_device_authorizations_oauth_grant_id ON oauth_device_authorizations(oauth_grant_id);

-- migrate:down

DROP TABLE IF EXISTS oauth_device_authorizations;
