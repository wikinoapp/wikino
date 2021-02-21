# frozen_string_literal: true

Rails.application.routes.draw do
  scope module: :api do
    constraints(subdomain: "api") do
      namespace :internal do
        post :graphql, to: "graphql#execute"
      end
    end
  end

  constraints format: "html" do
    match "/api/internal/notes",  via: :get,    as: :internal_api_note_list, to: "api/internal/notes#index"
    match "/my/appearance",       via: :get,    as: :appearance,             to: "my/appearances#show"
    match "/new",                 via: :get,    as: :new_note,               to: "notes#new"
    match "/notes",               via: :get,    as: :note_list,              to: "notes#index"
    match "/notes/:note_id",      via: :delete, as: :note,                   to: "notes#destroy", note_id: UUID_FORMAT
    match "/notes/:note_id",      via: :get,                                 to: "notes#show",    note_id: UUID_FORMAT
  end

  constraints(format: "json") do
    match "/api/internal/notes/:note_id", via: :patch, to: "api/internal/notes#update"

    if Rails.env.development?
      match "/api/local/graphql", via: :post, as: :local_graphql_api, to: "api/local/graphql#execute"
    end
  end

  root "welcome#show"
end
