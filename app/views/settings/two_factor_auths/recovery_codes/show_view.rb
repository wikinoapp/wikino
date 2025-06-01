# typed: strict
# frozen_string_literal: true

module Settings
  module TwoFactorAuths
    module RecoveryCodes
      class ShowView < ApplicationView
        sig { params(current_user: User, user_two_factor_auth: UserTwoFactorAuth, recovery_codes: T.nilable(T::Array[String]), show_download: T::Boolean, regeneration_form: T.nilable(TwoFactorAuthForm::RecoveryCodeRegeneration)).void }
        def initialize(current_user:, user_two_factor_auth:, recovery_codes: nil, show_download: false, regeneration_form: nil)
          @current_user = current_user
          @user_two_factor_auth = user_two_factor_auth
          @recovery_codes = recovery_codes
          @show_download = show_download
          @regeneration_form = T.let(regeneration_form || TwoFactorAuthForm::RecoveryCodeRegeneration.new, TwoFactorAuthForm::RecoveryCodeRegeneration)
        end

        sig { returns(User) }
        attr_reader :current_user
        private :current_user

        sig { returns(UserTwoFactorAuth) }
        attr_reader :user_two_factor_auth
        private :user_two_factor_auth

        sig { returns(T.nilable(T::Array[String])) }
        attr_reader :recovery_codes
        private :recovery_codes

        sig { returns(T::Boolean) }
        attr_reader :show_download
        private :show_download

        sig { returns(TwoFactorAuthForm::RecoveryCodeRegeneration) }
        attr_reader :regeneration_form
        private :regeneration_form

        sig { returns(String) }
        private def title
          t("meta.title.settings.two_factor_auth.recovery_codes")
        end

        sig { returns(PageName) }
        private def current_page_name
          PageName::SettingsTwoFactorAuthRecoveryCodes
        end
      end
    end
  end
end
