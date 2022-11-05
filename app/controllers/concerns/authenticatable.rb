# typed: strict
# frozen_string_literal: true

module Authenticatable
  extend T::Sig
  extend ActiveSupport::Concern

  included do
    helper_method :current_user, :sign_in_with_auth0_path, :sign_up_with_auth0_path
  end

  sig { returns(String) }
  def sign_in_with_auth0_path
    "/auth/auth0"
  end

  sig { returns(String) }
  def sign_up_with_auth0_path
    # ユーザ登録ページに直接遷移するために `screen_hint` パラメータを付与する
    # https://github.com/auth0/omniauth-auth0/pull/103
    "/auth/auth0?screen_hint=signup"
  end

  sig { params(return_to: String).returns(String) }
  def sign_out_with_auth0_url(return_to:)
    # NOTE: snake_caseとcamelCaseが混在しているが、Auth0がこのパラメータを受け付けている
    #   https://auth0.com/docs/api/authentication#logout
    query = {client_id: ENV.fetch("NONOTO_AUTH0_CLIENT_ID"), returnTo: return_to}.to_query

    "#{ENV.fetch("NONOTO_AUTH0_LOGOUT_URL")}?#{query}"
  end

  sig { params(user: User).returns(String) }
  def sign_in(user)
    user.track_sign_in!
    session[:user_id] = user.id
  end

  sig { returns(T.untyped) }
  def sign_out
    reset_session
  end

  sig { returns(T.nilable(User)) }
  def current_user
    return unless session[:user_id]

    @current_user ||= T.let(User.only_kept.find_by(id: session[:user_id]), T.nilable(User))
  end

  sig { returns(T::Boolean) }
  def user_signed_in?
    current_user.present?
  end

  sig { returns(T.untyped) }
  def authenticate_user
    unless user_signed_in?
      redirect_to root_path
    end
  end

  sig { returns(T.untyped) }
  def require_no_authentication
    if user_signed_in?
      flash[:notice] = t("messages.authentication.already_signed_in")
      redirect_to root_path
    end
  end
end
