# typed: strict
# frozen_string_literal: true

module Settings
  module TwoFactorAuths
    class NewController < ApplicationController
      include ControllerConcerns::Authenticatable
      include ControllerConcerns::Localizable

      around_action :set_locale
      before_action :require_authentication
      before_action :ensure_two_factor_auth_not_enabled

      sig { returns(T.untyped) }
      def call
        result = ::TwoFactorAuths::SetupService.new.call(user: current_user!)

        render_component Settings::TwoFactorAuths::NewView.new(
          current_user: current_user!,
          secret: result.secret,
          provisioning_uri: result.provisioning_uri,
          qr_code: result.qr_code,
          form: ::TwoFactorAuths::CreationForm.new
        )
      end

      sig { void }
      private def ensure_two_factor_auth_not_enabled
        user_two_factor_auth = UserTwoFactorAuthRepository.new.find_by_user(user_record: current_user_record!)

        if user_two_factor_auth&.enabled
          flash[:alert] = t("messages.two_factor_auth.already_enabled")
          redirect_to settings_two_factor_auth_path
        end
      end
    end
  end
end
