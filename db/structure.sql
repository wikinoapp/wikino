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
-- Name: pgcrypto; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;


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
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    email character varying NOT NULL,
    original_email character varying NOT NULL,
    event integer NOT NULL,
    token character varying NOT NULL,
    expires_at timestamp(6) without time zone NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: links; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.links (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    note_id uuid NOT NULL,
    target_note_id uuid NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: note_contents; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.note_contents (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    note_id uuid NOT NULL,
    body public.citext DEFAULT ''::public.citext NOT NULL,
    body_html text DEFAULT ''::text NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: notes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.notes (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    title public.citext DEFAULT ''::public.citext NOT NULL,
    content_type character varying NOT NULL,
    content_id uuid NOT NULL,
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
-- Name: stacked_note_contents; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.stacked_note_contents (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    note_id uuid NOT NULL,
    body public.citext DEFAULT ''::public.citext NOT NULL,
    body_html text DEFAULT ''::text NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    email character varying NOT NULL,
    original_email character varying NOT NULL,
    sign_in_count integer DEFAULT 0 NOT NULL,
    signed_up_at timestamp(6) without time zone NOT NULL,
    current_sign_in_at timestamp(6) without time zone,
    last_sign_in_at timestamp(6) without time zone,
    deleted_at timestamp(6) without time zone,
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
-- Name: links links_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.links
    ADD CONSTRAINT links_pkey PRIMARY KEY (id);


--
-- Name: note_contents note_contents_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.note_contents
    ADD CONSTRAINT note_contents_pkey PRIMARY KEY (id);


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
-- Name: stacked_note_contents stacked_note_contents_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stacked_note_contents
    ADD CONSTRAINT stacked_note_contents_pkey PRIMARY KEY (id);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: index_email_confirmations_on_token; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_email_confirmations_on_token ON public.email_confirmations USING btree (token);


--
-- Name: index_links_on_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_links_on_created_at ON public.links USING btree (created_at);


--
-- Name: index_links_on_note_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_links_on_note_id ON public.links USING btree (note_id);


--
-- Name: index_links_on_note_id_and_target_note_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_links_on_note_id_and_target_note_id ON public.links USING btree (note_id, target_note_id);


--
-- Name: index_links_on_target_note_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_links_on_target_note_id ON public.links USING btree (target_note_id);


--
-- Name: index_note_contents_on_note_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_note_contents_on_note_id ON public.note_contents USING btree (note_id);


--
-- Name: index_note_contents_on_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_note_contents_on_user_id ON public.note_contents USING btree (user_id);


--
-- Name: index_note_contents_on_user_id_and_note_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_note_contents_on_user_id_and_note_id ON public.note_contents USING btree (user_id, note_id);


--
-- Name: index_notes_on_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_notes_on_created_at ON public.notes USING btree (created_at);


--
-- Name: index_notes_on_modified_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_notes_on_modified_at ON public.notes USING btree (modified_at);


--
-- Name: index_notes_on_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_notes_on_user_id ON public.notes USING btree (user_id);


--
-- Name: index_notes_on_user_id_and_title; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_notes_on_user_id_and_title ON public.notes USING btree (user_id, title);


--
-- Name: index_stacked_note_contents_on_note_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_stacked_note_contents_on_note_id ON public.stacked_note_contents USING btree (note_id);


--
-- Name: index_stacked_note_contents_on_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_stacked_note_contents_on_user_id ON public.stacked_note_contents USING btree (user_id);


--
-- Name: index_stacked_note_contents_on_user_id_and_note_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_stacked_note_contents_on_user_id_and_note_id ON public.stacked_note_contents USING btree (user_id, note_id);


--
-- Name: index_users_on_email; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_users_on_email ON public.users USING btree (email);


--
-- Name: note_contents fk_rails_28bd55fa9f; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.note_contents
    ADD CONSTRAINT fk_rails_28bd55fa9f FOREIGN KEY (note_id) REFERENCES public.notes(id);


--
-- Name: stacked_note_contents fk_rails_3af95f6ef1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stacked_note_contents
    ADD CONSTRAINT fk_rails_3af95f6ef1 FOREIGN KEY (note_id) REFERENCES public.notes(id);


--
-- Name: links fk_rails_520010c2f8; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.links
    ADD CONSTRAINT fk_rails_520010c2f8 FOREIGN KEY (target_note_id) REFERENCES public.notes(id);


--
-- Name: note_contents fk_rails_67e211fca5; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.note_contents
    ADD CONSTRAINT fk_rails_67e211fca5 FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: notes fk_rails_7f2323ad43; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notes
    ADD CONSTRAINT fk_rails_7f2323ad43 FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: stacked_note_contents fk_rails_ca3dbfe673; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.stacked_note_contents
    ADD CONSTRAINT fk_rails_ca3dbfe673 FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: links fk_rails_f26877463b; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.links
    ADD CONSTRAINT fk_rails_f26877463b FOREIGN KEY (note_id) REFERENCES public.notes(id);


--
-- PostgreSQL database dump complete
--

SET search_path TO "$user", public;

INSERT INTO "schema_migrations" (version) VALUES
('20220505022747'),
('20220505022901');


