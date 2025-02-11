# typed: strict
# frozen_string_literal: true

module EmailConfirmations
  class EditView < ApplicationView
    use_helpers :set_meta_tags

    sig { params(form: EmailConfirmationForm).void }
    def initialize(form:)
      @form = form
    end

    sig { returns(EmailConfirmationForm) }
    attr_reader :form
    private :form

    sig { returns(PageName) }
    private def current_page_name
      PageName::EmailConfirmationEdit
    end
  end
end
