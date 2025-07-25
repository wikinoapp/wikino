# typed: strict
# frozen_string_literal: true

module Settings
  module TwoFactorAuths
    module RecoveryCodes
      class ShowController < ApplicationController
        include ControllerConcerns::Authenticatable
        include ControllerConcerns::Localizable

        around_action :set_locale
        before_action :require_authentication

        sig { returns(T.untyped) }
        def call
          user_two_factor_auth = UserTwoFactorAuthRepository.new.find_by_user(
            user_record: current_user_record!
          )

          unless user_two_factor_auth&.enabled
            flash[:alert] = t("messages.two_factor_auth.not_enabled")
            return redirect_to(settings_two_factor_auth_path)
          end

          form = TwoFactorAuths::RecoveryCodeRegenerationForm.new

          render_component Settings::TwoFactorAuths::RecoveryCodes::ShowView.new(
            current_user: current_user!,
            user_two_factor_auth: user_two_factor_auth.not_nil!,
            form:
          )
        end
      end
    end
  end
end
