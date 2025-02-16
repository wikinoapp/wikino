# typed: strict
# frozen_string_literal: true

module SignUp
  class ShowView < ApplicationView
    sig { params(form: NewEmailConfirmationForm).void }
    def initialize(form:)
      @form = form
    end

    def before_render
      title = I18n.t("meta.title.sign_up.show")
      helpers.set_meta_tags(title:, **default_meta_tags)
    end

    sig { returns(NewEmailConfirmationForm) }
    attr_reader :form
    private :form

    sig { returns(PageName) }
    private def current_page_name
      PageName::SignUp
    end
  end
end
