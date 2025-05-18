# typed: strict
# frozen_string_literal: true

module PasswordResets
  class CreateController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable

    around_action :set_locale
    before_action :require_no_authentication

    sig { returns(T.untyped) }
    def call
      form = EmailConfirmationForm::Creation.new(form_params)

      if form.invalid?
        return render_component(
          PasswordResets::NewView.new(form:),
          status: :unprocessable_entity
        )
      end

      result = EmailConfirmationService::Create.new.call(
        email: form.email.not_nil!,
        event: EmailConfirmationEvent::PasswordReset,
        locale: current_locale
      )

      session[:email_confirmation_id] = result.email_confirmation.id
      flash[:notice] = t("messages.email_confirmations.confirmation_mail_sent")
      redirect_to edit_email_confirmation_path
    end

    sig { returns(ActionController::Parameters) }
    private def form_params
      T.cast(params.require(:email_confirmation_form_creation), ActionController::Parameters)
        .permit(:email)
    end
  end
end
