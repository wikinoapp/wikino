# frozen_string_literal: true

module Internal::Authenticatable
  extend ActiveSupport::Concern

  include ActionController::HttpAuthentication::Token::ControllerMethods

  included do
    before_action :authenticate_with_id_token
  end

  def current_user
    @current_user
  end

  private

  def authenticate_with_id_token
    @current_user = begin
      authenticate_with_http_token do |id_token|
        payload, _header = decode_id_token(id_token)
        auth0_user_id = payload["sub"]

        User.only_kept.where(auth0_user_id:).first_or_create!
      end
    rescue JWT::VerificationError, JWT::DecodeError
      nil
    end
  end

  def decode_id_token(id_token)
    @decode_id_token ||= JWT.decode(id_token, nil, true,
      algorithms: "RS256",
      iss: "https://#{ENV.fetch("NONOTO_AUTH0_DOMAIN")}/",
      verify_iss: true,
      aud: ENV.fetch("NONOTO_AUTH0_CLIENT_ID"),
      verify_aud: true) do |header|
      jwks_hash[header["kid"]]
    end
  end

  def jwks_hash
    @jwks_hash ||= begin
      jwks_raw = Net::HTTP.get URI(ENV.fetch("NONOTO_AUTH0_JSON_WEB_KEY_SET"))
      jwks_keys = Array(JSON.parse(jwks_raw)["keys"])
      jwks_keys.map do |k|
        [
          k["kid"],
          OpenSSL::X509::Certificate.new(
            Base64.decode64(k["x5c"].first)
          ).public_key
        ]
      end.to_h
    end
  end
end
