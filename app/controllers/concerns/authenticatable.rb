# typed: strict
# frozen_string_literal: true

module Authenticatable
  extend T::Sig
  extend ActiveSupport::Concern

  included do
    helper_method :sign_in_with_auth0_path, :sign_up_with_auth0_path
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

  sig { params(user: User).returns(String) }
  def sign_in(user)
    user.track_sign_in!
    session[:user_id] = user.id
  end
end
