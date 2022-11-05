# typed: strict
# frozen_string_literal: true

Rails.application.routes.draw do
  # standard:disable Layout/ExtraSpacing
  match "/auth/callback",     via: :get, as: :auth_callback,     to: "auth/callback#call"
  match "/auth/failure",      via: :get, as: :auth_failure,      to: "auth/failure#call"
  match "/sign_out",          via: :get, as: :sign_out,          to: "sign_out/show#call"
  match "/sign_out/callback", via: :get, as: :sign_out_callback, to: "sign_out/callback/show#call"
  # standard:enable Layout/ExtraSpacing

  root "welcome/show#call"
end
