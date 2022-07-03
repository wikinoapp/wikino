# typed: strict
# frozen_string_literal: true

module Internal
  module Authenticatable
    extend T::Sig
    extend ActiveSupport::Concern

    include ActionController::HttpAuthentication::Token::ControllerMethods

    included do
      before_action :authenticate_with_id_token
    end

    sig { returns(T.nilable(User)) }
    attr_reader :current_user

    private

    sig { returns(T.nilable(User)) }
    def authenticate_with_id_token
      @current_user = T.let(begin
        authenticate_with_http_token do |id_token|
          payload, _header = JsonWebToken.decode_id_token(id_token)
          auth0_user_id = T.must(payload)["sub"]

          User.only_kept.where(auth0_user_id:).first_or_create!
        end
      rescue JWT::VerificationError, JWT::DecodeError
        nil
      end, T.nilable(User))
    end
  end
end
