# typed: true
# frozen_string_literal: true

module Settings
  module TwoFactorAuths
    class CreateController < ApplicationController
      include ControllerConcerns::Authenticatable
      include ControllerConcerns::Localizable

      around_action :set_locale
      before_action :require_authentication
      before_action :ensure_two_factor_auth_not_enabled

      sig { returns(T.untyped) }
      def call
        form = ::TwoFactorAuths::CreationForm.new(
          form_params.merge(user_record: current_user_record!)
        )

        if form.invalid?
          setup_result = ::TwoFactorAuths::SetupService.new.call(user: current_user!)

          return render_component(
            Settings::TwoFactorAuths::NewView.new(
              current_user: current_user!,
              secret: setup_result.secret,
              provisioning_uri: setup_result.provisioning_uri,
              qr_code: setup_result.qr_code,
              form:
            ),
            status: :unprocessable_entity
          )
        end

        ::TwoFactorAuths::EnableService.new.call(
          user_record: form.user_record.not_nil!,
          password: form.password.not_nil!,
          totp_code: form.totp_code.not_nil!
        )

        flash[:notice] = t("messages.two_factor_auth.enabled_successfully")
        redirect_to settings_two_factor_auth_recovery_codes_path
      end

      sig { returns(ActionController::Parameters) }
      private def form_params
        params.require(:two_factor_auths_creation_form).permit(
          :password,
          :totp_code
        )
      end

      sig { void }
      private def ensure_two_factor_auth_not_enabled
        user_two_factor_auth = UserTwoFactorAuthRepository.new.find_by_user(
          user_record: current_user_record!
        )

        if user_two_factor_auth&.enabled
          flash[:alert] = t("messages.two_factor_auth.already_enabled")
          redirect_to settings_two_factor_auth_path
        end
      end
    end
  end
end
