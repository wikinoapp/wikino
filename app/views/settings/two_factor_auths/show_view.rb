# typed: strict
# frozen_string_literal: true

module Settings
  module TwoFactorAuths
    class ShowView < ApplicationView
      sig { params(current_user: User, user_two_factor_auth: T.nilable(UserTwoFactorAuth), destruction_form: T.nilable(TwoFactorAuthForm::Destruction)).void }
      def initialize(current_user:, user_two_factor_auth:, destruction_form: nil)
        @current_user = current_user
        @user_two_factor_auth = user_two_factor_auth
        @destruction_form = T.let(destruction_form || TwoFactorAuthForm::Destruction.new, TwoFactorAuthForm::Destruction)
      end

      sig { returns(User) }
      attr_reader :current_user
      private :current_user

      sig { returns(T.nilable(UserTwoFactorAuth)) }
      attr_reader :user_two_factor_auth
      private :user_two_factor_auth

      sig { returns(TwoFactorAuthForm::Destruction) }
      attr_reader :destruction_form
      private :destruction_form

      sig { returns(String) }
      private def title
        t("meta.title.settings.two_factor_auth.show")
      end

      sig { returns(Symbol) }
      private def current_page_name
        :settings
      end
    end
  end
end
