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
        form = TwoFactorAuthForm::Creation.new(
          form_params.merge(user_record: current_user_record!)
        )

        if form.invalid?
          setup_result = TwoFactorAuthService::Setup.new.call(user: current_user!)

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

        result = TwoFactorAuthService::Enable.new.call(
          user: current_user!,
          password: form.password.not_nil!,
          totp_code: form.totp_code.not_nil!
        )

        if result.success
          # Store recovery codes in session temporarily to show them once
          session[:recovery_codes] = result.recovery_codes
          flash[:notice] = t("messages.two_factor_auth.enabled_successfully")
          redirect_to settings_two_factor_auth_recovery_codes_path
        else
          flash.now[:alert] = result.error_message
          setup_result = TwoFactorAuthService::Setup.new.call(user: current_user!)

          render_component(
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
      end

      sig { returns(ActionController::Parameters) }
      private def form_params
        params.require(:two_factor_auth_form_creation).permit(
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
