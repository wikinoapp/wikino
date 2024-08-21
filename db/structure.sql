SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

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
-- Name: list_members; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.list_members (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    space_id uuid NOT NULL,
    list_id uuid NOT NULL,
    user_id uuid NOT NULL,
    role integer NOT NULL,
    joined_at timestamp without time zone NOT NULL,
    last_note_modified_at timestamp(6) without time zone
);


--
-- Name: lists; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.lists (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    space_id uuid NOT NULL,
    number integer NOT NULL,
    name character varying NOT NULL,
    description character varying DEFAULT ''::character varying NOT NULL,
    visibility integer NOT NULL,
    discarded_at timestamp without time zone
);


--
-- Name: note_editors; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.note_editors (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    space_id uuid NOT NULL,
    note_id uuid NOT NULL,
    user_id uuid NOT NULL,
    last_note_modified_at timestamp(6) without time zone NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: note_links; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.note_links (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    source_note_id uuid NOT NULL,
    target_note_id uuid NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: note_revisions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.note_revisions (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    space_id uuid NOT NULL,
    note_id uuid NOT NULL,
    editor_id uuid NOT NULL,
    title public.citext DEFAULT ''::public.citext NOT NULL,
    body public.citext DEFAULT ''::public.citext NOT NULL,
    body_html text DEFAULT ''::text NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: notes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.notes (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    space_id uuid NOT NULL,
    list_id uuid NOT NULL,
    author_id uuid NOT NULL,
    number integer NOT NULL,
    title public.citext DEFAULT ''::public.citext NOT NULL,
    body public.citext DEFAULT ''::public.citext NOT NULL,
    body_html text DEFAULT ''::text NOT NULL,
    modified_at timestamp(6) without time zone NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
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
    ip_address character varying DEFAULT ''::character varying NOT NULL,
    user_agent character varying DEFAULT ''::character varying NOT NULL,
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
    name character varying DEFAULT ''::character varying NOT NULL,
    description character varying DEFAULT ''::character varying NOT NULL,
    locale integer NOT NULL,
    time_zone character varying NOT NULL,
    sign_in_count integer DEFAULT 0 NOT NULL,
    current_signed_in_at timestamp without time zone,
    last_signed_in_at timestamp without time zone,
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
-- Name: email_confirmations email_confirmations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.email_confirmations
    ADD CONSTRAINT email_confirmations_pkey PRIMARY KEY (id);


--
-- Name: list_members list_members_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.list_members
    ADD CONSTRAINT list_members_pkey PRIMARY KEY (id);


--
-- Name: lists lists_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lists
    ADD CONSTRAINT lists_pkey PRIMARY KEY (id);


--
-- Name: note_editors note_editors_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.note_editors
    ADD CONSTRAINT note_editors_pkey PRIMARY KEY (id);


--
-- Name: note_links note_links_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.note_links
    ADD CONSTRAINT note_links_pkey PRIMARY KEY (id);


--
-- Name: note_revisions note_revisions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.note_revisions
    ADD CONSTRAINT note_revisions_pkey PRIMARY KEY (id);


--
-- Name: notes notes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notes
    ADD CONSTRAINT notes_pkey PRIMARY KEY (id);


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
-- Name: index_email_confirmations_on_email_and_code; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_email_confirmations_on_email_and_code ON public.email_confirmations USING btree (email, code);


--
-- Name: index_email_confirmations_on_started_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_email_confirmations_on_started_at ON public.email_confirmations USING btree (started_at);


--
-- Name: index_list_members_on_list_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_list_members_on_list_id ON public.list_members USING btree (list_id);


--
-- Name: index_list_members_on_list_id_and_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_list_members_on_list_id_and_user_id ON public.list_members USING btree (list_id, user_id);


--
-- Name: index_list_members_on_space_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_list_members_on_space_id ON public.list_members USING btree (space_id);


--
-- Name: index_list_members_on_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_list_members_on_user_id ON public.list_members USING btree (user_id);


--
-- Name: index_lists_on_discarded_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_lists_on_discarded_at ON public.lists USING btree (discarded_at);


--
-- Name: index_lists_on_space_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_lists_on_space_id ON public.lists USING btree (space_id);


--
-- Name: index_lists_on_space_id_and_name; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_lists_on_space_id_and_name ON public.lists USING btree (space_id, name);


--
-- Name: index_lists_on_space_id_and_number; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_lists_on_space_id_and_number ON public.lists USING btree (space_id, number);


--
-- Name: index_note_editors_on_note_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_note_editors_on_note_id ON public.note_editors USING btree (note_id);


--
-- Name: index_note_editors_on_note_id_and_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_note_editors_on_note_id_and_user_id ON public.note_editors USING btree (note_id, user_id);


--
-- Name: index_note_editors_on_space_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_note_editors_on_space_id ON public.note_editors USING btree (space_id);


--
-- Name: index_note_editors_on_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_note_editors_on_user_id ON public.note_editors USING btree (user_id);


--
-- Name: index_note_links_on_source_note_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_note_links_on_source_note_id ON public.note_links USING btree (source_note_id);


--
-- Name: index_note_links_on_source_note_id_and_target_note_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_note_links_on_source_note_id_and_target_note_id ON public.note_links USING btree (source_note_id, target_note_id);


--
-- Name: index_note_links_on_target_note_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_note_links_on_target_note_id ON public.note_links USING btree (target_note_id);


--
-- Name: index_note_revisions_on_editor_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_note_revisions_on_editor_id ON public.note_revisions USING btree (editor_id);


--
-- Name: index_note_revisions_on_note_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_note_revisions_on_note_id ON public.note_revisions USING btree (note_id);


--
-- Name: index_note_revisions_on_space_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_note_revisions_on_space_id ON public.note_revisions USING btree (space_id);


--
-- Name: index_notes_on_author_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_notes_on_author_id ON public.notes USING btree (author_id);


--
-- Name: index_notes_on_list_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_notes_on_list_id ON public.notes USING btree (list_id);


--
-- Name: index_notes_on_list_id_and_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_notes_on_list_id_and_created_at ON public.notes USING btree (list_id, created_at);


--
-- Name: index_notes_on_list_id_and_modified_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_notes_on_list_id_and_modified_at ON public.notes USING btree (list_id, modified_at);


--
-- Name: index_notes_on_list_id_and_title; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_notes_on_list_id_and_title ON public.notes USING btree (list_id, title);


--
-- Name: index_notes_on_space_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_notes_on_space_id ON public.notes USING btree (space_id);


--
-- Name: index_notes_on_space_id_and_number; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_notes_on_space_id_and_number ON public.notes USING btree (space_id, number);


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
-- Name: index_user_passwords_on_space_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_user_passwords_on_space_id ON public.user_passwords USING btree (space_id);


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
-- Name: note_revisions fk_rails_1369420e41; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.note_revisions
    ADD CONSTRAINT fk_rails_1369420e41 FOREIGN KEY (editor_id) REFERENCES public.users(id);


--
-- Name: note_links fk_rails_28b9ac6b4e; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.note_links
    ADD CONSTRAINT fk_rails_28b9ac6b4e FOREIGN KEY (target_note_id) REFERENCES public.notes(id);


--
-- Name: notes fk_rails_36c9deba43; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notes
    ADD CONSTRAINT fk_rails_36c9deba43 FOREIGN KEY (author_id) REFERENCES public.users(id);


--
-- Name: note_editors fk_rails_3aad465980; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.note_editors
    ADD CONSTRAINT fk_rails_3aad465980 FOREIGN KEY (note_id) REFERENCES public.notes(id);


--
-- Name: list_members fk_rails_43c6abed89; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.list_members
    ADD CONSTRAINT fk_rails_43c6abed89 FOREIGN KEY (list_id) REFERENCES public.lists(id);


--
-- Name: notes fk_rails_551143b4f2; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notes
    ADD CONSTRAINT fk_rails_551143b4f2 FOREIGN KEY (list_id) REFERENCES public.lists(id);


--
-- Name: notes fk_rails_551dbc7e34; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notes
    ADD CONSTRAINT fk_rails_551dbc7e34 FOREIGN KEY (space_id) REFERENCES public.spaces(id);


--
-- Name: note_editors fk_rails_697e19aba9; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.note_editors
    ADD CONSTRAINT fk_rails_697e19aba9 FOREIGN KEY (space_id) REFERENCES public.spaces(id);


--
-- Name: note_editors fk_rails_6a191dd9ab; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.note_editors
    ADD CONSTRAINT fk_rails_6a191dd9ab FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: sessions fk_rails_758836b4f0; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.sessions
    ADD CONSTRAINT fk_rails_758836b4f0 FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: users fk_rails_96e4c019e9; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT fk_rails_96e4c019e9 FOREIGN KEY (space_id) REFERENCES public.spaces(id);


--
-- Name: list_members fk_rails_9bb6ff5a7a; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.list_members
    ADD CONSTRAINT fk_rails_9bb6ff5a7a FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: lists fk_rails_c2b88abd0a; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lists
    ADD CONSTRAINT fk_rails_c2b88abd0a FOREIGN KEY (space_id) REFERENCES public.spaces(id);


--
-- Name: note_revisions fk_rails_c65b77d10b; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.note_revisions
    ADD CONSTRAINT fk_rails_c65b77d10b FOREIGN KEY (note_id) REFERENCES public.notes(id);


--
-- Name: note_revisions fk_rails_c76e6bbeed; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.note_revisions
    ADD CONSTRAINT fk_rails_c76e6bbeed FOREIGN KEY (space_id) REFERENCES public.spaces(id);


--
-- Name: user_passwords fk_rails_c7888e4144; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_passwords
    ADD CONSTRAINT fk_rails_c7888e4144 FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: note_links fk_rails_dfb5d04224; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.note_links
    ADD CONSTRAINT fk_rails_dfb5d04224 FOREIGN KEY (source_note_id) REFERENCES public.notes(id);


--
-- Name: list_members fk_rails_ef08b83c98; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.list_members
    ADD CONSTRAINT fk_rails_ef08b83c98 FOREIGN KEY (space_id) REFERENCES public.spaces(id);


--
-- PostgreSQL database dump complete
--

SET search_path TO "$user", public;

INSERT INTO "schema_migrations" (version) VALUES
('20240000000002'),
('20240000000001');

