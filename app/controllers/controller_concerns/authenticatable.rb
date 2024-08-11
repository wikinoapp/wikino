# typed: strict
# frozen_string_literal: true

module ControllerConcerns
  module Authenticatable
    extend T::Sig
    extend ActiveSupport::Concern

    included do
      helper_method :signed_in?, :viewer, :viewer!
    end

    sig(:final) { params(session: Session).returns(T::Boolean) }
    def sign_in(session)
      cookies.signed.permanent[Session::COOKIE_KEY] = {
        value: session.token,
        httponly: true,
        same_site: :lax,
        domain: ".#{ENV.fetch("NONOTO_HOST")}"
      }

      true
    end

    sig(:final) { returns(T::Boolean) }
    def sign_out
      cookies.delete(Session::COOKIE_KEY)

      true
    end

    sig(:final) { returns(T.nilable(User)) }
    def viewer
      @viewer ||= T.let(begin
        return unless cookies.signed[Session::COOKIE_KEY]
        Session.find_by(token: cookies.signed[Session::COOKIE_KEY])&.user
      end, T.nilable(User))
    end

    sig(:final) { returns(User) }
    def viewer!
      viewer.not_nil!
    end

    sig(:final) { returns(T::Boolean) }
    def signed_in?
      !viewer.nil?
    end

    sig(:final) { returns(T.untyped) }
    def require_authentication
      unless signed_in?
        redirect_to root_path
      end
    end

    sig(:final) { returns(T.untyped) }
    def require_no_authentication
      if signed_in?
        flash[:notice] = t("messages.authentication.already_signed_in")
        redirect_to home_path
      end
    end

    sig(:final) { returns(T.nilable(String)) }
    def original_remote_ip
      request.env["HTTP_CF_CONNECTING_IP"] || request.remote_ip
    end
  end
end
