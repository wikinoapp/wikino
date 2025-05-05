# typed: strict
# frozen_string_literal: true

module SignIn
  class ShowView < ApplicationView
    sig { params(form: UserSessionForm::Creation).void }
    def initialize(form:)
      @form = form
    end

    sig { override.void }
    def before_render
      helpers.set_meta_tags(title: current_page_title, **default_meta_tags)
    end

    sig { returns(UserSessionForm::Creation) }
    attr_reader :form
    private :form

    sig { returns(String) }
    private def current_page_title
      t("meta.title.sign_in.show")
    end

    sig { returns(PageName) }
    private def current_page_name
      PageName::SignIn
    end
  end
end
