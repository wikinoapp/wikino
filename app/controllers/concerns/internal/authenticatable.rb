# typed: false
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
        payload, _header = JsonWebToken.decode_id_token(id_token)
        auth0_user_id = payload["sub"]

        User.only_kept.where(auth0_user_id:).first_or_create!
      end
    rescue JWT::VerificationError, JWT::DecodeError
      nil
    end
  end
end
