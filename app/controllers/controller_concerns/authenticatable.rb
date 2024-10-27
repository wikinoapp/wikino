# typed: strict
# frozen_string_literal: true

module ControllerConcerns
  module Authenticatable
    extend T::Sig
    extend ActiveSupport::Concern

    included do
      helper_method :signed_in?
    end

    sig(:final) { params(session: Session).returns(T::Boolean) }
    def sign_in(session)
      cookies.signed.permanent[Session::COOKIE_KEY] = {
        value: session.token,
        httponly: true,
        same_site: :lax,
        domain: ".#{Wikino.config.host}"
      }

      true
    end

    sig(:final) { returns(T::Boolean) }
    def sign_out
      DestroySessionUseCase.new.call(session_token: session_token.not_nil!)
      cookies.delete(Session::COOKIE_KEY)

      true
    end

    sig(:final) { returns(T::Boolean) }
    def signed_in?
      Current.user.present?
    end

    sig(:final) { void }
    def require_authentication
      (restore_session && check_space_identifier) || request_authentication
    end

    sig(:final) { returns(T.untyped) }
    def require_no_authentication
      if signed_in?
        flash[:notice] = t("messages.authentication.already_signed_in")
        redirect_to space_path(space_identifier: Current.space!.identifier)
      end
    end

    sig(:final) { returns(String) }
    def after_authentication_url
      session.delete(:return_to_after_authenticating) ||
        space_url(space_identifier: Current.space!.identifier)
    end

    sig(:final) { returns(T.nilable(String)) }
    def original_remote_ip
      request.env["HTTP_CF_CONNECTING_IP"] || request.remote_ip
    end

    sig(:final) { returns(T.nilable(String)) }
    private def session_token
      cookies.signed[Session::COOKIE_KEY]
    end

    sig(:final) { returns(T::Boolean) }
    private def restore_session
      return false unless session_token

      user = Session.find_by(token: session_token)&.user
      Current.user = user

      true
    end

    sig(:final) { returns(T::Boolean) }
    private def check_space_identifier
      Current.space.identifier == params[:space_identifier]
    end

    sig(:final) { void }
    private def request_authentication
      sign_out
      session[:return_to_after_authenticating] = request.url
      redirect_to sign_in_path
    end
  end
end
