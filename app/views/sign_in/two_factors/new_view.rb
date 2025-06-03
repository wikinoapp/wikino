# typed: strict
# frozen_string_literal: true

module SignIn
  module TwoFactors
    class NewView < ApplicationView
      sig { params(form: UserSessionForm::TwoFactorVerification).void }
      def initialize(form:)
        @form = form
      end

      sig { returns(UserSessionForm::TwoFactorVerification) }
      attr_reader :form
      private :form

      sig { returns(String) }
      private def title
        t("meta.title.sign_in.two_factors.new")
      end

      sig { returns(PageName) }
      private def current_page_name
        PageName::SignInTwoFactorNew
      end
    end
  end
end
