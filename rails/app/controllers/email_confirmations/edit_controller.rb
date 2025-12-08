# typed: true
# frozen_string_literal: true

module EmailConfirmations
  class EditController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::EmailConfirmationFindable

    around_action :set_locale
    before_action :require_email_confirmation_id

    sig { returns(T.untyped) }
    def call
      form = EmailConfirmations::CheckForm.new

      render_component EmailConfirmations::EditView.new(form:)
    end
  end
end
