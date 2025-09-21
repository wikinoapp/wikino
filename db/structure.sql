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
-- Name: public; Type: SCHEMA; Schema: -; Owner: -
--

-- *not* creating schema, since initdb creates it


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
-- Name: generate_ulid(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.generate_ulid() RETURNS uuid
    LANGUAGE sql
    AS $$
  SELECT (lpad(to_hex(floor(extract(epoch FROM clock_timestamp()) * 1000)::bigint), 12, '0') || encode(gen_random_bytes(10), 'hex'))::uuid;
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
-- Name: edit_suggestion_comments; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.edit_suggestion_comments (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    space_id uuid NOT NULL,
    edit_suggestion_id uuid NOT NULL,
    created_space_member_id uuid NOT NULL,
    body text NOT NULL,
    body_html text NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: edit_suggestion_page_revisions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.edit_suggestion_page_revisions (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    space_id uuid NOT NULL,
    edit_suggestion_page_id uuid NOT NULL,
    editor_space_member_id uuid NOT NULL,
    title public.citext NOT NULL,
    body public.citext NOT NULL,
    body_html text NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: edit_suggestion_pages; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.edit_suggestion_pages (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    space_id uuid NOT NULL,
    edit_suggestion_id uuid NOT NULL,
    page_id uuid,
    page_revision_id uuid,
    latest_revision_id uuid,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: edit_suggestions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.edit_suggestions (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    space_id uuid NOT NULL,
    topic_id uuid NOT NULL,
    created_space_member_id uuid NOT NULL,
    title character varying NOT NULL,
    description text NOT NULL,
    status integer DEFAULT 0 NOT NULL,
    applied_at timestamp(6) without time zone,
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
-- Name: draft_pages draft_pages_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_pages
    ADD CONSTRAINT draft_pages_pkey PRIMARY KEY (id);


--
-- Name: edit_suggestion_comments edit_suggestion_comments_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.edit_suggestion_comments
    ADD CONSTRAINT edit_suggestion_comments_pkey PRIMARY KEY (id);


--
-- Name: edit_suggestion_page_revisions edit_suggestion_page_revisions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.edit_suggestion_page_revisions
    ADD CONSTRAINT edit_suggestion_page_revisions_pkey PRIMARY KEY (id);


--
-- Name: edit_suggestion_pages edit_suggestion_pages_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.edit_suggestion_pages
    ADD CONSTRAINT edit_suggestion_pages_pkey PRIMARY KEY (id);


--
-- Name: edit_suggestions edit_suggestions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.edit_suggestions
    ADD CONSTRAINT edit_suggestions_pkey PRIMARY KEY (id);


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
-- Name: idx_on_edit_suggestion_page_id_b2797ca40b; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_on_edit_suggestion_page_id_b2797ca40b ON public.edit_suggestion_page_revisions USING btree (edit_suggestion_page_id);


--
-- Name: idx_on_edit_suggestion_page_id_created_at_155e14c0bd; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_on_edit_suggestion_page_id_created_at_155e14c0bd ON public.edit_suggestion_page_revisions USING btree (edit_suggestion_page_id, created_at);


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
-- Name: index_edit_suggestion_comments_on_created_space_member_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_edit_suggestion_comments_on_created_space_member_id ON public.edit_suggestion_comments USING btree (created_space_member_id);


--
-- Name: index_edit_suggestion_comments_on_edit_suggestion_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_edit_suggestion_comments_on_edit_suggestion_id ON public.edit_suggestion_comments USING btree (edit_suggestion_id);


--
-- Name: index_edit_suggestion_comments_on_space_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_edit_suggestion_comments_on_space_id ON public.edit_suggestion_comments USING btree (space_id);


--
-- Name: index_edit_suggestion_page_revisions_on_editor_space_member_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_edit_suggestion_page_revisions_on_editor_space_member_id ON public.edit_suggestion_page_revisions USING btree (editor_space_member_id);


--
-- Name: index_edit_suggestion_page_revisions_on_space_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_edit_suggestion_page_revisions_on_space_id ON public.edit_suggestion_page_revisions USING btree (space_id);


--
-- Name: index_edit_suggestion_pages_on_edit_suggestion_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_edit_suggestion_pages_on_edit_suggestion_id ON public.edit_suggestion_pages USING btree (edit_suggestion_id);


--
-- Name: index_edit_suggestion_pages_on_edit_suggestion_id_and_page_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_edit_suggestion_pages_on_edit_suggestion_id_and_page_id ON public.edit_suggestion_pages USING btree (edit_suggestion_id, page_id);


--
-- Name: index_edit_suggestion_pages_on_page_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_edit_suggestion_pages_on_page_id ON public.edit_suggestion_pages USING btree (page_id);


--
-- Name: index_edit_suggestion_pages_on_page_revision_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_edit_suggestion_pages_on_page_revision_id ON public.edit_suggestion_pages USING btree (page_revision_id);


--
-- Name: index_edit_suggestion_pages_on_space_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_edit_suggestion_pages_on_space_id ON public.edit_suggestion_pages USING btree (space_id);


--
-- Name: index_edit_suggestions_on_created_space_member_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_edit_suggestions_on_created_space_member_id ON public.edit_suggestions USING btree (created_space_member_id);


--
-- Name: index_edit_suggestions_on_space_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_edit_suggestions_on_space_id ON public.edit_suggestions USING btree (space_id);


--
-- Name: index_edit_suggestions_on_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_edit_suggestions_on_status ON public.edit_suggestions USING btree (status);


--
-- Name: index_edit_suggestions_on_topic_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_edit_suggestions_on_topic_id ON public.edit_suggestions USING btree (topic_id);


--
-- Name: index_edit_suggestions_on_topic_id_and_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_edit_suggestions_on_topic_id_and_status ON public.edit_suggestions USING btree (topic_id, status);


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
-- Name: edit_suggestion_pages fk_rails_3401ded95d; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.edit_suggestion_pages
    ADD CONSTRAINT fk_rails_3401ded95d FOREIGN KEY (page_revision_id) REFERENCES public.page_revisions(id);


--
-- Name: page_editors fk_rails_3b3700fcdf; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.page_editors
    ADD CONSTRAINT fk_rails_3b3700fcdf FOREIGN KEY (space_id) REFERENCES public.spaces(id);


--
-- Name: edit_suggestion_pages fk_rails_4226ebc7a6; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.edit_suggestion_pages
    ADD CONSTRAINT fk_rails_4226ebc7a6 FOREIGN KEY (latest_revision_id) REFERENCES public.edit_suggestion_page_revisions(id);


--
-- Name: edit_suggestion_pages fk_rails_42ff6f1c8f; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.edit_suggestion_pages
    ADD CONSTRAINT fk_rails_42ff6f1c8f FOREIGN KEY (edit_suggestion_id) REFERENCES public.edit_suggestions(id);


--
-- Name: edit_suggestion_pages fk_rails_4eb100714a; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.edit_suggestion_pages
    ADD CONSTRAINT fk_rails_4eb100714a FOREIGN KEY (space_id) REFERENCES public.spaces(id);


--
-- Name: topic_members fk_rails_50149efe6b; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.topic_members
    ADD CONSTRAINT fk_rails_50149efe6b FOREIGN KEY (topic_id) REFERENCES public.topics(id);


--
-- Name: edit_suggestion_page_revisions fk_rails_532e04880c; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.edit_suggestion_page_revisions
    ADD CONSTRAINT fk_rails_532e04880c FOREIGN KEY (space_id) REFERENCES public.spaces(id);


--
-- Name: edit_suggestion_page_revisions fk_rails_5814416d56; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.edit_suggestion_page_revisions
    ADD CONSTRAINT fk_rails_5814416d56 FOREIGN KEY (editor_space_member_id) REFERENCES public.space_members(id);


--
-- Name: pages fk_rails_5d9ff2d9dc; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pages
    ADD CONSTRAINT fk_rails_5d9ff2d9dc FOREIGN KEY (space_id) REFERENCES public.spaces(id);


--
-- Name: edit_suggestions fk_rails_644fb6c6f3; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.edit_suggestions
    ADD CONSTRAINT fk_rails_644fb6c6f3 FOREIGN KEY (created_space_member_id) REFERENCES public.space_members(id);


--
-- Name: edit_suggestion_comments fk_rails_67634e47e4; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.edit_suggestion_comments
    ADD CONSTRAINT fk_rails_67634e47e4 FOREIGN KEY (space_id) REFERENCES public.spaces(id);


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
-- Name: edit_suggestions fk_rails_98b4a981b3; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.edit_suggestions
    ADD CONSTRAINT fk_rails_98b4a981b3 FOREIGN KEY (topic_id) REFERENCES public.topics(id);


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
-- Name: edit_suggestion_page_revisions fk_rails_a82154c5a6; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.edit_suggestion_page_revisions
    ADD CONSTRAINT fk_rails_a82154c5a6 FOREIGN KEY (edit_suggestion_page_id) REFERENCES public.edit_suggestion_pages(id);


--
-- Name: export_statuses fk_rails_a8d9f2050b; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.export_statuses
    ADD CONSTRAINT fk_rails_a8d9f2050b FOREIGN KEY (export_id) REFERENCES public.exports(id);


--
-- Name: edit_suggestion_comments fk_rails_a97c7adbe3; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.edit_suggestion_comments
    ADD CONSTRAINT fk_rails_a97c7adbe3 FOREIGN KEY (created_space_member_id) REFERENCES public.space_members(id);


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
-- Name: edit_suggestion_comments fk_rails_bb1812de3e; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.edit_suggestion_comments
    ADD CONSTRAINT fk_rails_bb1812de3e FOREIGN KEY (edit_suggestion_id) REFERENCES public.edit_suggestions(id);


--
-- Name: active_storage_attachments fk_rails_c3b3935057; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.active_storage_attachments
    ADD CONSTRAINT fk_rails_c3b3935057 FOREIGN KEY (blob_id) REFERENCES public.active_storage_blobs(id);


--
-- Name: edit_suggestions fk_rails_c3e94da10b; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.edit_suggestions
    ADD CONSTRAINT fk_rails_c3e94da10b FOREIGN KEY (space_id) REFERENCES public.spaces(id);


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
-- Name: edit_suggestion_pages fk_rails_d3f13c3de7; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.edit_suggestion_pages
    ADD CONSTRAINT fk_rails_d3f13c3de7 FOREIGN KEY (page_id) REFERENCES public.pages(id);


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
-- PostgreSQL database dump complete
--

SET search_path TO "$user", public;

INSERT INTO "schema_migrations" (version) VALUES
('20250920082216'),
('20250920074019'),
('20250918173551'),
('20250918173550'),
('20250918173549'),
('20250918173528'),
('20250918173527'),
('20250918173454'),
('20250830075516'),
('20250830075345'),
('20250802185227'),
('20250802185226'),
('20250730164550'),
('20250730164526'),
('20250526173629'),
('20250517113235'),
('20250517113234'),
('20250517113233'),
('20250323084054'),
('20250323083136'),
('20250317095826'),
('20250204164034'),
('20250202165519'),
('20250202165124'),
('20250202063341'),
('20250119163728'),
('20241220175102'),
('20241201145213'),
('20241104061506'),
('20240000000002'),
('20240000000001');

