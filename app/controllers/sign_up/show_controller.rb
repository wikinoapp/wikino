# typed: true
# frozen_string_literal: true

module SignUp
  class ShowController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable

    around_action :set_locale
    before_action :require_no_authentication

    sig { returns(T.untyped) }
    def call
      form = EmailConfirmationForm::Creation.new

      render SignUp::ShowView.new(form:)
    end
  end
end
