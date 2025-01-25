# typed: strict
# frozen_string_literal: true

module ControllerConcerns
  module Authenticatable
    extend T::Sig
    extend ActiveSupport::Concern

    included do
      helper_method :signed_in?
    end

    sig(:final) { params(user_session: UserSession).returns(T::Boolean) }
    def sign_in(user_session)
      Current.user = user_session.user
      store_user_session_token(token: user_session.token)

      true
    end

    sig(:final) { returns(T::Boolean) }
    def sign_out
      return true unless session_tokens

      space_identifier = Current.space!.identifier
      session_token = session_tokens.not_nil![space_identifier]

      DestroySessionUseCase.new.call(session_token:) if session_token

      tokens = session_tokens.not_nil!.except(space_identifier)
      store_session_tokens_to_cookie(token_str: hash_to_string(tokens))

      true
    end

    sig(:final) { returns(T::Boolean) }
    def signed_in?
      Current.user.present?
    end

    sig(:final) { void }
    def require_authentication
      restore_user_session || request_authentication
    end

    sig(:final) { returns(T.untyped) }
    def require_no_authentication
      restore_user_session

      if signed_in?
        flash[:notice] = t("messages.authentication.already_signed_in")
        redirect_to root_path
      end
    end

    sig(:final) { returns(String) }
    def after_authentication_url
      session.delete(:return_to_after_authenticating) || root_url
    end

    sig(:final) { returns(T.nilable(String)) }
    def original_remote_ip
      request.env["HTTP_CF_CONNECTING_IP"] || request.remote_ip
    end

    sig(:final) { returns(T.nilable(String)) }
    private def user_session_token
      cookies.signed[UserSession::TOKENS_COOKIE_KEY]
    end

    sig(:final) { returns(T::Boolean) }
    private def restore_user_session
      return false unless user_session_token

      user_session = UserSession.find_by(token: user_session_token)
      return false unless user_session

      sign_in(user_session)

      true
    end

    sig(:final) { void }
    private def request_authentication
      sign_out
      session[:return_to_after_authenticating] = request.url
      redirect_to sign_in_path
    end

    sig(:final) { params(token: String).void }
    private def store_user_session_token(token:)
      cookies.signed.permanent[UserSession::TOKENS_COOKIE_KEY] = {
        value: token,
        httponly: true,
        same_site: :lax,
        domain: ".#{Wikino.config.host}"
      }
    end
  end
end
