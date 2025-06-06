# typed: strict
# frozen_string_literal: true

module Settings
  module TwoFactorAuths
    module RecoveryCodes
      class ShowView < ApplicationView
        sig do
          params(
            current_user: User,
            user_two_factor_auth: UserTwoFactorAuth,
            form: TwoFactorAuthForm::RecoveryCodeRegeneration
          ).void
        end
        def initialize(
          current_user:,
          user_two_factor_auth:,
          form:
        )
          @current_user = current_user
          @user_two_factor_auth = user_two_factor_auth
          @form = form
        end

        sig { override.void }
        def before_render
          helpers.set_meta_tags(title:, **default_meta_tags)
        end

        sig { returns(User) }
        attr_reader :current_user
        private :current_user

        sig { returns(UserTwoFactorAuth) }
        attr_reader :user_two_factor_auth
        private :user_two_factor_auth

        sig { returns(TwoFactorAuthForm::RecoveryCodeRegeneration) }
        attr_reader :form
        private :form

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
