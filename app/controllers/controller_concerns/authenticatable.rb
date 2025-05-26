# typed: strict
# frozen_string_literal: true

module ControllerConcerns
  module Authenticatable
    extend T::Sig
    extend ActiveSupport::Concern

    sig(:final) { params(user_session_record: UserSessionRecord).returns(T::Boolean) }
    def sign_in(user_session_record)
      @current_user_record = T.let(user_session_record.user_record, T.nilable(UserRecord))
      store_user_session_token(token: user_session_record.token)

      true
    end

    sig(:final) { returns(T::Boolean) }
    def sign_out
      return true unless user_session_token

      UserSessionService::Destroy.new.call(user_session_token: user_session_token.not_nil!)
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

      if signed_in?
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

    sig(:final) { returns(T.nilable(UserRecord)) }
    def current_user_record
      @current_user_record
    end

    sig(:final) { returns(UserRecord) }
    def current_user_record!
      current_user_record.not_nil!
    end

    sig(:final) { returns(T::Boolean) }
    def signed_in?
      !current_user_record.nil?
    end

    sig(:final) { returns(T.nilable(User)) }
    def current_user
      return if current_user_record.nil?

      current_user!
    end

    sig(:final) { returns(User) }
    def current_user!
      UserRepository.new.to_model(user_record: current_user_record!)
    end

    sig(:final) { returns(T.nilable(String)) }
    private def user_session_token
      cookies.signed[UserSession::TOKENS_COOKIE_KEY]
    end

    sig(:final) { returns(T::Boolean) }
    private def restore_user_session
      user_session_record = UserSessionRecord.find_by(token: user_session_token)

      if user_session_record
        return sign_in(user_session_record)
      end

      false
    end

    sig(:final) { void }
    private def request_authentication
      sign_out
      session[:return_to_after_authenticating] = request.url
      redirect_to sign_in_path
    end

    sig(:final) { params(token: String).void }
    private def store_user_session_token(token:)
      domain = (Rails.env.test? && !ENV["CI"]) ? `hostname`.strip.downcase : ".#{Wikino.config.host}"

      cookies.signed.permanent[UserSession::TOKENS_COOKIE_KEY] = {
        value: token,
        httponly: true,
        same_site: :lax,
        domain:
      }
    end
  end
end
