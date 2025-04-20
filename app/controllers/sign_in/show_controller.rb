# typed: true
# frozen_string_literal: true

module SignIn
  class ShowController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable

    around_action :set_locale
    before_action :require_no_authentication

    sig { returns(T.untyped) }
    def call
      user_session = UserSession.new

      render SignIn::ShowView.new(user_session:)
    end
  end
end
