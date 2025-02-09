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
      form = UserSessionForm.new

      render SignIn::ShowView.new(form:)
    end
  end
end
