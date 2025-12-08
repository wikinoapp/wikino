# typed: strict
# frozen_string_literal: true

module EmailConfirmations
  class UpdateController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::EmailConfirmationFindable

    around_action :set_locale
    before_action :require_email_confirmation_id
    before_action :restore_user_session

    sig { returns(T.untyped) }
    def call
      form = EmailConfirmations::CheckForm.new(
        form_params.merge(email_confirmation_id: session[:email_confirmation_id])
      )

      if form.invalid?
        return render_component(EmailConfirmations::EditView.new(form:), status: :unprocessable_entity)
      end

      result = Emails::ConfirmService.new.call(
        email_confirmation_record: form.email_confirmation_record!,
        user_record: current_user_record
      )

      flash_message(result.email_confirmation_record)
      redirect_to success_path(result.email_confirmation_record)
    end

    sig { returns(ActionController::Parameters) }
    private def form_params
      T.cast(
        params.require(:email_confirmations_check_form), ActionController::Parameters
      ).permit(:confirmation_code)
    end

    sig { params(email_confirmation: EmailConfirmationRecord).void }
    private def flash_message(email_confirmation)
      if email_confirmation.event_email_update?
        flash[:notice] = t("messages.email_confirmations.email_updated")
      end

      nil
    end

    sig { params(email_confirmation: EmailConfirmationRecord).returns(String) }
    private def success_path(email_confirmation)
      case email_confirmation.deserialized_event
      when EmailConfirmationEvent::SignUp
        new_account_path
      when EmailConfirmationEvent::EmailUpdate
        settings_email_path
      when EmailConfirmationEvent::PasswordReset
        edit_password_path
      end
    end
  end
end
