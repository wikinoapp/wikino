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
        before_action :ensure_two_factor_auth_enabled

        sig { returns(T.untyped) }
        def call
          user_two_factor_auth = UserTwoFactorAuthRepository.new.find_by_user(user_record: current_user_record!)
          
          # Check if we have recovery codes in session (just enabled 2FA)
          recovery_codes = session.delete(:recovery_codes)
          show_download = recovery_codes.present?

          render_component Settings::TwoFactorAuths::RecoveryCodes::ShowView.new(
            current_user: current_user!,
            user_two_factor_auth: user_two_factor_auth.not_nil!,
            recovery_codes: recovery_codes,
            show_download: show_download
          )
        end

        private

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

