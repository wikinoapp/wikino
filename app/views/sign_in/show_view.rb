# typed: strict
# frozen_string_literal: true

module SignIn
  class ShowView < ApplicationView
    sig { params(form: UserSessionForm).void }
    def initialize(form:)
      @form = form
    end

    sig { override.void }
    def before_render
      title = I18n.t("meta.title.sign_in.show")
      helpers.set_meta_tags(title:, **default_meta_tags)
    end

    sig { returns(UserSessionForm) }
    attr_reader :form
    private :form

    sig { returns(PageName) }
    private def current_page_name
      PageName::SignIn
    end
  end
end
