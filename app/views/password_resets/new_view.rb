# typed: strict
# frozen_string_literal: true

module PasswordResets
  class NewView < ApplicationView
    sig { params(form: EmailConfirmationForm::Creation).void }
    def initialize(form:)
      @form = form
    end

    sig { override.void }
    def before_render
      title = I18n.t("meta.title.password_resets.new")
      helpers.set_meta_tags(title:, **default_meta_tags)
    end

    sig { returns(EmailConfirmationForm::Creation) }
    attr_reader :form
    private :form

    sig { returns(PageName) }
    private def current_page_name
      PageName::PasswordReset
    end
  end
end
