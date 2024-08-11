# typed: false
# frozen_string_literal: true

Rails.application.routes.draw do
  if Rails.env.development?
    mount LetterOpenerWeb::Engine, at: "/letter_opener"
  end

  constraints(Nonoto::RoutingConstraint::SpaceSubdomain.new) do
    # standard:disable Layout/ExtraSpacing, Rails/MatchRoute
    match "/notes/new", via: :get,    as: :new_note, to: "notes/new#call"
    match "/sign_out",  via: :delete, as: :sign_out, to: "sessions/destroy#call"
    # standard:enable Layout/ExtraSpacing, Rails/MatchRoute

    root "spaces/show#call", as: :space_root
  end

  # standard:disable Layout/ExtraSpacing, Rails/MatchRoute
  match "/accounts",                via: :post,  as: :account_list,            to: "accounts/create#call"
  match "/accounts/new",            via: :get,   as: :new_account,             to: "accounts/new#call"
  match "/email_confirmation",      via: :patch, as: :email_confirmation,      to: "email_confirmations/update#call"
  match "/email_confirmation/edit", via: :get,   as: :edit_email_confirmation, to: "email_confirmations/edit#call"
  match "/sign_in",                 via: :get,   as: :sign_in,                 to: "sessions/new#call"
  match "/sign_up",                 via: :get,   as: :sign_up,                 to: "sign_up/new#call"
  match "/sign_up",                 via: :post,                                to: "sign_up/create#call"
  # standard:enable Layout/ExtraSpacing, Rails/MatchRoute

  root "welcome/show#call"
end
