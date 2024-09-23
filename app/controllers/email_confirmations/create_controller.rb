# typed: true
# frozen_string_literal: true

module EmailConfirmations
  class CreateController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable

    around_action :set_locale
    before_action :require_no_authentication

    sig { returns(T.untyped) }
    def call
      @form = NewEmailConfirmationForm.new(form_params)

      if @form.invalid?
        return render("sign_up/new/call", status: :unprocessable_entity)
      end

      result = CreateEmailConfirmationUseCase.new.call(
        email: @form.email.not_nil!,
        event: EmailConfirmationEvent::SignUp,
        locale: current_locale
      )

      session[:email_confirmation_id] = result.email_confirmation.id
      flash[:notice] = t("messages.email_confirmations.confirmation_mail_sent")
      redirect_to edit_email_confirmation_path
    end

    sig { returns(ActionController::Parameters) }
    private def form_params
      T.cast(params.require(:new_email_confirmation_form), ActionController::Parameters).permit(:email)
    end
  end
end
