# typed: false
# frozen_string_literal: true

Rails.application.config.middleware.use OmniAuth::Builder do
  provider(
    :auth0,
    ENV.fetch("NONOTO_AUTH0_CLIENT_ID"),
    ENV.fetch("NONOTO_AUTH0_CLIENT_SECRET"),
    ENV.fetch("NONOTO_AUTH0_DOMAIN"),
    callback_path: "/auth/callback",
    authorize_params: {
      scope: "openid profile"
    }
  )
end
