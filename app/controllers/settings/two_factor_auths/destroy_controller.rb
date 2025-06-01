# typed: true
# frozen_string_literal: true

module Settings
  module TwoFactorAuths
    class DestroyController < ApplicationController
      include ControllerConcerns::Authenticatable
      include ControllerConcerns::Localizable

      around_action :set_locale
      before_action :require_authentication
      before_action :ensure_two_factor_auth_enabled

      sig { returns(T.untyped) }
      def call
        form = TwoFactorAuthForm::Destruction.new(form_params.to_h)

        if form.invalid?
          user_two_factor_auth = UserTwoFactorAuthRepository.new.find_by_user(user_record: current_user_record!)

          return render_component(
            Settings::TwoFactorAuths::ShowView.new(
              current_user: current_user!,
              user_two_factor_auth:,
              destruction_form: form
            ),
            status: :unprocessable_entity
          )
        end

        result = TwoFactorAuthService::Disable.new.call(
          user: current_user!,
          password: form.password.not_nil!
        )

        if result.success
          flash[:notice] = t("messages.two_factor_auth.disabled_successfully")
          redirect_to settings_two_factor_auth_path
        else
          flash.now[:alert] = result.error_message
          user_two_factor_auth = UserTwoFactorAuthRepository.new.find_by_user(user_record: current_user_record!)

          render_component(
            Settings::TwoFactorAuths::ShowView.new(
              current_user: current_user!,
              user_two_factor_auth:,
              destruction_form: form
            ),
            status: :unprocessable_entity
          )
        end
      end

      private

      sig { returns(ActionController::Parameters) }
      def form_params
        params.require(:two_factor_auth_form_destruction).permit(
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
