# typed: strict
# frozen_string_literal: true

Rails.application.config.session_store :cookie_store,
  key: "_wikino_session",
  domain: :all,
  expire_after: 10.years
