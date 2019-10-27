# frozen_string_literal: true

class OauthCallbacksController < ApplicationController
  def show
    oauth_auth = request.env["omniauth.auth"]

    user = User.without_deleted.find_by(email: oauth_auth.dig("info", "email"))
    user ||= User.sign_up_with_google!(oauth_auth: oauth_auth, oauth_params: request.env["omniauth.params"])

    session[:user_id] = user.id

    redirect_to root_path
  end
end
