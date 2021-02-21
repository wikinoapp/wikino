# frozen_string_literal: true

module SignInTokenAuthenticatable
  extend ActiveSupport::Concern

  included do
    helper_method :current_user, :signed_in?
  end

  def signed_in?
    !current_user.nil?
  end

  def current_user
    @current_user ||= warden.user
  end

  def warden
    @warden ||= request.env["warden"]
  end

  def authenticate!
    warden.authenticate!
  end
end
