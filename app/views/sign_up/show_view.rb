# typed: strict
# frozen_string_literal: true

module SignUp
  class ShowView < ApplicationView
    use_helpers :set_meta_tags

    sig { params(form: NewEmailConfirmationForm).void }
    def initialize(form:)
      @form = form
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
