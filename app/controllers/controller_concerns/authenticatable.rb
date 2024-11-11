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
      Current.user = session.user

      space_identifier = session.space.not_nil!.identifier
      tokens = session_tokens || {}
      # ハッシュの先頭に追加する
      tokens = tokens.except(space_identifier).reverse_merge(space_identifier => session.token)
      store_session_tokens_to_cookie(token_str: hash_to_string(tokens))

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
      restore_session || request_authentication
    end

    sig(:final) { returns(T.untyped) }
    def require_no_authentication
      return if params[:skip_no_authentication].present?

      restore_session

      if signed_in?
        flash[:notice] = t("messages.authentication.already_signed_in")
        redirect_to space_path(Current.user!.space.not_nil!.identifier)
      end
    end

    sig(:final) { returns(String) }
    def after_authentication_url
      session.delete(:return_to_after_authenticating) ||
        space_url(Current.user!.space.not_nil!.identifier)
    end

    sig(:final) { returns(T.nilable(String)) }
    def original_remote_ip
      request.env["HTTP_CF_CONNECTING_IP"] || request.remote_ip
    end

    sig(:final) { returns(T.nilable(T::Hash[String, String])) }
    private def session_tokens
      token_str = cookies.signed[Session::TOKENS_COOKIE_KEY]
      return unless token_str

      string_to_hash(token_str)
    end

    sig(:final) { returns(T::Array[String]) }
    private def cookie_user_ids
      return [] unless session_tokens

      Session.where(token: session_tokens.not_nil!.values).pluck(:user_id)
    end

    sig(:final) { returns(T::Boolean) }
    private def restore_session
      return false unless session_tokens

      token = if Current.space
        session_tokens.not_nil![Current.space!.identifier]
      else
        session_tokens.not_nil!.values.first
      end
      session = Session.find_by(token:)
      return false unless session

      sign_in(session)

      true
    end

    sig(:final) { void }
    private def request_authentication
      sign_out
      session[:return_to_after_authenticating] = request.url
      redirect_to sign_in_path
    end

    sig(:final) { params(token_str: String).void }
    private def store_session_tokens_to_cookie(token_str:)
      cookies.signed.permanent[Session::TOKENS_COOKIE_KEY] = {
        value: token_str,
        httponly: true,
        same_site: :lax,
        domain: ".#{Wikino.config.host}"
      }
    end

    # 例: "a:1|b:2" => { "a" => "1", "b" => "2" }
    sig { params(str: String).returns(T::Hash[String, String]) }
    private def string_to_hash(str)
      str.split("|").map { |pair| pair.split(":") }.to_h { |k, v| [k.not_nil!, v.not_nil!] }
    end

    # 例: { "a" => "1", "b" => "2" } => "a:1|b:2"
    sig { params(hash: T::Hash[String, String]).returns(String) }
    def hash_to_string(hash)
      hash.map { |k, v| "#{k}:#{v}" }.join("|")
    end
  end
end
