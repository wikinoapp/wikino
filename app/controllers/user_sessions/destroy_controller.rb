# typed: true
# frozen_string_literal: true

module UserSessions
  class DestroyController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      sign_out

      redirect_to root_path
    end
  end
end
