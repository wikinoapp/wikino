# typed: false
# frozen_string_literal: true

Rails.application.routes.draw do
  if Rails.env.development?
    mount LetterOpenerWeb::Engine, at: "/letter_opener"
  end

  # standard:disable Layout/ExtraSpacing, Rails/MatchRoute
  match "/accounts",                               via: :post,   as: :accounts,                to: "accounts/create#call"
  match "/accounts/new",                           via: :get,    as: :new_account,             to: "accounts/new#call"
  match "/email_confirmation",                     via: :patch,  as: :email_confirmation,      to: "email_confirmations/update#call"
  match "/email_confirmation",                     via: :post,                                 to: "email_confirmations/create#call"
  match "/email_confirmation/edit",                via: :get,    as: :edit_email_confirmation, to: "email_confirmations/edit#call"
  match "/s/:space_identifier",                    via: :get,    as: :space,                   to: "spaces/show#call"
  match "/s/:space_identifier/lists",              via: :post,   as: :lists,                   to: "lists/create#call"
  match "/s/:space_identifier/lists/:list_number", via: :get,    as: :list,                    to: "lists/show#call",                list_number: /\d+/
  match "/s/:space_identifier/lists/new",          via: :get,    as: :new_list,                to: "lists/new#call"
  match "/s/:space_identifier/notes/new",          via: :get,    as: :new_note,                to: "notes/new#call"
  match "/s/:space_identifier/session",            via: :delete, as: :session,                 to: "sessions/destroy#call"
  match "/sessions",                               via: :post,   as: :sessions,                to: "sessions/create#call"
  match "/sign_in",                                via: :get,    as: :sign_in,                 to: "sign_in/show#call"
  match "/sign_up",                                via: :get,    as: :sign_up,                 to: "sign_up/show#call"
  # standard:enable Layout/ExtraSpacing, Rails/MatchRoute

  root "welcome/show#call"
end
