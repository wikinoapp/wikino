# typed: true
# frozen_string_literal: true

module SignIn
  module TwoFactors
    module Recoveries
      class CreateController < ApplicationController
        include ControllerConcerns::Authenticatable
        include ControllerConcerns::Localizable

        around_action :set_locale
        before_action :require_no_authentication

        sig { returns(T.untyped) }
        def call
          pending_user_record = UserRecord.visible.find_by(id: session[:pending_user_id])

          unless pending_user_record&.two_factor_enabled?
            return redirect_to(sign_in_path)
          end

          form = UserSessions::TwoFactorRecoveryForm.new(
            form_params.merge(user_record: pending_user_record)
          )

          if form.invalid?
            return render_component(
              SignIn::TwoFactors::Recoveries::NewView.new(form:),
              status: :unprocessable_entity
            )
          end

          result = UserSessions::CreateWithRecoveryCodeService.new.call(
            user_two_factor_auth_record: pending_user_record.user_two_factor_auth_record.not_nil!,
            recovery_code: form.recovery_code.not_nil!,
            ip_address: original_remote_ip,
            user_agent: request.user_agent
          )

          session.delete(:pending_user_id)
          sign_in(result.user_session_record)

          flash[:notice] = t("messages.accounts.signed_in_successfully")
          redirect_to after_authentication_url
        end

        sig { returns(ActionController::Parameters) }
        private def form_params
          params.require(:user_sessions_two_factor_recovery_form).permit(
            :recovery_code
          )
        end
      end
    end
  end
end
