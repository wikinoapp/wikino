# frozen_string_literal: true

Rails.application.routes.draw do
  devise_for :users, skip: %i(passwords registrations sessions)

  constraints format: "html" do
    devise_scope :user do
      match "/password",      via: :patch,  as: :password,          to: "passwords#update"
      match "/password",      via: :post,                           to: "passwords#create"
      match "/password/edit", via: :get,    as: :edit_password,     to: "passwords#edit"
      match "/password/new",  via: :get,    as: :new_password,      to: "passwords#new"
      match "/sign_in",       via: :get,    as: :new_user_session,  to: "sessions#new"
      match "/sign_in",       via: :get,    as: :sign_in,           to: "sessions#new"
      match "/sign_in",       via: :post,   as: :user_session,      to: "sessions#create"
      match "/sign_out",      via: :delete, as: :sign_out,          to: "sessions#destroy"
      match "/sign_up",       via: :get,    as: :sign_up,           to: "registrations#new"
      match "/sign_up",       via: :post,   as: :user_registration, to: "registrations#create"
    end

    match "/notes", via: :get, as: :note_list, to: "notes#index"
  end

  constraints(format: "json") do
    if Rails.env.development?
      match "/api/local/graphql", via: :post, as: :local_graphql_api, to: "api/local/graphql#execute"
    end
  end

  root "welcome#show"
end
