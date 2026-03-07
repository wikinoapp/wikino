
-- Dumped from database version 18.1 (Debian 18.1-1.pgdg13+2)
-- Dumped by pg_dump version 18.3 (Debian 18.3-1.pgdg13+1)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: SCHEMA public; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON SCHEMA public IS '';


--
-- Name: citext; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS citext WITH SCHEMA public;


--
-- Name: EXTENSION citext; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION citext IS 'data type for case-insensitive character strings';


--
-- Name: pgcrypto; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;


--
-- Name: EXTENSION pgcrypto; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION pgcrypto IS 'cryptographic functions';


--
-- Name: river_job_state; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.river_job_state AS ENUM (
    'available',
    'cancelled',
    'completed',
    'discarded',
    'pending',
    'retryable',
    'running',
    'scheduled'
);


--
-- Name: generate_ulid(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.generate_ulid() RETURNS uuid
    LANGUAGE sql
    AS $$
  SELECT (lpad(to_hex(floor(extract(epoch FROM clock_timestamp()) * 1000)::bigint), 12, '0') || encode(gen_random_bytes(10), 'hex'))::uuid;
$$;


--
-- Name: river_job_state_in_bitmask(bit, public.river_job_state); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.river_job_state_in_bitmask(bitmask bit, state public.river_job_state) RETURNS boolean
    LANGUAGE sql IMMUTABLE
    AS $$
    SELECT CASE state
        WHEN 'available' THEN get_bit(bitmask, 7)
        WHEN 'cancelled' THEN get_bit(bitmask, 6)
        WHEN 'completed' THEN get_bit(bitmask, 5)
        WHEN 'discarded' THEN get_bit(bitmask, 4)
        WHEN 'pending'   THEN get_bit(bitmask, 3)
        WHEN 'retryable' THEN get_bit(bitmask, 2)
        WHEN 'running'   THEN get_bit(bitmask, 1)
        WHEN 'scheduled' THEN get_bit(bitmask, 0)
        ELSE 0
    END = 1;
$$;


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: active_storage_attachments; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.active_storage_attachments (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    name character varying NOT NULL,
    record_type character varying NOT NULL,
    record_id uuid NOT NULL,
    blob_id uuid NOT NULL,
    created_at timestamp(6) without time zone NOT NULL
);


--
-- Name: active_storage_blobs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.active_storage_blobs (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    key character varying NOT NULL,
    filename character varying NOT NULL,
    content_type character varying,
    metadata text,
    service_name character varying NOT NULL,
    byte_size bigint NOT NULL,
    checksum character varying,
    created_at timestamp(6) without time zone NOT NULL
);


--
-- Name: active_storage_variant_records; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.active_storage_variant_records (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    blob_id uuid NOT NULL,
    variation_digest character varying NOT NULL
);


--
-- Name: ar_internal_metadata; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.ar_internal_metadata (
    key character varying NOT NULL,
    value character varying,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: attachments; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.attachments (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    space_id uuid NOT NULL,
    active_storage_attachment_id uuid NOT NULL,
    attached_space_member_id uuid NOT NULL,
    attached_at timestamp(6) without time zone NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL,
    processing_status integer DEFAULT 0 NOT NULL
);


--
-- Name: draft_page_revisions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.draft_page_revisions (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    draft_page_id uuid NOT NULL,
    space_id uuid NOT NULL,
    space_member_id uuid NOT NULL,
    title character varying NOT NULL,
    body character varying NOT NULL,
    body_html character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: draft_pages; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.draft_pages (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    space_id uuid NOT NULL,
    page_id uuid NOT NULL,
    space_member_id uuid NOT NULL,
    topic_id uuid NOT NULL,
    title public.citext,
    body public.citext NOT NULL,
    body_html text NOT NULL,
    linked_page_ids character varying[] NOT NULL,
    modified_at timestamp(6) without time zone NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: email_confirmations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.email_confirmations (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    email character varying NOT NULL,
    event integer NOT NULL,
    code character varying NOT NULL,
    started_at timestamp without time zone NOT NULL,
    succeeded_at timestamp without time zone,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: export_statuses; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.export_statuses (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    space_id uuid NOT NULL,
    export_id uuid NOT NULL,
    kind integer NOT NULL,
    changed_at timestamp(6) without time zone NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: exports; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.exports (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    space_id uuid NOT NULL,
    queued_by_id uuid NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: feature_flags; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.feature_flags (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    user_id uuid NOT NULL,
    name character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: page_attachment_references; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.page_attachment_references (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    attachment_id uuid NOT NULL,
    page_id uuid NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: page_editors; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.page_editors (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    space_id uuid NOT NULL,
    page_id uuid NOT NULL,
    space_member_id uuid NOT NULL,
    last_page_modified_at timestamp(6) without time zone NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: page_revisions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.page_revisions (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    space_id uuid NOT NULL,
    space_member_id uuid NOT NULL,
    page_id uuid NOT NULL,
    body public.citext NOT NULL,
    body_html text NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL,
    title public.citext NOT NULL
);


--
-- Name: pages; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.pages (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    space_id uuid NOT NULL,
    topic_id uuid NOT NULL,
    number integer NOT NULL,
    title public.citext,
    body public.citext NOT NULL,
    body_html text NOT NULL,
    linked_page_ids character varying[] NOT NULL,
    modified_at timestamp(6) without time zone NOT NULL,
    published_at timestamp(6) without time zone,
    trashed_at timestamp(6) without time zone,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL,
    pinned_at timestamp without time zone,
    discarded_at timestamp(6) without time zone,
    featured_image_attachment_id uuid
);


--
-- Name: password_reset_tokens; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.password_reset_tokens (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    user_id uuid NOT NULL,
    token_digest character varying NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    used_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: rate_limits; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.rate_limits (
    id character varying DEFAULT public.generate_ulid() NOT NULL,
    key character varying NOT NULL,
    window_start timestamp with time zone NOT NULL,
    count integer DEFAULT 1 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: river_client; Type: TABLE; Schema: public; Owner: -
--

CREATE UNLOGGED TABLE public.river_client (
    id text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    paused_at timestamp with time zone,
    updated_at timestamp with time zone NOT NULL,
    CONSTRAINT name_length CHECK (((char_length(id) > 0) AND (char_length(id) < 128)))
);


--
-- Name: river_client_queue; Type: TABLE; Schema: public; Owner: -
--

CREATE UNLOGGED TABLE public.river_client_queue (
    river_client_id text NOT NULL,
    name text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    max_workers bigint DEFAULT 0 NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    num_jobs_completed bigint DEFAULT 0 NOT NULL,
    num_jobs_running bigint DEFAULT 0 NOT NULL,
    updated_at timestamp with time zone NOT NULL,
    CONSTRAINT name_length CHECK (((char_length(name) > 0) AND (char_length(name) < 128))),
    CONSTRAINT num_jobs_completed_zero_or_positive CHECK ((num_jobs_completed >= 0)),
    CONSTRAINT num_jobs_running_zero_or_positive CHECK ((num_jobs_running >= 0))
);


--
-- Name: river_job; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.river_job (
    id bigint NOT NULL,
    state public.river_job_state DEFAULT 'available'::public.river_job_state NOT NULL,
    attempt smallint DEFAULT 0 NOT NULL,
    max_attempts smallint NOT NULL,
    attempted_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    finalized_at timestamp with time zone,
    scheduled_at timestamp with time zone DEFAULT now() NOT NULL,
    priority smallint DEFAULT 1 NOT NULL,
    args jsonb NOT NULL,
    attempted_by text[],
    errors jsonb[],
    kind text NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    queue text DEFAULT 'default'::text NOT NULL,
    tags character varying(255)[] DEFAULT '{}'::character varying[] NOT NULL,
    unique_key bytea,
    unique_states bit(8),
    CONSTRAINT finalized_or_finalized_at_null CHECK ((((finalized_at IS NULL) AND (state <> ALL (ARRAY['cancelled'::public.river_job_state, 'completed'::public.river_job_state, 'discarded'::public.river_job_state]))) OR ((finalized_at IS NOT NULL) AND (state = ANY (ARRAY['cancelled'::public.river_job_state, 'completed'::public.river_job_state, 'discarded'::public.river_job_state]))))),
    CONSTRAINT kind_length CHECK (((char_length(kind) > 0) AND (char_length(kind) < 128))),
    CONSTRAINT max_attempts_is_positive CHECK ((max_attempts > 0)),
    CONSTRAINT priority_in_range CHECK (((priority >= 1) AND (priority <= 4))),
    CONSTRAINT queue_length CHECK (((char_length(queue) > 0) AND (char_length(queue) < 128)))
);


--
-- Name: river_job_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.river_job_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: river_job_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.river_job_id_seq OWNED BY public.river_job.id;


--
-- Name: river_leader; Type: TABLE; Schema: public; Owner: -
--

CREATE UNLOGGED TABLE public.river_leader (
    elected_at timestamp with time zone NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    leader_id text NOT NULL,
    name text DEFAULT 'default'::text NOT NULL,
    CONSTRAINT leader_id_length CHECK (((char_length(leader_id) > 0) AND (char_length(leader_id) < 128))),
    CONSTRAINT name_length CHECK ((name = 'default'::text))
);


--
-- Name: river_queue; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.river_queue (
    name text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    paused_at timestamp with time zone,
    updated_at timestamp with time zone NOT NULL
);


--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.schema_migrations (
    version character varying NOT NULL
);


--
-- Name: space_members; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.space_members (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    space_id uuid NOT NULL,
    user_id uuid NOT NULL,
    role integer NOT NULL,
    joined_at timestamp without time zone NOT NULL,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: spaces; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.spaces (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    identifier public.citext NOT NULL,
    name character varying NOT NULL,
    plan integer NOT NULL,
    joined_at timestamp without time zone NOT NULL,
    discarded_at timestamp without time zone,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: topic_members; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.topic_members (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    space_id uuid NOT NULL,
    topic_id uuid NOT NULL,
    space_member_id uuid NOT NULL,
    role integer NOT NULL,
    joined_at timestamp without time zone NOT NULL,
    last_page_modified_at timestamp(6) without time zone,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: topics; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.topics (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    space_id uuid NOT NULL,
    number integer NOT NULL,
    name character varying NOT NULL,
    description character varying NOT NULL,
    visibility integer NOT NULL,
    discarded_at timestamp without time zone,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: user_passwords; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_passwords (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    user_id uuid NOT NULL,
    password_digest character varying NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: user_sessions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_sessions (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    user_id uuid NOT NULL,
    token character varying NOT NULL,
    ip_address character varying NOT NULL,
    user_agent character varying NOT NULL,
    signed_in_at timestamp(6) without time zone NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: user_two_factor_auths; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_two_factor_auths (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    user_id uuid NOT NULL,
    secret character varying NOT NULL,
    enabled boolean DEFAULT false NOT NULL,
    enabled_at timestamp(6) without time zone,
    recovery_codes character varying[] DEFAULT '{}'::character varying[] NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    email character varying NOT NULL,
    atname public.citext NOT NULL,
    name character varying NOT NULL,
    description character varying NOT NULL,
    locale integer NOT NULL,
    time_zone character varying NOT NULL,
    joined_at timestamp without time zone NOT NULL,
    discarded_at timestamp without time zone,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: river_job id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.river_job ALTER COLUMN id SET DEFAULT nextval('public.river_job_id_seq'::regclass);


--
-- Name: active_storage_attachments active_storage_attachments_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.active_storage_attachments
    ADD CONSTRAINT active_storage_attachments_pkey PRIMARY KEY (id);


--
-- Name: active_storage_blobs active_storage_blobs_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.active_storage_blobs
    ADD CONSTRAINT active_storage_blobs_pkey PRIMARY KEY (id);


--
-- Name: active_storage_variant_records active_storage_variant_records_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.active_storage_variant_records
    ADD CONSTRAINT active_storage_variant_records_pkey PRIMARY KEY (id);


--
-- Name: ar_internal_metadata ar_internal_metadata_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ar_internal_metadata
    ADD CONSTRAINT ar_internal_metadata_pkey PRIMARY KEY (key);


--
-- Name: attachments attachments_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.attachments
    ADD CONSTRAINT attachments_pkey PRIMARY KEY (id);


--
-- Name: draft_page_revisions draft_page_revisions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_page_revisions
    ADD CONSTRAINT draft_page_revisions_pkey PRIMARY KEY (id);


--
-- Name: draft_pages draft_pages_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_pages
    ADD CONSTRAINT draft_pages_pkey PRIMARY KEY (id);


--
-- Name: email_confirmations email_confirmations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.email_confirmations
    ADD CONSTRAINT email_confirmations_pkey PRIMARY KEY (id);


--
-- Name: export_statuses export_statuses_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.export_statuses
    ADD CONSTRAINT export_statuses_pkey PRIMARY KEY (id);


--
-- Name: exports exports_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.exports
    ADD CONSTRAINT exports_pkey PRIMARY KEY (id);


--
-- Name: feature_flags feature_flags_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feature_flags
    ADD CONSTRAINT feature_flags_pkey PRIMARY KEY (id);


--
-- Name: feature_flags feature_flags_user_id_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feature_flags
    ADD CONSTRAINT feature_flags_user_id_name_key UNIQUE (user_id, name);


--
-- Name: page_attachment_references page_attachment_references_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.page_attachment_references
    ADD CONSTRAINT page_attachment_references_pkey PRIMARY KEY (id);


--
-- Name: page_editors page_editorships_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.page_editors
    ADD CONSTRAINT page_editorships_pkey PRIMARY KEY (id);


--
-- Name: page_revisions page_revisions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.page_revisions
    ADD CONSTRAINT page_revisions_pkey PRIMARY KEY (id);


--
-- Name: pages pages_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pages
    ADD CONSTRAINT pages_pkey PRIMARY KEY (id);


--
-- Name: password_reset_tokens password_reset_tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.password_reset_tokens
    ADD CONSTRAINT password_reset_tokens_pkey PRIMARY KEY (id);


--
-- Name: rate_limits rate_limits_key_window_start_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rate_limits
    ADD CONSTRAINT rate_limits_key_window_start_key UNIQUE (key, window_start);


--
-- Name: rate_limits rate_limits_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rate_limits
    ADD CONSTRAINT rate_limits_pkey PRIMARY KEY (id);


--
-- Name: river_client river_client_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.river_client
    ADD CONSTRAINT river_client_pkey PRIMARY KEY (id);


--
-- Name: river_client_queue river_client_queue_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.river_client_queue
    ADD CONSTRAINT river_client_queue_pkey PRIMARY KEY (river_client_id, name);


--
-- Name: river_job river_job_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.river_job
    ADD CONSTRAINT river_job_pkey PRIMARY KEY (id);


--
-- Name: river_leader river_leader_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.river_leader
    ADD CONSTRAINT river_leader_pkey PRIMARY KEY (name);


--
-- Name: river_queue river_queue_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.river_queue
    ADD CONSTRAINT river_queue_pkey PRIMARY KEY (name);


--
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);


--
-- Name: user_sessions sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_sessions
    ADD CONSTRAINT sessions_pkey PRIMARY KEY (id);


--
-- Name: space_members space_members_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.space_members
    ADD CONSTRAINT space_members_pkey PRIMARY KEY (id);


--
-- Name: spaces spaces_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.spaces
    ADD CONSTRAINT spaces_pkey PRIMARY KEY (id);


--
-- Name: topic_members topic_memberships_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.topic_members
    ADD CONSTRAINT topic_memberships_pkey PRIMARY KEY (id);


--
-- Name: topics topics_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.topics
    ADD CONSTRAINT topics_pkey PRIMARY KEY (id);


--
-- Name: user_passwords user_passwords_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_passwords
    ADD CONSTRAINT user_passwords_pkey PRIMARY KEY (id);


--
-- Name: user_two_factor_auths user_two_factor_auths_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_two_factor_auths
    ADD CONSTRAINT user_two_factor_auths_pkey PRIMARY KEY (id);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: idx_draft_page_revisions_draft_page_id_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_draft_page_revisions_draft_page_id_created_at ON public.draft_page_revisions USING btree (draft_page_id, created_at);


--
-- Name: idx_feature_flags_name; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_feature_flags_name ON public.feature_flags USING btree (name);


--
-- Name: idx_feature_flags_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_feature_flags_user_id ON public.feature_flags USING btree (user_id);


--
-- Name: idx_password_reset_tokens_token_digest; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_password_reset_tokens_token_digest ON public.password_reset_tokens USING btree (token_digest);


--
-- Name: idx_password_reset_tokens_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_password_reset_tokens_user_id ON public.password_reset_tokens USING btree (user_id);


--
-- Name: idx_rate_limits_key_window_start; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_rate_limits_key_window_start ON public.rate_limits USING btree (key, window_start);


--
-- Name: idx_rate_limits_window_start; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_rate_limits_window_start ON public.rate_limits USING btree (window_start);


--
-- Name: index_active_storage_attachments_on_blob_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_active_storage_attachments_on_blob_id ON public.active_storage_attachments USING btree (blob_id);


--
-- Name: index_active_storage_attachments_uniqueness; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_active_storage_attachments_uniqueness ON public.active_storage_attachments USING btree (record_type, record_id, name, blob_id);


--
-- Name: index_active_storage_blobs_on_key; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_active_storage_blobs_on_key ON public.active_storage_blobs USING btree (key);


--
-- Name: index_active_storage_variant_records_uniqueness; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_active_storage_variant_records_uniqueness ON public.active_storage_variant_records USING btree (blob_id, variation_digest);


--
-- Name: index_attachments_on_active_storage_attachment_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_attachments_on_active_storage_attachment_id ON public.attachments USING btree (active_storage_attachment_id);


--
-- Name: index_attachments_on_attached_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_attachments_on_attached_at ON public.attachments USING btree (attached_at);


--
-- Name: index_attachments_on_attached_space_member_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_attachments_on_attached_space_member_id ON public.attachments USING btree (attached_space_member_id);


--
-- Name: index_attachments_on_processing_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_attachments_on_processing_status ON public.attachments USING btree (processing_status);


--
-- Name: index_attachments_on_space_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_attachments_on_space_id ON public.attachments USING btree (space_id);


--
-- Name: index_draft_pages_on_linked_page_ids; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_draft_pages_on_linked_page_ids ON public.draft_pages USING gin (linked_page_ids);


--
-- Name: index_draft_pages_on_page_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_draft_pages_on_page_id ON public.draft_pages USING btree (page_id);


--
-- Name: index_draft_pages_on_space_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_draft_pages_on_space_id ON public.draft_pages USING btree (space_id);


--
-- Name: index_draft_pages_on_space_member_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_draft_pages_on_space_member_id ON public.draft_pages USING btree (space_member_id);


--
-- Name: index_draft_pages_on_space_member_id_and_page_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_draft_pages_on_space_member_id_and_page_id ON public.draft_pages USING btree (space_member_id, page_id);


--
-- Name: index_draft_pages_on_topic_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_draft_pages_on_topic_id ON public.draft_pages USING btree (topic_id);


--
-- Name: index_email_confirmations_on_code; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_email_confirmations_on_code ON public.email_confirmations USING btree (code);


--
-- Name: index_email_confirmations_on_started_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_email_confirmations_on_started_at ON public.email_confirmations USING btree (started_at);


--
-- Name: index_export_statuses_on_export_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_export_statuses_on_export_id ON public.export_statuses USING btree (export_id);


--
-- Name: index_export_statuses_on_space_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_export_statuses_on_space_id ON public.export_statuses USING btree (space_id);


--
-- Name: index_exports_on_queued_by_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_exports_on_queued_by_id ON public.exports USING btree (queued_by_id);


--
-- Name: index_exports_on_space_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_exports_on_space_id ON public.exports USING btree (space_id);


--
-- Name: index_page_attachment_references_on_attachment_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_page_attachment_references_on_attachment_id ON public.page_attachment_references USING btree (attachment_id);


--
-- Name: index_page_attachment_references_on_page_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_page_attachment_references_on_page_id ON public.page_attachment_references USING btree (page_id);


--
-- Name: index_page_attachment_references_on_page_id_and_attachment_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_page_attachment_references_on_page_id_and_attachment_id ON public.page_attachment_references USING btree (page_id, attachment_id);


--
-- Name: index_page_editors_on_page_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_page_editors_on_page_id ON public.page_editors USING btree (page_id);


--
-- Name: index_page_editors_on_page_id_and_space_member_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_page_editors_on_page_id_and_space_member_id ON public.page_editors USING btree (page_id, space_member_id);


--
-- Name: index_page_editors_on_space_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_page_editors_on_space_id ON public.page_editors USING btree (space_id);


--
-- Name: index_page_editors_on_space_member_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_page_editors_on_space_member_id ON public.page_editors USING btree (space_member_id);


--
-- Name: index_page_revisions_on_page_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_page_revisions_on_page_id ON public.page_revisions USING btree (page_id);


--
-- Name: index_page_revisions_on_space_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_page_revisions_on_space_id ON public.page_revisions USING btree (space_id);


--
-- Name: index_page_revisions_on_space_member_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_page_revisions_on_space_member_id ON public.page_revisions USING btree (space_member_id);


--
-- Name: index_pages_on_discarded_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_pages_on_discarded_at ON public.pages USING btree (discarded_at);


--
-- Name: index_pages_on_featured_image_attachment_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_pages_on_featured_image_attachment_id ON public.pages USING btree (featured_image_attachment_id);


--
-- Name: index_pages_on_linked_page_ids; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_pages_on_linked_page_ids ON public.pages USING gin (linked_page_ids);


--
-- Name: index_pages_on_space_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_pages_on_space_id ON public.pages USING btree (space_id);


--
-- Name: index_pages_on_space_id_and_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_pages_on_space_id_and_created_at ON public.pages USING btree (space_id, created_at);


--
-- Name: index_pages_on_space_id_and_modified_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_pages_on_space_id_and_modified_at ON public.pages USING btree (space_id, modified_at);


--
-- Name: index_pages_on_space_id_and_number; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_pages_on_space_id_and_number ON public.pages USING btree (space_id, number);


--
-- Name: index_pages_on_space_id_and_pinned_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_pages_on_space_id_and_pinned_at ON public.pages USING btree (space_id, pinned_at);


--
-- Name: index_pages_on_space_id_and_published_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_pages_on_space_id_and_published_at ON public.pages USING btree (space_id, published_at);


--
-- Name: index_pages_on_space_id_and_trashed_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_pages_on_space_id_and_trashed_at ON public.pages USING btree (space_id, trashed_at);


--
-- Name: index_pages_on_topic_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_pages_on_topic_id ON public.pages USING btree (topic_id);


--
-- Name: index_pages_on_topic_id_and_title; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_pages_on_topic_id_and_title ON public.pages USING btree (topic_id, title);


--
-- Name: index_space_members_on_space_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_space_members_on_space_id ON public.space_members USING btree (space_id);


--
-- Name: index_space_members_on_space_id_and_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_space_members_on_space_id_and_user_id ON public.space_members USING btree (space_id, user_id);


--
-- Name: index_space_members_on_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_space_members_on_user_id ON public.space_members USING btree (user_id);


--
-- Name: index_spaces_on_discarded_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_spaces_on_discarded_at ON public.spaces USING btree (discarded_at);


--
-- Name: index_spaces_on_identifier; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_spaces_on_identifier ON public.spaces USING btree (identifier);


--
-- Name: index_topic_members_on_space_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_topic_members_on_space_id ON public.topic_members USING btree (space_id);


--
-- Name: index_topic_members_on_space_member_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_topic_members_on_space_member_id ON public.topic_members USING btree (space_member_id);


--
-- Name: index_topic_members_on_topic_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_topic_members_on_topic_id ON public.topic_members USING btree (topic_id);


--
-- Name: index_topic_members_on_topic_id_and_space_member_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_topic_members_on_topic_id_and_space_member_id ON public.topic_members USING btree (topic_id, space_member_id);


--
-- Name: index_topics_on_discarded_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_topics_on_discarded_at ON public.topics USING btree (discarded_at);


--
-- Name: index_topics_on_space_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_topics_on_space_id ON public.topics USING btree (space_id);


--
-- Name: index_topics_on_space_id_and_discarded_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_topics_on_space_id_and_discarded_at ON public.topics USING btree (space_id, discarded_at);


--
-- Name: index_topics_on_space_id_and_name; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_topics_on_space_id_and_name ON public.topics USING btree (space_id, name);


--
-- Name: index_topics_on_space_id_and_number; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_topics_on_space_id_and_number ON public.topics USING btree (space_id, number);


--
-- Name: index_user_passwords_on_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_user_passwords_on_user_id ON public.user_passwords USING btree (user_id);


--
-- Name: index_user_sessions_on_token; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_user_sessions_on_token ON public.user_sessions USING btree (token);


--
-- Name: index_user_sessions_on_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_user_sessions_on_user_id ON public.user_sessions USING btree (user_id);


--
-- Name: index_user_two_factor_auths_on_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_user_two_factor_auths_on_user_id ON public.user_two_factor_auths USING btree (user_id);


--
-- Name: index_users_on_atname; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_users_on_atname ON public.users USING btree (atname);


--
-- Name: index_users_on_discarded_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_users_on_discarded_at ON public.users USING btree (discarded_at);


--
-- Name: index_users_on_email; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_users_on_email ON public.users USING btree (email);


--
-- Name: river_job_args_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX river_job_args_index ON public.river_job USING gin (args);


--
-- Name: river_job_kind; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX river_job_kind ON public.river_job USING btree (kind);


--
-- Name: river_job_metadata_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX river_job_metadata_index ON public.river_job USING gin (metadata);


--
-- Name: river_job_prioritized_fetching_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX river_job_prioritized_fetching_index ON public.river_job USING btree (state, queue, priority, scheduled_at, id);


--
-- Name: river_job_state_and_finalized_at_index; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX river_job_state_and_finalized_at_index ON public.river_job USING btree (state, finalized_at) WHERE (finalized_at IS NOT NULL);


--
-- Name: river_job_unique_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX river_job_unique_idx ON public.river_job USING btree (unique_key) WHERE ((unique_key IS NOT NULL) AND (unique_states IS NOT NULL) AND public.river_job_state_in_bitmask(unique_states, state));


--
-- Name: draft_page_revisions draft_page_revisions_draft_page_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_page_revisions
    ADD CONSTRAINT draft_page_revisions_draft_page_id_fkey FOREIGN KEY (draft_page_id) REFERENCES public.draft_pages(id);


--
-- Name: draft_page_revisions draft_page_revisions_space_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_page_revisions
    ADD CONSTRAINT draft_page_revisions_space_id_fkey FOREIGN KEY (space_id) REFERENCES public.spaces(id);


--
-- Name: draft_page_revisions draft_page_revisions_space_member_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_page_revisions
    ADD CONSTRAINT draft_page_revisions_space_member_id_fkey FOREIGN KEY (space_member_id) REFERENCES public.space_members(id);


--
-- Name: feature_flags feature_flags_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.feature_flags
    ADD CONSTRAINT feature_flags_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: attachments fk_rails_06223f0ea2; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.attachments
    ADD CONSTRAINT fk_rails_06223f0ea2 FOREIGN KEY (space_id) REFERENCES public.spaces(id);


--
-- Name: topic_members fk_rails_0f8ef246f7; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.topic_members
    ADD CONSTRAINT fk_rails_0f8ef246f7 FOREIGN KEY (space_id) REFERENCES public.spaces(id);


--
-- Name: page_attachment_references fk_rails_1bb2aa81d8; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.page_attachment_references
    ADD CONSTRAINT fk_rails_1bb2aa81d8 FOREIGN KEY (page_id) REFERENCES public.pages(id);


--
-- Name: page_editors fk_rails_2088082077; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.page_editors
    ADD CONSTRAINT fk_rails_2088082077 FOREIGN KEY (space_member_id) REFERENCES public.space_members(id);


--
-- Name: space_members fk_rails_26446af0e7; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.space_members
    ADD CONSTRAINT fk_rails_26446af0e7 FOREIGN KEY (space_id) REFERENCES public.spaces(id);


--
-- Name: page_attachment_references fk_rails_2cba0cae86; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.page_attachment_references
    ADD CONSTRAINT fk_rails_2cba0cae86 FOREIGN KEY (attachment_id) REFERENCES public.attachments(id);


--
-- Name: page_editors fk_rails_3b3700fcdf; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.page_editors
    ADD CONSTRAINT fk_rails_3b3700fcdf FOREIGN KEY (space_id) REFERENCES public.spaces(id);


--
-- Name: topic_members fk_rails_50149efe6b; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.topic_members
    ADD CONSTRAINT fk_rails_50149efe6b FOREIGN KEY (topic_id) REFERENCES public.topics(id);


--
-- Name: pages fk_rails_5d9ff2d9dc; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pages
    ADD CONSTRAINT fk_rails_5d9ff2d9dc FOREIGN KEY (space_id) REFERENCES public.spaces(id);


--
-- Name: page_revisions fk_rails_6eb3eeb6b7; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.page_revisions
    ADD CONSTRAINT fk_rails_6eb3eeb6b7 FOREIGN KEY (page_id) REFERENCES public.pages(id);


--
-- Name: exports fk_rails_703ee3dae6; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.exports
    ADD CONSTRAINT fk_rails_703ee3dae6 FOREIGN KEY (queued_by_id) REFERENCES public.space_members(id);


--
-- Name: page_revisions fk_rails_74648de0a3; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.page_revisions
    ADD CONSTRAINT fk_rails_74648de0a3 FOREIGN KEY (space_member_id) REFERENCES public.space_members(id);


--
-- Name: user_sessions fk_rails_758836b4f0; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_sessions
    ADD CONSTRAINT fk_rails_758836b4f0 FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: pages fk_rails_793c81c055; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pages
    ADD CONSTRAINT fk_rails_793c81c055 FOREIGN KEY (topic_id) REFERENCES public.topics(id);


--
-- Name: exports fk_rails_7fa4a1a0c0; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.exports
    ADD CONSTRAINT fk_rails_7fa4a1a0c0 FOREIGN KEY (space_id) REFERENCES public.spaces(id);


--
-- Name: topic_members fk_rails_80fd6512fa; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.topic_members
    ADD CONSTRAINT fk_rails_80fd6512fa FOREIGN KEY (space_member_id) REFERENCES public.space_members(id);


--
-- Name: attachments fk_rails_850712f875; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.attachments
    ADD CONSTRAINT fk_rails_850712f875 FOREIGN KEY (attached_space_member_id) REFERENCES public.space_members(id);


--
-- Name: pages fk_rails_89b3a4bafa; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pages
    ADD CONSTRAINT fk_rails_89b3a4bafa FOREIGN KEY (featured_image_attachment_id) REFERENCES public.attachments(id);


--
-- Name: draft_pages fk_rails_8d9bc1217e; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_pages
    ADD CONSTRAINT fk_rails_8d9bc1217e FOREIGN KEY (space_member_id) REFERENCES public.space_members(id);


--
-- Name: draft_pages fk_rails_8e68719216; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_pages
    ADD CONSTRAINT fk_rails_8e68719216 FOREIGN KEY (topic_id) REFERENCES public.topics(id);


--
-- Name: active_storage_variant_records fk_rails_993965df05; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.active_storage_variant_records
    ADD CONSTRAINT fk_rails_993965df05 FOREIGN KEY (blob_id) REFERENCES public.active_storage_blobs(id);


--
-- Name: topics fk_rails_9b3ff1bd6e; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.topics
    ADD CONSTRAINT fk_rails_9b3ff1bd6e FOREIGN KEY (space_id) REFERENCES public.spaces(id);


--
-- Name: attachments fk_rails_a2990ed7e9; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.attachments
    ADD CONSTRAINT fk_rails_a2990ed7e9 FOREIGN KEY (active_storage_attachment_id) REFERENCES public.active_storage_attachments(id);


--
-- Name: space_members fk_rails_a7900d8de9; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.space_members
    ADD CONSTRAINT fk_rails_a7900d8de9 FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: export_statuses fk_rails_a8d9f2050b; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.export_statuses
    ADD CONSTRAINT fk_rails_a8d9f2050b FOREIGN KEY (export_id) REFERENCES public.exports(id);


--
-- Name: draft_pages fk_rails_a989662ed2; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_pages
    ADD CONSTRAINT fk_rails_a989662ed2 FOREIGN KEY (space_id) REFERENCES public.spaces(id);


--
-- Name: draft_pages fk_rails_b87bfd2937; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_pages
    ADD CONSTRAINT fk_rails_b87bfd2937 FOREIGN KEY (page_id) REFERENCES public.pages(id);


--
-- Name: active_storage_attachments fk_rails_c3b3935057; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.active_storage_attachments
    ADD CONSTRAINT fk_rails_c3b3935057 FOREIGN KEY (blob_id) REFERENCES public.active_storage_blobs(id);


--
-- Name: user_passwords fk_rails_c7888e4144; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_passwords
    ADD CONSTRAINT fk_rails_c7888e4144 FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: export_statuses fk_rails_cab71249f9; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.export_statuses
    ADD CONSTRAINT fk_rails_cab71249f9 FOREIGN KEY (space_id) REFERENCES public.spaces(id);


--
-- Name: page_revisions fk_rails_d3466381ba; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.page_revisions
    ADD CONSTRAINT fk_rails_d3466381ba FOREIGN KEY (space_id) REFERENCES public.spaces(id);


--
-- Name: page_editors fk_rails_fc18bdf1dd; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.page_editors
    ADD CONSTRAINT fk_rails_fc18bdf1dd FOREIGN KEY (page_id) REFERENCES public.pages(id);


--
-- Name: user_two_factor_auths fk_rails_fd6d01946c; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_two_factor_auths
    ADD CONSTRAINT fk_rails_fd6d01946c FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: password_reset_tokens password_reset_tokens_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.password_reset_tokens
    ADD CONSTRAINT password_reset_tokens_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: river_client_queue river_client_queue_river_client_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.river_client_queue
    ADD CONSTRAINT river_client_queue_river_client_id_fkey FOREIGN KEY (river_client_id) REFERENCES public.river_client(id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--



--
-- Dbmate schema migrations
--

INSERT INTO public.schema_migrations (version) VALUES
    ('20240000000001'),
    ('20240000000002'),
    ('20241104061506'),
    ('20241201145213'),
    ('20241220175102'),
    ('20250119163728'),
    ('20250202063341'),
    ('20250202165124'),
    ('20250202165519'),
    ('20250204164034'),
    ('20250317095826'),
    ('20250323083136'),
    ('20250323084054'),
    ('20250517113233'),
    ('20250517113234'),
    ('20250517113235'),
    ('20250526173629'),
    ('20250730164526'),
    ('20250730164550'),
    ('20250802185226'),
    ('20250802185227'),
    ('20250830075345'),
    ('20250830075516'),
    ('20250920074019'),
    ('20250920082216'),
    ('20260202060000'),
    ('20260202160000'),
    ('20260204160000'),
    ('20260301154347'),
    ('20260305154013');
