# typed: strict
# frozen_string_literal: true

module PasswordResets
  class NewController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable

    around_action :set_locale
    before_action :require_no_authentication

    sig { returns(T.untyped) }
    def call
      form = EmailConfirmations::CreationForm.new

      render_component PasswordResets::NewView.new(form:)
    end
  end
end
