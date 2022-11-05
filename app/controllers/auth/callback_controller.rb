# typed: true
# frozen_string_literal: true

class Auth::CallbackController < ActionController::Base
  extend T::Sig

  include Authenticatable

  def call
    auth_info = request.env["omniauth.auth"]
    authentication = Authentication.new(auth0_user_id: auth_info.uid)

    if authentication.invalid?
      flash[:alert] = t("messages.auth.callback.sign_in_failed")
      return redirect_to(root_path)
    end

    ActiveRecord::Base.transaction do
      user = authentication.find_or_create_user!
      sign_in user
    end

    redirect_to root_path
  end
end
