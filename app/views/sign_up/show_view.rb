# typed: strict
# frozen_string_literal: true

module SignUp
  class ShowView < ApplicationView
    sig { params(form: EmailConfirmationForm::Creation).void }
    def initialize(form:)
      @form = form
    end

    sig { override.void }
    def before_render
      helpers.set_meta_tags(title:, **default_meta_tags)
    end

    sig { returns(EmailConfirmationForm::Creation) }
    attr_reader :form
    private :form

    sig { returns(String) }
    private def title
      I18n.t("meta.title.sign_up.show")
    end

    sig { returns(PageName) }
    private def current_page_name
      PageName::SignUp
    end
  end
end
