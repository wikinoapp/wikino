# typed: strict
# frozen_string_literal: true

module EmailConfirmations
  class EditView < ApplicationView
    sig { params(form: EmailConfirmationForm::Check).void }
    def initialize(form:)
      @form = form
    end

    sig { override.void }
    def before_render
      title = I18n.t("meta.title.email_confirmations.edit")
      helpers.set_meta_tags(title:, **default_meta_tags)
    end

    sig { returns(EmailConfirmationForm::Check) }
    attr_reader :form
    private :form

    sig { returns(PageName) }
    private def current_page_name
      PageName::EmailConfirmationEdit
    end
  end
end
