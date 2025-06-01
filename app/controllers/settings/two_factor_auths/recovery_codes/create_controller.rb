# typed: true
# frozen_string_literal: true

module Settings
  module TwoFactorAuths
    module RecoveryCodes
      class CreateController < ApplicationController
        include ControllerConcerns::Authenticatable
        include ControllerConcerns::Localizable

        around_action :set_locale
        before_action :require_authentication
        before_action :ensure_two_factor_auth_enabled

        sig { returns(T.untyped) }
        def call
          form = TwoFactorAuthForm::RecoveryCodeRegeneration.new(form_params.to_h)

          if form.invalid?
            user_two_factor_auth = UserTwoFactorAuthRepository.new.find_by_user(user_record: current_user_record!)

            return render_component(
              Settings::TwoFactorAuths::RecoveryCodes::ShowView.new(
                current_user: current_user!,
                user_two_factor_auth: user_two_factor_auth.not_nil!,
                regeneration_form: form
              ),
              status: :unprocessable_entity
            )
          end

          result = TwoFactorAuthService::RegenerateRecoveryCodes.new.call(
            user: current_user!,
            password: form.password.not_nil!
          )

          if result.success
            # 新しいリカバリーコードをセッションに保存して表示
            session[:recovery_codes] = result.recovery_codes
            flash[:notice] = t("messages.two_factor_auth.recovery_codes_regenerated")
            redirect_to settings_two_factor_auth_recovery_codes_path
          else
            flash.now[:alert] = result.error_message
            user_two_factor_auth = UserTwoFactorAuthRepository.new.find_by_user(user_record: current_user_record!)

            render_component(
              Settings::TwoFactorAuths::RecoveryCodes::ShowView.new(
                current_user: current_user!,
                user_two_factor_auth: user_two_factor_auth.not_nil!,
                regeneration_form: form
              ),
              status: :unprocessable_entity
            )
          end
        end

        private

        sig { returns(ActionController::Parameters) }
        def form_params
          params.require(:two_factor_auth_form_recovery_code_regeneration).permit(
            :password
          )
        end

        sig { void }
        def ensure_two_factor_auth_enabled
          user_two_factor_auth = UserTwoFactorAuthRepository.new.find_by_user(user_record: current_user_record!)

          unless user_two_factor_auth&.enabled
            flash[:alert] = t("messages.two_factor_auth.not_enabled")
            redirect_to settings_two_factor_auth_path
          end
        end
      end
    end
  end
end
