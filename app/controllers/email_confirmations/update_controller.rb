# typed: true
# frozen_string_literal: true

module EmailConfirmations
  class UpdateController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::EmailConfirmationFindable

    around_action :set_locale
    before_action :require_email_confirmation_id

    sig { returns(T.untyped) }
    def call
      form = EmailConfirmationForm.new(
        form_params.merge(email_confirmation_id: session[:email_confirmation_id])
      )

      if form.invalid?
        return render(EmailConfirmations::EditView.new(form:), status: :unprocessable_entity)
      end

      result = ConfirmEmailUseCase.new.call(email_confirmation: form.email_confirmation!)

      flash_message(result.email_confirmation)
      redirect_to success_path(result.email_confirmation)
    end

    sig { returns(ActionController::Parameters) }
    private def form_params
      T.cast(
        params.require(:email_confirmation_form), ActionController::Parameters
      ).permit(:confirmation_code)
    end

    sig { params(email_confirmation: EmailConfirmation).void }
    private def flash_message(email_confirmation)
      if email_confirmation.event_email_update?
        flash[:notice] = t("messages.email_confirmations.email_updated")
      end

      nil
    end

    sig { params(email_confirmation: EmailConfirmation).returns(String) }
    private def success_path(email_confirmation)
      if email_confirmation.event_sign_up?
        new_account_path
      else
        root_path
      end
    end
  end
end
