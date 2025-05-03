# typed: true
# frozen_string_literal: true

module Settings
  module Emails
    class UpdateController < ApplicationController
      include ControllerConcerns::Authenticatable
      include ControllerConcerns::Localizable

      around_action :set_locale
      before_action :require_authentication

      sig { returns(T.untyped) }
      def call
        form = EmailForm::Edit.new(form_params)

        if form.invalid?
          return render(
            Settings::Emails::ShowView.new(
              current_user: current_user!,
              form:
            ),
            status: :unprocessable_entity
          )
        end

        result = EmailConfirmationService::Create.new.call(
          email: form.new_email.not_nil!,
          event: EmailConfirmationEvent::EmailUpdate,
          locale: current_locale
        )

        session[:email_confirmation_id] = result.email_confirmation.id
        flash[:notice] = t("messages.email_confirmations.confirmation_mail_sent")
        redirect_to edit_email_confirmation_path
      end

      sig { returns(ActionController::Parameters) }
      private def form_params
        params.require(:email_form_edit).permit(
          :new_email
        )
      end
    end
  end
end
