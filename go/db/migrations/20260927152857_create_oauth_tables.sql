-- migrate:up

-- 公開APIのOAuth 2.0認可サーバーのテーブル。
--
-- OAuthアプリ (oauth_applications) はスペースが持ち、登録したスペースでだけ使える。
-- space_idがNULLのアプリは、全スペース共通のWikinoの公式クライアント (CLI・デスクトップ) である。
-- 利用者がアプリを許可すると、スペースメンバー単位の許可 (oauth_grants) を作り、
-- 認可コード・アクセストークン・リフレッシュトークンはどれもその許可に属する。
-- トークンの持ち主と束縛先のスペースは許可から決まる。
--
-- 認可コード・トークン・クライアントシークレットの値そのものは保存せず、SHA-256の
-- ダイジェストだけを持つ。scopesはspace_members.scopesと同じ形で持つ。
--
-- 許可に属する行は、束縛先のスペースを取り違えないよう (oauth_grant_id, space_id) の
-- 複合外部キーで許可を参照する。親が消えたら一緒に消えるべきデータなので、外部キーは
-- ON DELETE CASCADEにする。

-- 作成したメンバーはcreated_space_member_idに記録する。作成したメンバーがいなくなっても
-- アプリと他のメンバーの連携は残るよう、メンバーが消えたらこの列だけをNULLにする。
-- 作成者を同じスペースのメンバーに限定するため、space_idとの複合外部キーにする。
--
-- client_idは内部の主キーと分け、将来Client ID Metadata Document (URLのclient_id) を
-- 受け入れられるようTEXTにする。client_typeは0がconfidential、1がpublicで、
-- confidentialクライアントだけがシークレットを持つ
CREATE TABLE oauth_applications (
    id UUID NOT NULL DEFAULT generate_ulid() PRIMARY KEY,
    space_id UUID REFERENCES spaces(id) ON DELETE CASCADE,
    created_space_member_id UUID,
    name VARCHAR NOT NULL,
    client_id TEXT NOT NULL,
    client_secret_digest VARCHAR,
    client_type INTEGER NOT NULL,
    redirect_uris TEXT[] NOT NULL DEFAULT '{}',
    discarded_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT oauth_applications_created_space_member_id_space_id_fkey
        FOREIGN KEY (created_space_member_id, space_id)
        REFERENCES space_members(id, space_id) ON DELETE SET NULL (created_space_member_id),
    CONSTRAINT chk_oauth_applications_client_type CHECK (client_type IN (0, 1)),
    CONSTRAINT chk_oauth_applications_client_secret_digest
        CHECK ((client_type = 0) = (client_secret_digest IS NOT NULL))
);

CREATE UNIQUE INDEX idx_oauth_applications_client_id ON oauth_applications(client_id);
CREATE INDEX idx_oauth_applications_space_id ON oauth_applications(space_id);
CREATE INDEX idx_oauth_applications_created_space_member_id ON oauth_applications(created_space_member_id);

-- 持ち主を束縛先スペースのメンバーに限定するため、space_member_idはspace_idとの複合
-- 外部キーにする (personal_access_tokensと同じ)。同じアプリ・メンバーに有効な許可は1つに限る。
-- 公式クライアントはスペースを持たないため、アプリへの参照は複合外部キーにできない。
-- 「アプリのspace_idがNULLか、許可のspace_idと一致する」ことはアプリケーションで確かめる
CREATE TABLE oauth_grants (
    id UUID NOT NULL DEFAULT generate_ulid() PRIMARY KEY,
    oauth_application_id UUID NOT NULL REFERENCES oauth_applications(id) ON DELETE CASCADE,
    space_id UUID NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
    space_member_id UUID NOT NULL,
    scopes TEXT[] NOT NULL DEFAULT '{}',
    revoked_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT oauth_grants_space_member_id_space_id_fkey
        FOREIGN KEY (space_member_id, space_id)
        REFERENCES space_members(id, space_id) ON DELETE CASCADE,
    -- idは単独でも一意だが、複合外部キーの参照先にするため組み合わせにも一意制約を置く
    CONSTRAINT oauth_grants_id_space_id_key UNIQUE (id, space_id)
);

CREATE UNIQUE INDEX idx_oauth_grants_unrevoked_application_member
    ON oauth_grants(oauth_application_id, space_member_id) WHERE revoked_at IS NULL;
CREATE INDEX idx_oauth_grants_oauth_application_id ON oauth_grants(oauth_application_id);
CREATE INDEX idx_oauth_grants_space_id ON oauth_grants(space_id);
CREATE INDEX idx_oauth_grants_space_member_id ON oauth_grants(space_member_id);

-- 認可コード。PKCEはS256だけを受け付けるため、code_challenge_methodは持たない。
-- トークン要求でリダイレクトURIの一致を確かめるため、認可要求のredirect_uriを残す
CREATE TABLE oauth_authorization_codes (
    id UUID NOT NULL DEFAULT generate_ulid() PRIMARY KEY,
    oauth_grant_id UUID NOT NULL,
    space_id UUID NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
    code_digest VARCHAR NOT NULL,
    scopes TEXT[] NOT NULL DEFAULT '{}',
    redirect_uri TEXT NOT NULL,
    code_challenge VARCHAR NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT oauth_authorization_codes_oauth_grant_id_space_id_fkey
        FOREIGN KEY (oauth_grant_id, space_id)
        REFERENCES oauth_grants(id, space_id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX idx_oauth_authorization_codes_code_digest ON oauth_authorization_codes(code_digest);
CREATE INDEX idx_oauth_authorization_codes_oauth_grant_id ON oauth_authorization_codes(oauth_grant_id);
CREATE INDEX idx_oauth_authorization_codes_space_id ON oauth_authorization_codes(space_id);

CREATE TABLE oauth_access_tokens (
    id UUID NOT NULL DEFAULT generate_ulid() PRIMARY KEY,
    oauth_grant_id UUID NOT NULL,
    space_id UUID NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
    token_digest VARCHAR NOT NULL,
    scopes TEXT[] NOT NULL DEFAULT '{}',
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    revoked_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT oauth_access_tokens_oauth_grant_id_space_id_fkey
        FOREIGN KEY (oauth_grant_id, space_id)
        REFERENCES oauth_grants(id, space_id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX idx_oauth_access_tokens_token_digest ON oauth_access_tokens(token_digest);
CREATE INDEX idx_oauth_access_tokens_oauth_grant_id ON oauth_access_tokens(oauth_grant_id);
CREATE INDEX idx_oauth_access_tokens_space_id ON oauth_access_tokens(space_id);

-- リフレッシュトークンは使うたびにローテーションする。previous_refresh_token_idは
-- ローテーション前のトークンを指す。scopesは更新で発行するアクセストークンのスコープで、
-- 許可のスコープが後から変わっても、認可したときの範囲を超えないようにトークンが持つ
CREATE TABLE oauth_refresh_tokens (
    id UUID NOT NULL DEFAULT generate_ulid() PRIMARY KEY,
    oauth_grant_id UUID NOT NULL,
    space_id UUID NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
    token_digest VARCHAR NOT NULL,
    scopes TEXT[] NOT NULL DEFAULT '{}',
    previous_refresh_token_id UUID REFERENCES oauth_refresh_tokens(id) ON DELETE SET NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    used_at TIMESTAMP WITH TIME ZONE,
    revoked_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT oauth_refresh_tokens_oauth_grant_id_space_id_fkey
        FOREIGN KEY (oauth_grant_id, space_id)
        REFERENCES oauth_grants(id, space_id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX idx_oauth_refresh_tokens_token_digest ON oauth_refresh_tokens(token_digest);
CREATE INDEX idx_oauth_refresh_tokens_oauth_grant_id ON oauth_refresh_tokens(oauth_grant_id);
CREATE INDEX idx_oauth_refresh_tokens_space_id ON oauth_refresh_tokens(space_id);
CREATE INDEX idx_oauth_refresh_tokens_previous_refresh_token_id ON oauth_refresh_tokens(previous_refresh_token_id);

-- migrate:down

DROP TABLE IF EXISTS oauth_refresh_tokens;
DROP TABLE IF EXISTS oauth_access_tokens;
DROP TABLE IF EXISTS oauth_authorization_codes;
DROP TABLE IF EXISTS oauth_grants;
DROP TABLE IF EXISTS oauth_applications;
