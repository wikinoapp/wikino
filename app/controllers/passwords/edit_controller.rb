# typed: strict
# frozen_string_literal: true

module Passwords
  class EditController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::EmailConfirmationFindable

    around_action :set_locale
    before_action :require_no_authentication
    before_action :require_succeeded_email_confirmation

    sig { returns(T.untyped) }
    def call
      form = PasswordResetForm::Creation.new

      render_component Passwords::EditView.new(form:)
    end
  end
end
