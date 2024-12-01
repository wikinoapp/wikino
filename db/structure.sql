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
-- Name: ar_internal_metadata; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.ar_internal_metadata (
    key character varying NOT NULL,
    value character varying,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: draft_pages; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.draft_pages (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    space_id uuid NOT NULL,
    page_id uuid NOT NULL,
    editor_id uuid NOT NULL,
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
-- Name: page_editorships; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.page_editorships (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    space_id uuid NOT NULL,
    page_id uuid NOT NULL,
    editor_id uuid NOT NULL,
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
    editor_id uuid NOT NULL,
    page_id uuid NOT NULL,
    body public.citext NOT NULL,
    body_html text NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
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
    pinned_at timestamp without time zone
);


--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.schema_migrations (
    version character varying NOT NULL
);


--
-- Name: sessions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.sessions (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    space_id uuid NOT NULL,
    user_id uuid NOT NULL,
    token character varying NOT NULL,
    ip_address character varying NOT NULL,
    user_agent character varying NOT NULL,
    signed_in_at timestamp(6) without time zone NOT NULL,
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
-- Name: topic_memberships; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.topic_memberships (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    space_id uuid NOT NULL,
    topic_id uuid NOT NULL,
    member_id uuid NOT NULL,
    role integer NOT NULL,
    joined_at timestamp without time zone NOT NULL,
    last_page_modified_at timestamp(6) without time zone
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
    discarded_at timestamp without time zone
);


--
-- Name: user_passwords; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_passwords (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    space_id uuid NOT NULL,
    user_id uuid NOT NULL,
    password_digest character varying NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    space_id uuid NOT NULL,
    email character varying NOT NULL,
    atname public.citext NOT NULL,
    role integer NOT NULL,
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
-- Name: ar_internal_metadata ar_internal_metadata_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ar_internal_metadata
    ADD CONSTRAINT ar_internal_metadata_pkey PRIMARY KEY (key);


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
-- Name: page_editorships page_editorships_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.page_editorships
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
-- Name: sessions sessions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sessions
    ADD CONSTRAINT sessions_pkey PRIMARY KEY (id);


--
-- Name: spaces spaces_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.spaces
    ADD CONSTRAINT spaces_pkey PRIMARY KEY (id);


--
-- Name: topic_memberships topic_memberships_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.topic_memberships
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
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: index_draft_pages_on_editor_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_draft_pages_on_editor_id ON public.draft_pages USING btree (editor_id);


--
-- Name: index_draft_pages_on_editor_id_and_page_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_draft_pages_on_editor_id_and_page_id ON public.draft_pages USING btree (editor_id, page_id);


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
-- Name: index_page_editorships_on_editor_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_page_editorships_on_editor_id ON public.page_editorships USING btree (editor_id);


--
-- Name: index_page_editorships_on_page_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_page_editorships_on_page_id ON public.page_editorships USING btree (page_id);


--
-- Name: index_page_editorships_on_page_id_and_editor_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_page_editorships_on_page_id_and_editor_id ON public.page_editorships USING btree (page_id, editor_id);


--
-- Name: index_page_editorships_on_space_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_page_editorships_on_space_id ON public.page_editorships USING btree (space_id);


--
-- Name: index_page_revisions_on_editor_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_page_revisions_on_editor_id ON public.page_revisions USING btree (editor_id);


--
-- Name: index_page_revisions_on_page_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_page_revisions_on_page_id ON public.page_revisions USING btree (page_id);


--
-- Name: index_page_revisions_on_space_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_page_revisions_on_space_id ON public.page_revisions USING btree (space_id);


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
-- Name: index_sessions_on_space_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_sessions_on_space_id ON public.sessions USING btree (space_id);


--
-- Name: index_sessions_on_token; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_sessions_on_token ON public.sessions USING btree (token);


--
-- Name: index_sessions_on_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_sessions_on_user_id ON public.sessions USING btree (user_id);


--
-- Name: index_spaces_on_discarded_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_spaces_on_discarded_at ON public.spaces USING btree (discarded_at);


--
-- Name: index_spaces_on_identifier; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_spaces_on_identifier ON public.spaces USING btree (identifier);


--
-- Name: index_topic_memberships_on_member_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_topic_memberships_on_member_id ON public.topic_memberships USING btree (member_id);


--
-- Name: index_topic_memberships_on_space_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_topic_memberships_on_space_id ON public.topic_memberships USING btree (space_id);


--
-- Name: index_topic_memberships_on_topic_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_topic_memberships_on_topic_id ON public.topic_memberships USING btree (topic_id);


--
-- Name: index_topic_memberships_on_topic_id_and_member_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_topic_memberships_on_topic_id_and_member_id ON public.topic_memberships USING btree (topic_id, member_id);


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
-- Name: index_user_passwords_on_space_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_user_passwords_on_space_id ON public.user_passwords USING btree (space_id);


--
-- Name: index_user_passwords_on_space_id_and_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_user_passwords_on_space_id_and_user_id ON public.user_passwords USING btree (space_id, user_id);


--
-- Name: index_user_passwords_on_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_user_passwords_on_user_id ON public.user_passwords USING btree (user_id);


--
-- Name: index_users_on_discarded_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_users_on_discarded_at ON public.users USING btree (discarded_at);


--
-- Name: index_users_on_space_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_users_on_space_id ON public.users USING btree (space_id);


--
-- Name: index_users_on_space_id_and_atname; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_users_on_space_id_and_atname ON public.users USING btree (space_id, atname);


--
-- Name: index_users_on_space_id_and_discarded_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_users_on_space_id_and_discarded_at ON public.users USING btree (space_id, discarded_at);


--
-- Name: index_users_on_space_id_and_email; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_users_on_space_id_and_email ON public.users USING btree (space_id, email);


--
-- Name: user_passwords fk_rails_081bbe105f; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_passwords
    ADD CONSTRAINT fk_rails_081bbe105f FOREIGN KEY (space_id) REFERENCES public.spaces(id);


--
-- Name: sessions fk_rails_0f63041bba; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sessions
    ADD CONSTRAINT fk_rails_0f63041bba FOREIGN KEY (space_id) REFERENCES public.spaces(id);


--
-- Name: topic_memberships fk_rails_0f8ef246f7; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.topic_memberships
    ADD CONSTRAINT fk_rails_0f8ef246f7 FOREIGN KEY (space_id) REFERENCES public.spaces(id);


--
-- Name: page_editorships fk_rails_2088082077; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.page_editorships
    ADD CONSTRAINT fk_rails_2088082077 FOREIGN KEY (editor_id) REFERENCES public.users(id);


--
-- Name: page_editorships fk_rails_3b3700fcdf; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.page_editorships
    ADD CONSTRAINT fk_rails_3b3700fcdf FOREIGN KEY (space_id) REFERENCES public.spaces(id);


--
-- Name: topic_memberships fk_rails_50149efe6b; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.topic_memberships
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
-- Name: page_revisions fk_rails_74648de0a3; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.page_revisions
    ADD CONSTRAINT fk_rails_74648de0a3 FOREIGN KEY (editor_id) REFERENCES public.users(id);


--
-- Name: sessions fk_rails_758836b4f0; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sessions
    ADD CONSTRAINT fk_rails_758836b4f0 FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: pages fk_rails_793c81c055; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.pages
    ADD CONSTRAINT fk_rails_793c81c055 FOREIGN KEY (topic_id) REFERENCES public.topics(id);


--
-- Name: topic_memberships fk_rails_80fd6512fa; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.topic_memberships
    ADD CONSTRAINT fk_rails_80fd6512fa FOREIGN KEY (member_id) REFERENCES public.users(id);


--
-- Name: draft_pages fk_rails_8d9bc1217e; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_pages
    ADD CONSTRAINT fk_rails_8d9bc1217e FOREIGN KEY (editor_id) REFERENCES public.users(id);


--
-- Name: draft_pages fk_rails_8e68719216; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.draft_pages
    ADD CONSTRAINT fk_rails_8e68719216 FOREIGN KEY (topic_id) REFERENCES public.topics(id);


--
-- Name: users fk_rails_96e4c019e9; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT fk_rails_96e4c019e9 FOREIGN KEY (space_id) REFERENCES public.spaces(id);


--
-- Name: topics fk_rails_9b3ff1bd6e; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.topics
    ADD CONSTRAINT fk_rails_9b3ff1bd6e FOREIGN KEY (space_id) REFERENCES public.spaces(id);


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
-- Name: user_passwords fk_rails_c7888e4144; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_passwords
    ADD CONSTRAINT fk_rails_c7888e4144 FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: page_revisions fk_rails_d3466381ba; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.page_revisions
    ADD CONSTRAINT fk_rails_d3466381ba FOREIGN KEY (space_id) REFERENCES public.spaces(id);


--
-- Name: page_editorships fk_rails_fc18bdf1dd; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.page_editorships
    ADD CONSTRAINT fk_rails_fc18bdf1dd FOREIGN KEY (page_id) REFERENCES public.pages(id);


--
-- PostgreSQL database dump complete
--

SET search_path TO "$user", public;

INSERT INTO "schema_migrations" (version) VALUES
('20241201145213'),
('20241104061506'),
('20240000000002'),
('20240000000001');

