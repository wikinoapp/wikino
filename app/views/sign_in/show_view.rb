# typed: strict
# frozen_string_literal: true

module SignIn
  class ShowView < ApplicationView
    use_helpers :set_meta_tags

    sig { params(form: UserSessionForm).void }
    def initialize(form:)
      @form = form
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
