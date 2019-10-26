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


SET default_tablespace = '';

SET default_with_oids = false;

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
-- Name: edges; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.edges (
    id uuid DEFAULT public.gen_random_uuid() NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL,
    note_id uuid NOT NULL,
    target_note_id uuid NOT NULL
);


--
-- Name: notes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.notes (
    id uuid DEFAULT public.gen_random_uuid() NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL,
    team_id uuid NOT NULL,
    project_id uuid NOT NULL,
    creator_id uuid NOT NULL,
    number integer NOT NULL,
    body text
);


--
-- Name: oauth_providers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.oauth_providers (
    id uuid DEFAULT public.gen_random_uuid() NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL,
    user_id uuid NOT NULL,
    name integer NOT NULL,
    uid character varying NOT NULL,
    token character varying NOT NULL,
    token_expires_at integer
);


--
-- Name: participants; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.participants (
    id uuid DEFAULT public.gen_random_uuid() NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL,
    note_id uuid NOT NULL,
    user_id uuid NOT NULL
);


--
-- Name: project_members; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.project_members (
    id uuid DEFAULT public.gen_random_uuid() NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL,
    team_id uuid NOT NULL,
    user_id uuid NOT NULL,
    project_id uuid NOT NULL,
    team_member_id uuid NOT NULL
);


--
-- Name: projects; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.projects (
    id uuid DEFAULT public.gen_random_uuid() NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL,
    deleted_at timestamp without time zone,
    team_id uuid NOT NULL,
    urlname character varying NOT NULL,
    name character varying NOT NULL
);


--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.schema_migrations (
    version character varying NOT NULL
);


--
-- Name: taggings; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.taggings (
    id uuid DEFAULT public.gen_random_uuid() NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL,
    team_id uuid NOT NULL,
    project_id uuid NOT NULL,
    note_id uuid NOT NULL,
    tag_id uuid NOT NULL
);


--
-- Name: tags; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.tags (
    id uuid DEFAULT public.gen_random_uuid() NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL,
    team_id uuid NOT NULL,
    project_id uuid NOT NULL,
    name public.citext NOT NULL
);


--
-- Name: team_members; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.team_members (
    id uuid DEFAULT public.gen_random_uuid() NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL,
    deleted_at timestamp without time zone,
    team_id uuid NOT NULL,
    user_id uuid NOT NULL,
    username public.citext NOT NULL,
    name character varying
);


--
-- Name: teams; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.teams (
    id uuid DEFAULT public.gen_random_uuid() NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL,
    deleted_at timestamp without time zone,
    subdomain public.citext NOT NULL,
    name character varying NOT NULL
);


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    id uuid DEFAULT public.gen_random_uuid() NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL,
    deleted_at timestamp without time zone,
    email public.citext NOT NULL
);


--
-- Name: ar_internal_metadata ar_internal_metadata_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ar_internal_metadata
    ADD CONSTRAINT ar_internal_metadata_pkey PRIMARY KEY (key);


--
-- Name: edges edges_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.edges
    ADD CONSTRAINT edges_pkey PRIMARY KEY (id);


--
-- Name: notes notes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notes
    ADD CONSTRAINT notes_pkey PRIMARY KEY (id);


--
-- Name: oauth_providers oauth_providers_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.oauth_providers
    ADD CONSTRAINT oauth_providers_pkey PRIMARY KEY (id);


--
-- Name: participants participants_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.participants
    ADD CONSTRAINT participants_pkey PRIMARY KEY (id);


--
-- Name: project_members project_members_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_members
    ADD CONSTRAINT project_members_pkey PRIMARY KEY (id);


--
-- Name: projects projects_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.projects
    ADD CONSTRAINT projects_pkey PRIMARY KEY (id);


--
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);


--
-- Name: taggings taggings_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.taggings
    ADD CONSTRAINT taggings_pkey PRIMARY KEY (id);


--
-- Name: tags tags_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tags
    ADD CONSTRAINT tags_pkey PRIMARY KEY (id);


--
-- Name: team_members team_members_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.team_members
    ADD CONSTRAINT team_members_pkey PRIMARY KEY (id);


--
-- Name: teams teams_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.teams
    ADD CONSTRAINT teams_pkey PRIMARY KEY (id);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: index_edges_on_note_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_edges_on_note_id ON public.edges USING btree (note_id);


--
-- Name: index_edges_on_note_id_and_target_note_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_edges_on_note_id_and_target_note_id ON public.edges USING btree (note_id, target_note_id);


--
-- Name: index_edges_on_target_note_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_edges_on_target_note_id ON public.edges USING btree (target_note_id);


--
-- Name: index_notes_on_creator_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_notes_on_creator_id ON public.notes USING btree (creator_id);


--
-- Name: index_notes_on_project_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_notes_on_project_id ON public.notes USING btree (project_id);


--
-- Name: index_notes_on_team_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_notes_on_team_id ON public.notes USING btree (team_id);


--
-- Name: index_notes_on_team_id_and_number; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_notes_on_team_id_and_number ON public.notes USING btree (team_id, number);


--
-- Name: index_oauth_providers_on_name_and_uid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_oauth_providers_on_name_and_uid ON public.oauth_providers USING btree (name, uid);


--
-- Name: index_oauth_providers_on_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_oauth_providers_on_user_id ON public.oauth_providers USING btree (user_id);


--
-- Name: index_participants_on_note_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_participants_on_note_id ON public.participants USING btree (note_id);


--
-- Name: index_participants_on_note_id_and_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_participants_on_note_id_and_user_id ON public.participants USING btree (note_id, user_id);


--
-- Name: index_participants_on_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_participants_on_user_id ON public.participants USING btree (user_id);


--
-- Name: index_project_members_on_project_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_project_members_on_project_id ON public.project_members USING btree (project_id);


--
-- Name: index_project_members_on_project_id_and_team_member_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_project_members_on_project_id_and_team_member_id ON public.project_members USING btree (project_id, team_member_id);


--
-- Name: index_project_members_on_team_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_project_members_on_team_id ON public.project_members USING btree (team_id);


--
-- Name: index_project_members_on_team_member_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_project_members_on_team_member_id ON public.project_members USING btree (team_member_id);


--
-- Name: index_project_members_on_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_project_members_on_user_id ON public.project_members USING btree (user_id);


--
-- Name: index_projects_on_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_projects_on_deleted_at ON public.projects USING btree (deleted_at);


--
-- Name: index_projects_on_team_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_projects_on_team_id ON public.projects USING btree (team_id);


--
-- Name: index_projects_on_urlname; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_projects_on_urlname ON public.projects USING btree (urlname);


--
-- Name: index_taggings_on_note_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_taggings_on_note_id ON public.taggings USING btree (note_id);


--
-- Name: index_taggings_on_note_id_and_tag_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_taggings_on_note_id_and_tag_id ON public.taggings USING btree (note_id, tag_id);


--
-- Name: index_taggings_on_project_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_taggings_on_project_id ON public.taggings USING btree (project_id);


--
-- Name: index_taggings_on_tag_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_taggings_on_tag_id ON public.taggings USING btree (tag_id);


--
-- Name: index_taggings_on_team_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_taggings_on_team_id ON public.taggings USING btree (team_id);


--
-- Name: index_tags_on_project_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_tags_on_project_id ON public.tags USING btree (project_id);


--
-- Name: index_tags_on_project_id_and_name; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_tags_on_project_id_and_name ON public.tags USING btree (project_id, name);


--
-- Name: index_tags_on_team_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_tags_on_team_id ON public.tags USING btree (team_id);


--
-- Name: index_team_members_on_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_team_members_on_deleted_at ON public.team_members USING btree (deleted_at);


--
-- Name: index_team_members_on_team_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_team_members_on_team_id ON public.team_members USING btree (team_id);


--
-- Name: index_team_members_on_team_id_and_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_team_members_on_team_id_and_user_id ON public.team_members USING btree (team_id, user_id);


--
-- Name: index_team_members_on_team_id_and_username; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_team_members_on_team_id_and_username ON public.team_members USING btree (team_id, username);


--
-- Name: index_team_members_on_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_team_members_on_user_id ON public.team_members USING btree (user_id);


--
-- Name: index_teams_on_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_teams_on_deleted_at ON public.teams USING btree (deleted_at);


--
-- Name: index_teams_on_subdomain; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_teams_on_subdomain ON public.teams USING btree (subdomain);


--
-- Name: index_users_on_deleted_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_users_on_deleted_at ON public.users USING btree (deleted_at);


--
-- Name: index_users_on_email; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_users_on_email ON public.users USING btree (email);


--
-- Name: notes fk_rails_024ba0b8ac; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notes
    ADD CONSTRAINT fk_rails_024ba0b8ac FOREIGN KEY (team_id) REFERENCES public.teams(id);


--
-- Name: edges fk_rails_18cc985ef2; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.edges
    ADD CONSTRAINT fk_rails_18cc985ef2 FOREIGN KEY (target_note_id) REFERENCES public.notes(id);


--
-- Name: team_members fk_rails_194b5b076d; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.team_members
    ADD CONSTRAINT fk_rails_194b5b076d FOREIGN KEY (team_id) REFERENCES public.teams(id);


--
-- Name: tags fk_rails_2f90b9163e; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tags
    ADD CONSTRAINT fk_rails_2f90b9163e FOREIGN KEY (project_id) REFERENCES public.projects(id);


--
-- Name: project_members fk_rails_49ebe01c9d; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_members
    ADD CONSTRAINT fk_rails_49ebe01c9d FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: oauth_providers fk_rails_5bebf45322; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.oauth_providers
    ADD CONSTRAINT fk_rails_5bebf45322 FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: notes fk_rails_5d4a723a34; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notes
    ADD CONSTRAINT fk_rails_5d4a723a34 FOREIGN KEY (creator_id) REFERENCES public.users(id);


--
-- Name: taggings fk_rails_6cd618bbbb; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.taggings
    ADD CONSTRAINT fk_rails_6cd618bbbb FOREIGN KEY (team_id) REFERENCES public.teams(id);


--
-- Name: edges fk_rails_7b04f04961; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.edges
    ADD CONSTRAINT fk_rails_7b04f04961 FOREIGN KEY (note_id) REFERENCES public.notes(id);


--
-- Name: notes fk_rails_99e097b079; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notes
    ADD CONSTRAINT fk_rails_99e097b079 FOREIGN KEY (project_id) REFERENCES public.projects(id);


--
-- Name: team_members fk_rails_9ec2d5e75e; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.team_members
    ADD CONSTRAINT fk_rails_9ec2d5e75e FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: taggings fk_rails_9fcd2e236b; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.taggings
    ADD CONSTRAINT fk_rails_9fcd2e236b FOREIGN KEY (tag_id) REFERENCES public.tags(id);


--
-- Name: project_members fk_rails_a9658c42a3; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_members
    ADD CONSTRAINT fk_rails_a9658c42a3 FOREIGN KEY (team_member_id) REFERENCES public.team_members(id);


--
-- Name: taggings fk_rails_a9b34328d4; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.taggings
    ADD CONSTRAINT fk_rails_a9b34328d4 FOREIGN KEY (note_id) REFERENCES public.notes(id);


--
-- Name: participants fk_rails_b9a3c50f15; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.participants
    ADD CONSTRAINT fk_rails_b9a3c50f15 FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: project_members fk_rails_d23d6decaf; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_members
    ADD CONSTRAINT fk_rails_d23d6decaf FOREIGN KEY (team_id) REFERENCES public.teams(id);


--
-- Name: taggings fk_rails_e1d638f7e4; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.taggings
    ADD CONSTRAINT fk_rails_e1d638f7e4 FOREIGN KEY (project_id) REFERENCES public.projects(id);


--
-- Name: tags fk_rails_e39f546aa9; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tags
    ADD CONSTRAINT fk_rails_e39f546aa9 FOREIGN KEY (team_id) REFERENCES public.teams(id);


--
-- Name: participants fk_rails_e9195a9785; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.participants
    ADD CONSTRAINT fk_rails_e9195a9785 FOREIGN KEY (note_id) REFERENCES public.notes(id);


--
-- Name: projects fk_rails_ecc227a0c2; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.projects
    ADD CONSTRAINT fk_rails_ecc227a0c2 FOREIGN KEY (team_id) REFERENCES public.teams(id);


--
-- Name: project_members fk_rails_f3b43b5269; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.project_members
    ADD CONSTRAINT fk_rails_f3b43b5269 FOREIGN KEY (project_id) REFERENCES public.projects(id);


--
-- PostgreSQL database dump complete
--

SET search_path TO "$user", public;

INSERT INTO "schema_migrations" (version) VALUES
('0');


