# typed: false
# frozen_string_literal: true

Rails.application.routes.draw do
  # standard:disable Layout/ExtraSpacing
  match "/notes",             via: :get, as: :note_list,         to: "notes/index#call"
  match "/notes/new",         via: :get, as: :new_note,          to: "notes/new#call"
  match "/notes/:note_id",    via: :get, as: :note,              to: "notes#show",                  note_id: UUID_FORMAT
  match "/sign_in/callback",  via: :get, as: :sign_in_callback,  to: "sign_in/callback#call"
  match "/sign_in/failure",   via: :get, as: :sign_in_failure,   to: "sign_in/failure#call"
  match "/sign_out",          via: :get, as: :sign_out,          to: "sign_out/show#call"
  match "/sign_out/callback", via: :get, as: :sign_out_callback, to: "sign_out/callback/show#call"
  # standard:enable Layout/ExtraSpacing

  root "welcome/show#call"
end
