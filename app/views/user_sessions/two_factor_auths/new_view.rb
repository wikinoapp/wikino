# typed: strict
# frozen_string_literal: true

module UserSessions
  module TwoFactorAuths
    class NewView < ApplicationView
      sig { params(form: UserSessionForm::TwoFactorVerification).void }
      def initialize(form:)
        @form = form
      end

      sig { returns(UserSessionForm::TwoFactorVerification) }
      attr_reader :form

      sig { returns(String) }
      def title
        t("meta.title.user_sessions.two_factor_auth")
      end

      sig { returns(PageName) }
      def current_page_name
        PageName::SignIn
      end
    end
  end
end
