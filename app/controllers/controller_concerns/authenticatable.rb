# typed: strict
# frozen_string_literal: true

module ControllerConcerns
  module Authenticatable
    extend T::Sig
    extend ActiveSupport::Concern

    sig(:final) { params(user_session: UserSession).returns(T::Boolean) }
    def sign_in(user_session)
      Current.viewer = user_session.user
      store_user_session_token(token: user_session.token)

      true
    end

    sig(:final) { returns(T::Boolean) }
    def sign_out
      return true unless user_session_token

      DestroySessionService.new.call(user_session_token: user_session_token.not_nil!)
      cookies.delete(UserSession::TOKENS_COOKIE_KEY)

      true
    end

    sig(:final) { void }
    def require_authentication
      restore_user_session || request_authentication
    end

    sig(:final) { returns(T.untyped) }
    def require_no_authentication
      restore_user_session

      if Current.viewer.signed_in?
        flash[:notice] = t("messages.authentication.already_signed_in")
        redirect_to home_path
      end
    end

    sig(:final) { returns(String) }
    def after_authentication_url
      session.delete(:return_to_after_authenticating) || home_url
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
      if user_session_token && (user_session = UserSessionRecord.find_by(token: user_session_token))
        sign_in(user_session)
      else
        Current.viewer = Visitor.new
        false
      end
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
