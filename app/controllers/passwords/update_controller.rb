# typed: strict
# frozen_string_literal: true

module Passwords
  class UpdateController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::EmailConfirmationFindable

    around_action :set_locale
    before_action :require_no_authentication
    before_action :require_succeeded_email_confirmation

    sig { returns(T.untyped) }
    def call
      form = PasswordResetForm::Creation.new(form_params)

      if form.invalid?
        return render_component(
          Passwords::EditView.new(form:),
          status: :unprocessable_entity
        )
      end

      PasswordService::Update.new.call(
        email: @email_confirmation.not_nil!.email.not_nil!,
        password: form.password.not_nil!
      )

      reset_session

      flash[:notice] = t("messages.passwords.reset_successfully_html")
      redirect_to sign_in_path
    end

    sig { returns(ActionController::Parameters) }
    private def form_params
      T.cast(params.require(:password_reset_form_creation), ActionController::Parameters)
        .permit(:password)
    end
  end
end
