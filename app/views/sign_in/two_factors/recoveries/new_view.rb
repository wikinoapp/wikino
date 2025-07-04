# typed: strict
# frozen_string_literal: true

module SignIn
  module TwoFactors
    module Recoveries
      class NewView < ApplicationView
        sig { params(form: UserSessionForm::TwoFactorRecovery).void }
        def initialize(form:)
          @form = form
        end

        sig { override.void }
        def before_render
          helpers.set_meta_tags(title:, **default_meta_tags)
        end

        sig { returns(UserSessionForm::TwoFactorRecovery) }
        attr_reader :form
        private :form

        sig { returns(String) }
        private def title
          t("meta.title.sign_in.two_factors.recoveries.show")
        end

        sig { returns(PageName) }
        private def current_page_name
          PageName::SignInTwoFactorRecovery
        end
      end
    end
  end
end
