# typed: strict
# frozen_string_literal: true

Rails.application.routes.draw do
  # standard:disable Layout/ExtraSpacing
  match "/auth/callback", via: :get, as: :auth_callback, to: "auth/callback#call"
  match "/auth/failure",  via: :get, as: :auth_failure,  to: "auth/failure#call"
  # standard:enable Layout/ExtraSpacing

  root "welcome/show#call"
end
