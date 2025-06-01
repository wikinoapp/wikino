# typed: strict
# frozen_string_literal: true

module Settings
  module TwoFactorAuths
    class NewView < ApplicationView
      sig { params(current_user: User, setup_result: TwoFactorAuthService::Setup::SetupResult, form: TwoFactorAuthForm::Creation).void }
      def initialize(current_user:, setup_result:, form:)
        @current_user = current_user
        @setup_result = setup_result
        @form = form
      end

      sig { returns(User) }
      attr_reader :current_user

      sig { returns(TwoFactorAuthService::Setup::SetupResult) }
      attr_reader :setup_result

      sig { returns(TwoFactorAuthForm::Creation) }
      attr_reader :form

      sig { returns(String) }
      def title
        t("meta.title.settings.two_factor_auth.new")
      end

      sig { returns(Symbol) }
      def current_page_name
        :settings
      end
    end
  end
end
