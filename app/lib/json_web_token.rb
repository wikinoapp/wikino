# typed: true
# frozen_string_literal: true

class JsonWebToken
  def self.decode_id_token(id_token)
    @decode_id_token ||= JWT.decode(id_token, nil, true,
      algorithms: "RS256",
      iss: "https://#{ENV.fetch("NONOTO_AUTH0_DOMAIN")}/",
      verify_iss: true,
      aud: ENV.fetch("NONOTO_AUTH0_CLIENT_ID"),
      verify_aud: true) do |header|
      jwks_hash[header["kid"]]
    end
  end

  def self.jwks_hash
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
